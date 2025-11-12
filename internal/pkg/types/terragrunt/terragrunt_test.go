package terragrunt

import (
	"encoding/json"
	"errors"
	"gh_foundations/internal/pkg/types"
	typeMocks "gh_foundations/internal/pkg/types/mocks"
	"io"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestTerragruntArchiveTestSuite(t *testing.T) {
	suite.Run(t, new(TerragruntArchiveTestSuite))
}

type TerragruntArchiveTestSuite struct {
	suite.Suite
	mockCmdExecutor *typeMocks.MockICommandExecutor
}

func (suite *TerragruntArchiveTestSuite) SetupTest() {
	fs = afero.NewMemMapFs()
	suite.mockCmdExecutor = new(typeMocks.MockICommandExecutor)
}

func (suite *TerragruntArchiveTestSuite) TestNewTerragruntPlanFile() {
	name := "test"
	modulePath := "path/to/module"
	moduleDir := "path/to/module/dir"
	outputFilePath := "path/to/output/file"

	planFile, err := NewTerragruntPlanFile(name, modulePath, moduleDir, outputFilePath)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), planFile)
	assert.Equal(suite.T(), name, planFile.Name)
	assert.Equal(suite.T(), modulePath, planFile.ModulePath)
	assert.Equal(suite.T(), moduleDir, planFile.ModuleDir)
	assert.Equal(suite.T(), outputFilePath, planFile.OutputFilePath)
}

func (suite *TerragruntArchiveTestSuite) TestNewTerragruntPlanFileAlreadyExists() {
	name := "test"
	modulePath := "path/to/module"
	moduleDir := "path/to/module/dir"
	outputFilePaths := []string{"path/to/output/file", "path/to/output/file.json"}
	expectedOutputFilePaths := []string{"path/to/output/copy_file", "path/to/output/copy_file.json"}

	for i := range outputFilePaths {
		f, err := fs.Create(outputFilePaths[i])
		require.NoError(suite.T(), err)
		f.Close()

		planFile, err := NewTerragruntPlanFile(name, modulePath, moduleDir, outputFilePaths[i])

		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), planFile)
		assert.Equal(suite.T(), name, planFile.Name)
		assert.Equal(suite.T(), modulePath, planFile.ModulePath)
		assert.Equal(suite.T(), moduleDir, planFile.ModuleDir)
		assert.Equal(suite.T(), expectedOutputFilePaths[i], planFile.OutputFilePath)
	}
}

func (suite *TerragruntArchiveTestSuite) TestPlanFileRunPlan() {
	actualArgs := make([]string, 0)
	newCommandExecutor = func(_ string, args ...string) types.ICommandExecutor {
		if args[0] == "plan" {
			actualArgs = args
		}
		return suite.mockCmdExecutor
	}
	planFile := &PlanFile{
		Name:           "test",
		ModulePath:     "path/to/module",
		ModuleDir:      "path/to/module/dir",
		OutputFilePath: "path/to/output/file",
	}

	suite.mockCmdExecutor.EXPECT().Run().Return(nil)
	suite.mockCmdExecutor.EXPECT().SetDir("path/to/module/dir").Return()
	suite.mockCmdExecutor.EXPECT().SetOutput(mock.Anything).Return()
	suite.mockCmdExecutor.EXPECT().SetErrorOutput(mock.Anything).Return()
	suite.mockCmdExecutor.EXPECT().String().Return("")

	targets := []string{"", "test-target"}
	for _, target := range targets {
		var err error
		if target == "" {
			err = planFile.RunPlan(nil)
		} else {
			err = planFile.RunPlan(&target)
		}
		assert.NoError(suite.T(), err)
		if target != "" {
			assert.Contains(suite.T(), actualArgs, "-target="+target)
		}
	}
}

func (suite *TerragruntArchiveTestSuite) TestPlanFileRunPlanCommandFailure() {
	var errBufferWriterFunc func(errMessage string)
	newCommandExecutor = func(_ string, args ...string) types.ICommandExecutor {
		if args[0] == "plan" {
			suite.mockCmdExecutor.EXPECT().Run().RunAndReturn(func() error {
				errBufferWriterFunc("error bad plan")
				return errors.New("")
			}).Once()
		} else {
			suite.mockCmdExecutor.EXPECT().Run().Return(nil).Once()
		}
		return suite.mockCmdExecutor
	}
	planFile := &PlanFile{
		Name:           "test",
		ModulePath:     "path/to/module",
		ModuleDir:      "path/to/module/dir",
		OutputFilePath: "path/to/output/file",
	}
	expectedErrorMessage := "error running plan: error bad plan"

	suite.mockCmdExecutor.EXPECT().SetDir("path/to/module/dir").Return()
	suite.mockCmdExecutor.EXPECT().SetOutput(mock.Anything).Return()
	suite.mockCmdExecutor.EXPECT().SetErrorOutput(mock.Anything).Run(func(writer io.Writer) {
		errBufferWriterFunc = func(errMessage string) {
			writer.Write([]byte(errMessage))
		}
	}).Return()
	suite.mockCmdExecutor.EXPECT().String().Return("")

	err := planFile.RunPlan(nil)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), expectedErrorMessage, err.Error())
}

func (suite *TerragruntArchiveTestSuite) TestPlanFileRunPlanFileCreateFailure() {
	fs = afero.NewReadOnlyFs(fs)
	newCommandExecutor = func(_ string, _ ...string) types.ICommandExecutor {
		return suite.mockCmdExecutor
	}
	planFile := &PlanFile{
		Name:           "test",
		ModulePath:     "path/to/module",
		ModuleDir:      "path/to/module/dir",
		OutputFilePath: "bad/path/plan.json",
	}

	// Expect to see error thrown by the afero read only file system
	expectedErrorMessage := "operation not permitted"

	suite.mockCmdExecutor.EXPECT().Run().Return(nil)
	suite.mockCmdExecutor.EXPECT().SetDir("path/to/module/dir").Return()
	suite.mockCmdExecutor.EXPECT().SetOutput(mock.Anything).Return()
	suite.mockCmdExecutor.EXPECT().SetErrorOutput(mock.Anything).Return()
	suite.mockCmdExecutor.EXPECT().String().Return("")

	err := planFile.RunPlan(nil)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), expectedErrorMessage, err.Error())
}

func (suite *TerragruntArchiveTestSuite) TestPlanFileRunShowCommandFailure() {
	var errBufferWriterFunc func(errMessage string)
	newCommandExecutor = func(_ string, args ...string) types.ICommandExecutor {
		if args[0] == "show" {
			suite.mockCmdExecutor.EXPECT().Run().RunAndReturn(func() error {
				errBufferWriterFunc("error show failed")
				return errors.New("")
			}).Once()
		} else {
			suite.mockCmdExecutor.EXPECT().Run().Return(nil).Once()
		}
		return suite.mockCmdExecutor
	}
	planFile := &PlanFile{
		Name:           "test",
		ModulePath:     "path/to/module",
		ModuleDir:      "path/to/module/dir",
		OutputFilePath: "path/to/output/file",
	}
	expectedErrorMessage := "error outputting plan: error show failed"

	suite.mockCmdExecutor.EXPECT().SetDir("path/to/module/dir").Return()
	suite.mockCmdExecutor.EXPECT().SetOutput(mock.Anything).Return()
	suite.mockCmdExecutor.EXPECT().SetErrorOutput(mock.AnythingOfType("*bytes.Buffer")).Run(func(writer io.Writer) {
		errBufferWriterFunc = func(errMessage string) {
			writer.Write([]byte(errMessage))
		}
	}).Return()
	suite.mockCmdExecutor.EXPECT().String().Return("")

	err := planFile.RunPlan(nil)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), expectedErrorMessage, err.Error())
}

func (suite *TerragruntArchiveTestSuite) TestPlanFileGetStateExplorer() {
	planFile := &PlanFile{
		Name:           "test",
		ModulePath:     "path/to/module",
		ModuleDir:      "path/to/module/dir",
		OutputFilePath: "plan.json",
	}
	versions := []string{"1.2"}
	for _, version := range versions {
		var fileName = "plan.json"
		jsonContents := map[string]any{
			"format_version": version,
		}
		bytes, err := json.Marshal(jsonContents)
		require.NoError(suite.T(), err)
		err = afero.WriteFile(fs, fileName, bytes, 0644)
		require.NoError(suite.T(), err)

		stateExplorer, err := planFile.GetStateExplorer()
		assert.NotNil(suite.T(), stateExplorer)
		assert.NoError(suite.T(), err)

		fs.Remove(fileName)
	}
}

func (suite *TerragruntArchiveTestSuite) TestPlanFileGetStateExplorerReadFileFailure() {
	planFile := &PlanFile{
		Name:           "test",
		ModulePath:     "path/to/module",
		ModuleDir:      "path/to/module/dir",
		OutputFilePath: "plan.json",
	}
	stateExplorer, err := planFile.GetStateExplorer()
	assert.Nil(suite.T(), stateExplorer)
	assert.Error(suite.T(), err)
}

func (suite *TerragruntArchiveTestSuite) TestPlanFileGetStateExplorerVersionQueryFailure() {
	planFile := &PlanFile{
		Name:           "test",
		ModulePath:     "path/to/module",
		ModuleDir:      "path/to/module/dir",
		OutputFilePath: "plan.json",
	}

	fileContents := []map[string]any{
		{
			"bad_format_version_key": 1.2,
		},
		{
			"format_version": map[string]any{
				"bad_format_version_type": 1.2,
			},
		},
	}

	for _, fileContent := range fileContents {
		var fileName = "plan.json"
		bytes, err := json.Marshal(fileContent)
		require.NoError(suite.T(), err)
		err = afero.WriteFile(fs, fileName, bytes, 0644)
		require.NoError(suite.T(), err)

		stateExplorer, err := planFile.GetStateExplorer()

		assert.Nil(suite.T(), stateExplorer)
		assert.Error(suite.T(), err)

		fs.Remove(fileName)
	}
}

func (suite *TerragruntArchiveTestSuite) TestPlanFileGetStateExplorerUnsupportedVersionFailure() {
	planFile := &PlanFile{
		Name:           "test",
		ModulePath:     "path/to/module",
		ModuleDir:      "path/to/module/dir",
		OutputFilePath: "plan.json",
	}

	var fileName = "plan.json"
	jsonContents := map[string]any{
		"format_version": "1.0",
	}
	bytes, err := json.Marshal(jsonContents)
	require.NoError(suite.T(), err)
	err = afero.WriteFile(fs, fileName, bytes, 0644)
	require.NoError(suite.T(), err)

	stateExplorer, err := planFile.GetStateExplorer()
	assert.Nil(suite.T(), stateExplorer)
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), "unsupported version \"1.0\"", err.Error())

	fs.Remove(fileName)
}

// New test to validate parsing of inputs with a locals block and local references
func TestGetInputsFromFile_WithLocals(t *testing.T) {
		fs = afero.NewMemMapFs()
		fileName := "terragrunt.hcl"
		contents := `
locals {
	repo_description = "Example repository"
}

inputs = {
	private_repositories = {
		sample = {
			description = local.repo_description
			default_branch = "main"
			advance_security = true
			has_vulnerability_alerts = true
			topics = ["a", "b"]
			homepage = "https://example.com"
			delete_head_on_merge = true
			requires_web_commit_signing = true
			dependabot_security_updates = true
			protected_branches = ["main"]
			allow_auto_merge = false
		}
	}
	public_repositories = {}
}
`
		err := afero.WriteFile(fs, fileName, []byte(contents), 0644)
		require.NoError(t, err)

		hclFile := HCLFile{Path: fileName}
		inputs, err := hclFile.GetInputsFromFile()
		require.NoError(t, err)
		if len(inputs.PrivateRepositories) != 1 {
				t.Fatalf("expected 1 private repo, got %d", len(inputs.PrivateRepositories))
		}
		repo := inputs.PrivateRepositories["sample"]
		if repo.Description != "Example repository" {
				t.Fatalf("expected description to be 'Example repository', got %s", repo.Description)
		}
}

// TestGetIncludedFilePath tests detecting and resolving include blocks
func TestGetIncludedFilePath(t *testing.T) {
	fs = afero.NewMemMapFs()

	testCases := []struct {
		name           string
		contents       string
		currentPath    string
		setupFiles     map[string]string // path -> content
		expectedPath   string
		expectNotFound bool
	}{
		{
			name: "finds root.hcl in parent directory",
			contents: `
include "root" {
	path = find_in_parent_folders("root.hcl")
}
`,
			currentPath: "/project/org/repositories/terragrunt.hcl",
			setupFiles: map[string]string{
				"/project/org/root.hcl": "# root config",
			},
			expectedPath: "/project/org/root.hcl",
		},
		{
			name: "finds root.hcl in grandparent directory",
			contents: `
include "root" {
	path = find_in_parent_folders("root.hcl")
}
`,
			currentPath: "/project/org/subdir/repositories/terragrunt.hcl",
			setupFiles: map[string]string{
				"/project/org/root.hcl": "# root config",
			},
			expectedPath: "/project/org/root.hcl",
		},
		{
			name: "no include block",
			contents: `
inputs = {
	test = "value"
}
`,
			currentPath:    "/project/terragrunt.hcl",
			setupFiles:     map[string]string{},
			expectNotFound: true,
		},
		{
			name: "file not found",
			contents: `
include "root" {
	path = find_in_parent_folders("nonexistent.hcl")
}
`,
			currentPath:    "/project/org/terragrunt.hcl",
			setupFiles:     map[string]string{},
			expectNotFound: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs = afero.NewMemMapFs()

			// Setup parent files
			for path, content := range tc.setupFiles {
				dir := filepath.Dir(path)
				err := fs.MkdirAll(dir, 0755)
				require.NoError(t, err)
				err = afero.WriteFile(fs, path, []byte(content), 0644)
				require.NoError(t, err)
			}

			result := getIncludedFilePath(tc.contents, tc.currentPath)

			if tc.expectNotFound {
				assert.Empty(t, result, "should not find any file")
			} else {
				assert.Equal(t, tc.expectedPath, result)
			}
		})
	}
}

// TestGetInputsFromFile_WithInclude tests resolving inputs from included parent files
func TestGetInputsFromFile_WithInclude(t *testing.T) {
	fs = afero.NewMemMapFs()

	// Create parent root.hcl with inputs
	rootContent := `
inputs = {
	private_repositories = {
		included-repo = {
			description = "Repository from parent"
			default_branch = "main"
		}
	}
	public_repositories = {}
	default_repository_team_permissions = {}
}
`
	err := fs.MkdirAll("/project/org", 0755)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "/project/org/root.hcl", []byte(rootContent), 0644)
	require.NoError(t, err)

	// Create child terragrunt.hcl that includes parent
	childContent := `
include "root" {
	path = find_in_parent_folders("root.hcl")
}

locals {
	project_name = "test-project"
}
`
	err = fs.MkdirAll("/project/org/repositories", 0755)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "/project/org/repositories/terragrunt.hcl", []byte(childContent), 0644)
	require.NoError(t, err)

	// Test reading from child file
	hclFile := HCLFile{Path: "/project/org/repositories/terragrunt.hcl"}
	inputs, err := hclFile.GetInputsFromFile()
	require.NoError(t, err)

	// Should have resolved inputs from parent
	assert.Equal(t, 1, len(inputs.PrivateRepositories))
	repo, exists := inputs.PrivateRepositories["included-repo"]
	assert.True(t, exists)
	assert.Equal(t, "Repository from parent", repo.Description)
}

// TestGetInputsFromFile_DirectInputs tests reading direct inputs (backward compatibility)
func TestGetInputsFromFile_DirectInputs(t *testing.T) {
	fs = afero.NewMemMapFs()

	content := `
inputs = {
	private_repositories = {
		direct-repo = {
			description = "Direct repository"
			default_branch = "main"
		}
	}
	public_repositories = {}
	default_repository_team_permissions = {}
}
`
	fileName := "terragrunt.hcl"
	err := afero.WriteFile(fs, fileName, []byte(content), 0644)
	require.NoError(t, err)

	hclFile := HCLFile{Path: fileName}
	inputs, err := hclFile.GetInputsFromFile()
	require.NoError(t, err)

	assert.Equal(t, 1, len(inputs.PrivateRepositories))
	repo, exists := inputs.PrivateRepositories["direct-repo"]
	assert.True(t, exists)
	assert.Equal(t, "Direct repository", repo.Description)
}

// TestGetInputsFromFile_CircularInclude tests circular include detection
func TestGetInputsFromFile_CircularInclude(t *testing.T) {
	fs = afero.NewMemMapFs()

	// Create fileA that includes fileB
	fileAContent := `
include "root" {
	path = find_in_parent_folders("fileB.hcl")
}
`
	err := fs.MkdirAll("/project", 0755)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "/project/fileA.hcl", []byte(fileAContent), 0644)
	require.NoError(t, err)

	// Create fileB that includes fileA (circular)
	fileBContent := `
include "root" {
	path = find_in_parent_folders("fileA.hcl")
}
`
	err = afero.WriteFile(fs, "/project/fileB.hcl", []byte(fileBContent), 0644)
	require.NoError(t, err)

	// Try to read fileA (which will try to include fileB, which tries to include fileA)
	hclFile := HCLFile{Path: "/project/fileA.hcl"}
	inputs, err := hclFile.GetInputsFromFile()

	// Should return empty inputs without crashing
	assert.NoError(t, err)
	assert.Equal(t, 0, len(inputs.PrivateRepositories))
	assert.Equal(t, 0, len(inputs.PublicRepositories))
}

// TestGetInputsFromFile_NoInputsSection tests files without inputs section
func TestGetInputsFromFile_NoInputsSection(t *testing.T) {
	fs = afero.NewMemMapFs()

	content := `
locals {
	project_name = "test"
}
`
	fileName := "terragrunt.hcl"
	err := afero.WriteFile(fs, fileName, []byte(content), 0644)
	require.NoError(t, err)

	hclFile := HCLFile{Path: fileName}
	inputs, err := hclFile.GetInputsFromFile()

	// Should return empty inputs without error
	assert.NoError(t, err)
	assert.Equal(t, 0, len(inputs.PrivateRepositories))
	assert.Equal(t, 0, len(inputs.PublicRepositories))
}

// TestGetInputsFromFile_EmptyInputs tests files with empty inputs
func TestGetInputsFromFile_EmptyInputs(t *testing.T) {
	fs = afero.NewMemMapFs()

	content := `
inputs = {}
`
	fileName := "terragrunt.hcl"
	err := afero.WriteFile(fs, fileName, []byte(content), 0644)
	require.NoError(t, err)

	hclFile := HCLFile{Path: fileName}
	inputs, err := hclFile.GetInputsFromFile()

	assert.NoError(t, err)
	assert.Equal(t, 0, len(inputs.PrivateRepositories))
	assert.Equal(t, 0, len(inputs.PublicRepositories))
}

// TestGetInputsFromFile_MultipleRepositories tests parsing multiple repositories
func TestGetInputsFromFile_MultipleRepositories(t *testing.T) {
	fs = afero.NewMemMapFs()

	content := `
inputs = {
	private_repositories = {
		repo1 = {
			description = "First repo"
			default_branch = "main"
		}
		repo2 = {
			description = "Second repo"
			default_branch = "develop"
		}
	}
	public_repositories = {
		public-repo = {
			description = "Public repo"
			default_branch = "main"
		}
	}
	default_repository_team_permissions = {}
}
`
	fileName := "terragrunt.hcl"
	err := afero.WriteFile(fs, fileName, []byte(content), 0644)
	require.NoError(t, err)

	hclFile := HCLFile{Path: fileName}
	inputs, err := hclFile.GetInputsFromFile()
	require.NoError(t, err)

	assert.Equal(t, 2, len(inputs.PrivateRepositories))
	assert.Equal(t, 1, len(inputs.PublicRepositories))

	repo1, exists := inputs.PrivateRepositories["repo1"]
	assert.True(t, exists)
	assert.Equal(t, "First repo", repo1.Description)

	repo2, exists := inputs.PrivateRepositories["repo2"]
	assert.True(t, exists)
	assert.Equal(t, "Second repo", repo2.Description)

	publicRepo, exists := inputs.PublicRepositories["public-repo"]
	assert.True(t, exists)
	assert.Equal(t, "Public repo", publicRepo.Description)
}

// TestGetLocalsMap tests extracting locals from HCL files
func TestGetLocalsMap(t *testing.T) {
	fs = afero.NewMemMapFs()

	testCases := []struct {
		name     string
		content  string
		expected map[string]string
	}{
		{
			name: "simple locals",
			content: `
locals {
	organization_name = "test-org"
	region = "us-east-1"
}
`,
			expected: map[string]string{
				"organization_name": "test-org",
				"region":            "us-east-1",
			},
		},
		{
			name: "no locals",
			content: `
inputs = {
	test = "value"
}
`,
			expected: map[string]string{},
		},
		{
			name: "locals with various types",
			content: `
locals {
	string_value = "test"
	number_value = 42
	bool_value = true
}
`,
			expected: map[string]string{
				"string_value": "test",
				"number_value": "42",
				"bool_value":   "true",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fileName := "test.hcl"
			err := afero.WriteFile(fs, fileName, []byte(tc.content), 0644)
			require.NoError(t, err)

			hclFile := HCLFile{Path: fileName}
			result := hclFile.GetLocalsMap()

			for key, expectedValue := range tc.expected {
				actualValue, exists := result[key]
				assert.True(t, exists, "key %s should exist", key)
				assert.Equal(t, expectedValue, actualValue, "value for key %s", key)
			}

			// Check no extra keys
			assert.Equal(t, len(tc.expected), len(result))
		})
	}
}
