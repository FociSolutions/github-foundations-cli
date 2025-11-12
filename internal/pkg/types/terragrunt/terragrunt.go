package terragrunt

import (
	"bytes"
	"fmt"
	"gh_foundations/internal/pkg/types"
	"gh_foundations/internal/pkg/types/status"
	"gh_foundations/internal/pkg/types/terraform_state"
	v1_2 "gh_foundations/internal/pkg/types/terraform_state/v1.2"
	"io"
	"log"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/tidwall/gjson"
	"github.com/zclconf/go-cty/cty"
)

var fs = afero.NewOsFs()

// Compile regex patterns once at package initialization for better performance
var (
	// Pattern to match: include "name" { path = find_in_parent_folders("filename") }
	includeRegex = regexp.MustCompile(`include\s+["']?\w+["']?\s*{\s*path\s*=\s*find_in_parent_folders\s*\(\s*["']([^"']+)["']\s*\)`)
)

// command creation function for mocking
var newCommandExecutor = func(name string, args ...string) types.ICommandExecutor {
	return &types.CommandExecutor{
		Cmd: exec.Command(name, args...),
	}
}

type IPlanFile interface {
	Cleanup() error
	RunPlan(target *string) error
	GetStateExplorer() (terraform_state.IStateExplorer, error)
	GetPlanFilePath() string
}

type PlanFile struct {
	Name           string
	ModulePath     string
	ModuleDir      string
	OutputFilePath string
}

type HCLFile struct {
	Path string
}

func getRepository(repo map[string]interface{}) (status.Repository, error) {
	var repository status.Repository
	if err := mapstructure.Decode(repo, &repository); err != nil {
		return repository, fmt.Errorf("getRepository decode error: %w", err)
	}
	return repository, nil
}


// Given a repository map, returned by Viper, return a map of status.Repository
// Accept both the historical []map[string]interface{} shape and a direct map[string]interface{}
func getRepositoryMap(raw interface{}) (map[string]status.Repository, error) {
	repos := make(map[string]status.Repository)

	// Case 1: []map[string]interface{} (legacy viper decoding)
	if list, ok := raw.([]map[string]interface{}); ok && len(list) > 0 {
		for name, r := range list[0] {
			switch typed := r.(type) {
			case []map[string]interface{}: // current nested slice form
				if len(typed) == 0 { continue }
				repo, err := getRepository(typed[0])
				if err != nil { return repos, err }
				repo.Name = name
				repos[name] = repo
			case map[string]interface{}: // simplified direct object
				repo, err := getRepository(typed)
				if err != nil { return repos, err }
				repo.Name = name
				repos[name] = repo
			default:
				log.Printf("getRepositoryMap: unsupported repository value type for %s: %T", name, r)
			}
		}
		return repos, nil
	}

	// Case 2: direct map[string]interface{}
	if direct, ok := raw.(map[string]interface{}); ok {
		for name, r := range direct {
			if typed, ok := r.(map[string]interface{}); ok {
				repo, err := getRepository(typed)
				if err != nil { return repos, err }
				repo.Name = name
				repos[name] = repo
			} else if typedList, ok := r.([]map[string]interface{}); ok && len(typedList) > 0 {
				repo, err := getRepository(typedList[0])
				if err != nil { return repos, err }
				repo.Name = name
				repos[name] = repo
			} else {
				log.Printf("getRepositoryMap: unsupported direct repository value type for %s: %T", name, r)
			}
		}
		return repos, nil
	}

	log.Printf("getRepositoryMap: unrecognized raw repository structure: %T", raw)
	return repos, nil
}

// Return the locals block from the HCL file as a slice of string slices
func getLocalsBlock(contents string) (string, [][]string) {
	// Very lightweight parser for a single 'locals { ... }' block.
	// We iterate line by line once we find the opening 'locals {' until the matching '}'.
	lines := strings.Split(contents, "\n")
	start := -1
	braceDepth := 0
	for i, line := range lines {
		if start == -1 && strings.HasPrefix(strings.TrimSpace(line), "locals") && strings.Contains(line, "{") {
			start = i
			braceDepth = strings.Count(line, "{") - strings.Count(line, "}")
			if braceDepth == 0 { // single line locals { }
				break
			}
			continue
		}
		if start != -1 {
			braceDepth += strings.Count(line, "{") - strings.Count(line, "}")
			if braceDepth == 0 { // end of block
				end := i
				localsBlock := strings.Join(lines[start:end+1], "\n")
				innerLines := lines[start+1 : end]
				matches := make([][]string, 0)
				currentKey := ""
				var currentValLines []string
				flush := func() {
					if currentKey != "" {
						val := strings.TrimSpace(strings.Join(currentValLines, "\n"))
						matches = append(matches, []string{currentKey, val})
						currentKey = ""
						currentValLines = nil
					}
				}
				for _, l := range innerLines {
					trimmed := strings.TrimSpace(l)
					if trimmed == "" {
						continue
					}
					// detect new key = value line
					if eq := strings.Index(trimmed, "="); eq > 0 {
						// heuristic: treat as new key if currentKey empty OR line starts with an identifier
						left := strings.TrimSpace(trimmed[:eq])
						if regexp.MustCompile(`^[A-Za-z0-9_]+$`).MatchString(left) {
							// flush previous
							flush()
							currentKey = left
							currentValLines = []string{strings.TrimSpace(trimmed[eq+1:])}
							continue
						}
					}
					// continuation of previous value
					if currentKey != "" {
						currentValLines = append(currentValLines, trimmed)
					}
				}
				flush()
				return localsBlock, matches
			}
		}
	}
	return "", make([][]string, 0)
}

// Check if the HCL file has an include block and return the included file path
func getIncludedFilePath(contents string, currentFilePath string) string {
	// Use pre-compiled regex for better performance
	matches := includeRegex.FindStringSubmatch(contents)

	if len(matches) > 1 {
		filename := matches[1]
		// Walk up the directory tree to find the file
		dir := path.Dir(currentFilePath)
		for {
			candidatePath := path.Join(dir, filename)
			if _, err := fs.Stat(candidatePath); err == nil {
				return candidatePath
			}
			parentDir := path.Dir(dir)
			if parentDir == dir {
				// Reached root, file not found
				break
			}
			dir = parentDir
		}
	}

	return ""
}

// Read inputs from a parent file referenced by include
// visitedFiles tracks files we've already processed to prevent circular includes
func getInputsFromIncludedFile(includedFilePath string, visitedFiles map[string]bool) (status.Inputs, error) {
	var inputs status.Inputs

	if includedFilePath == "" {
		return inputs, nil
	}

	// Check for circular includes
	if visitedFiles[includedFilePath] {
		log.Printf("Circular include detected for file: %s\n", includedFilePath)
		return inputs, fmt.Errorf("circular include detected for file: %s", includedFilePath)
	}

	// Mark this file as visited
	visitedFiles[includedFilePath] = true

	// Read the included file and parse its inputs
	hclFile := HCLFile{Path: includedFilePath}
	return hclFile.getInputsFromFileWithVisited(visitedFiles)
}



// The locals are in the form of locals = { key = value }
// Then, they are referred to as local.key in the configuration
// This function replaces the locals with their values
func replaceLocals(contents string) string {
	// Obtain the locals block and individual key/value pairs
	localsBlock, matches := getLocalsBlock(contents)
	if len(matches) == 0 {
		return contents // nothing to do
	}

	// Remove the entire locals block – we will inline the references
	if localsBlock != "" {
		contents = strings.Replace(contents, localsBlock, "", 1)
	}

	// For each local, replace occurrences of local.<key> ONLY (do not replace bare key names)
	for _, m := range matches {
		if len(m) < 2 { continue }
		key := m[0]
		value := m[1]
		trimmed := strings.TrimSpace(value)
		if !(strings.HasPrefix(trimmed, `"`) || strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "{") || trimmed == "true" || trimmed == "false" || regexp.MustCompile(`^[0-9]+$`).MatchString(trimmed)) {
			value = fmt.Sprintf("\"%s\"", trimmed)
		}
		contents = strings.ReplaceAll(contents, fmt.Sprintf("local.%s", key), value)
	}
	return contents
}

// Given an HCL file, return the inputs
func (h *HCLFile) GetInputsFromFile() (status.Inputs, error) {
	// Initialize visited files map to prevent circular includes
	visitedFiles := make(map[string]bool)
	visitedFiles[h.Path] = true
	return h.getInputsFromFileWithVisited(visitedFiles)
}

// Internal function that tracks visited files to prevent circular includes and performs parsing
func (h *HCLFile) getInputsFromFileWithVisited(visitedFiles map[string]bool) (status.Inputs, error) {
	var inputs status.Inputs

    // Helper: parse a file path into inputs using viper with locals replacement fallback
	// Returns a generic map[string]interface{} representing the 'inputs' object with all cty.Value converted
	parsePath := func(filePath string) (map[string]interface{}, error) {
        v := viper.New()
        v.SetConfigType("hcl")
        v.SetConfigFile(filePath)
        if err := v.ReadInConfig(); err != nil {
            if _, ok := err.(viper.ConfigFileNotFoundError); ok {
                return nil, fmt.Errorf("config not found: %s", filePath)
            }
            // attempt locals replacement fallback
            contents, rerr := afero.ReadFile(fs, filePath)
            if rerr != nil { return nil, fmt.Errorf("fallback read error: %w", rerr) }
            contents = []byte(replaceLocals(string(contents)))
            tempPath := "/tmp/" + path.Base(filePath)
            if werr := afero.WriteFile(fs, tempPath, contents, 0644); werr != nil { return nil, fmt.Errorf("fallback write error: %w", werr) }
            v.SetConfigFile(tempPath)
            if rerr2 := v.ReadInConfig(); rerr2 != nil {
                // final fallback: native hcl parser
                parser := hclparse.NewParser()
                file, perr := parser.ParseHCL(contents, filePath)
                if perr != nil { return nil, perr }
                attrs, _ := file.Body.JustAttributes()
                if attr, ok := attrs["inputs"]; ok {
                    val, diag := attr.Expr.Value(&hcl.EvalContext{})
                    if diag.HasErrors() { return nil, fmt.Errorf("eval diagnostics: %s", diag.Error()) }
                    if val.Type().IsObjectType() {
						obj := make(map[string]interface{})
						for k, v := range val.AsValueMap() { obj[k] = ctyToInterface(v) }
						return obj, nil
                    }
                }
                return nil, nil
            }
        }
        raw := v.Get("inputs")
        if raw == nil { return nil, nil }
        switch typed := raw.(type) {
        case []map[string]interface{}:
            if len(typed) > 0 { return typed[0], nil }
            return nil, nil
        case map[string]interface{}:
            return typed, nil
        default:
            return nil, fmt.Errorf("unsupported inputs structure: %T", raw)
        }
    }

    top, err := parsePath(h.Path)
    if err != nil {
        // Non-fatal; just log and continue
        log.Printf("GetInputsFromFile: parse error for %s: %v", h.Path, err)
    }

	if top == nil {
		// Attempt include-based parent resolution first
		contents, rerr := afero.ReadFile(fs, h.Path)
		if rerr == nil {
			included := getIncludedFilePath(string(contents), h.Path)
			if included != "" {
				parentInputs, ierr := getInputsFromIncludedFile(included, visitedFiles)
				if ierr == nil && (len(parentInputs.PrivateRepositories) > 0 || len(parentInputs.PublicRepositories) > 0 || len(parentInputs.DefaultRepositoryTeamPermissions) > 0) {
					return parentInputs, nil
				}
			}
		}
		// Fallback: walk for root.hcl/providers.hcl upward
		searchDir := path.Dir(h.Path)
		for i := 0; i < 8 && searchDir != "/"; i++ { // limit depth
			for _, candidate := range []string{"root.hcl", "providers.hcl"} {
				candidatePath := path.Join(searchDir, candidate)
				if _, statErr := fs.Stat(candidatePath); statErr == nil {
					parentTop, perr := parsePath(candidatePath)
					if perr == nil && parentTop != nil {
						top = parentTop
						break
					}
				}
			}
			if top != nil { break }
			searchDir = path.Dir(searchDir)
		}
	}

    if top == nil { return inputs, nil }

    for key, input := range top {
        switch key {
        case "private_repositories":
            repos, rerr := getRepositoryMap(input)
            if rerr != nil { log.Printf("private_repositories parse error in %s: %v", h.Path, rerr); continue }
            inputs.PrivateRepositories = repos
        case "public_repositories":
            repos, rerr := getRepositoryMap(input)
            if rerr != nil { log.Printf("public_repositories parse error in %s: %v", h.Path, rerr); continue }
            inputs.PublicRepositories = repos
        case "default_repository_team_permissions":
            permissions := make(map[string]string)
            switch typed := input.(type) {
            case []map[string]interface{}:
                if len(typed) > 0 {
                    for permission, value := range typed[0] {
                        if s, ok := value.(string); ok { permissions[permission] = s }
                    }
                }
            case map[string]interface{}:
                for permission, value := range typed {
                    if s, ok := value.(string); ok { permissions[permission] = s }
                }
            }
            inputs.DefaultRepositoryTeamPermissions = permissions
        default:
            // ignore forward-compatible keys
        }
    }
    return inputs, nil
}

// Recursively convert cty.Value into native Go types (map[string]interface{}, []interface{}, primitives)
func ctyToInterface(v cty.Value) interface{} {
	if !v.IsKnown() || v.IsNull() {
		return nil
	}
	t := v.Type()
	switch {
	case t.IsObjectType() || t.IsMapType():
		result := make(map[string]interface{})
		for k, cv := range v.AsValueMap() {
			result[k] = ctyToInterface(cv)
		}
		return result
	case t.IsTupleType() || t.IsListType() || t.IsSetType():
		var list []interface{}
		it := v.ElementIterator()
		for it.Next() {
			_, elem := it.Element()
			list = append(list, ctyToInterface(elem))
		}
		return list
	case t == cty.String:
		return v.AsString()
	case t == cty.Bool:
		return v.True() // AsBool is ambiguous? Use v.True() only if value is true else false via v.False when appropriate
	default:
		// Attempt numeric
		if t == cty.Number {
			f, _ := v.AsBigFloat().Float64()
			return f
		}
	}
	return v.GoString()
}


// Given the string content of an HCL file, return a map of locals
func (h *HCLFile) GetLocalsMap() map[string]string {

	// If the path is not set, return an empty map
	if h.Path == "" {
		return make(map[string]string)
	}

	// If the path is set, read the file and return the locals
	content, err := afero.ReadFile(fs, h.Path)
	if err != nil {
		log.Printf(`GetLocalsMap: unable to read config file: %s`, h.Path)
		return make(map[string]string)
	}

	_, macthes := getLocalsBlock(string(content))
	locals := make(map[string]string)
	for _, match := range macthes {
		if len(match) >= 2 {
			// Trim any surrounding single or double quotes from the value
			value := strings.Trim(match[1], "'\"")
			locals[match[0]] = value
		}
	}
	return locals
}


func NewTerragruntPlanFile(name string, modulePath string, moduleDir string, outputFilePath string) (*PlanFile, error) {
	// If there is a file conflict with the output file, create a new file with a "copy_" prefix
	if _, err := fs.Stat(outputFilePath); err == nil {
		dir := path.Dir(outputFilePath)
		filename := path.Base(outputFilePath)
		outputFilePath = path.Join(dir, "copy_"+filename)
	}

	return &PlanFile{
		Name:           name,
		ModuleDir:      moduleDir,
		ModulePath:     modulePath,
		OutputFilePath: outputFilePath,
	}, nil
}

func (t *PlanFile) Cleanup() error {
	return fs.Remove(t.OutputFilePath)
}

func (t *PlanFile) GetPlanFilePath() string {
	return t.OutputFilePath
}

func (t *PlanFile) RunPlan(target *string) error {
	if _, errBytes, err := runPlan(t.ModuleDir, &t.Name, target); err != nil {
		return fmt.Errorf("error running plan: %s", errBytes.String())
	}

	planFile, err := fs.Create(t.OutputFilePath)
	if err != nil {
		return err
	}
	defer planFile.Close()

	if errBytes, err := outputPlan(t.Name, planFile, t.ModuleDir); err != nil {
		return fmt.Errorf("error outputting plan: %s", errBytes.String())
	}

	return nil
}

func (t *PlanFile) GetStateExplorer() (terraform_state.IStateExplorer, error) {
	planBytes, err := afero.ReadFile(fs, t.OutputFilePath)
	if err != nil {
		return nil, err
	}

	var explorer terraform_state.IStateExplorer
	versionQuery := "format_version"
	gjsonResult := gjson.GetBytes(planBytes, versionQuery)
	if !gjsonResult.Exists() {
		return nil, fmt.Errorf("unable to determine plan version")
	} else if gjsonResult.Type != gjson.String {
		return nil, fmt.Errorf("unexpected type for %q: %s", versionQuery, gjsonResult.Type)
	}
	version := gjsonResult.String()

	switch version {
	case "1.2":
		explorer = &v1_2.StateExplorer{}
	default:
		return nil, fmt.Errorf("unsupported version %q", version)
	}

	explorer.SetPlan(planBytes)
	return explorer, nil
}

type ImportIdResolver interface {
	ResolveImportId(resourceAddress string) (string, error)
}

func outputPlan(planName string, planFile io.Writer, dir string) (bytes.Buffer, error) {
	errBuffer := &bytes.Buffer{}
	cmdExecutor := newCommandExecutor("terragrunt", "show", "-json", planName)
	cmdExecutor.SetOutput(planFile)
	cmdExecutor.SetErrorOutput(errBuffer)
	cmdExecutor.SetDir(dir)
	if err := cmdExecutor.Run(); err != nil {
		return *errBuffer, err
	}
	return *errBuffer, nil
}

func runPlan(dir string, output *string, target *string) (bytes.Buffer, bytes.Buffer, error) {
	errBuffer := &bytes.Buffer{}
	logBuffer := &bytes.Buffer{}
	args := []string{"plan", "-lock=false"}
	if output != nil {
		args = append(args, fmt.Sprintf("-out=%s", *output))
	}
	if target != nil {
		args = append(args, fmt.Sprintf("-target=%s", *target))
	}

	cmdExecutor := newCommandExecutor("terragrunt", args...)
	cmdExecutor.SetErrorOutput(errBuffer)
	cmdExecutor.SetDir(dir)
	if err := cmdExecutor.Run(); err != nil {
		return *logBuffer, *errBuffer, err
	} else {
		logBuffer.WriteString(fmt.Sprintf("Command %q complete", cmdExecutor.String()))
	}
	return *logBuffer, *errBuffer, nil
}
