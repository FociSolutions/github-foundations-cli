package functions

import (
	"context"
	githubfoundations "gh_foundations/internal/pkg/types/github_foundations"
	"gh_foundations/internal/pkg/types/status"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gruntwork-io/terragrunt/config"
	"github.com/gruntwork-io/terragrunt/config/hclparse"
	"github.com/gruntwork-io/terragrunt/options"
	"github.com/mitchellh/mapstructure"
)

// Given a set of HCL file, find the org name
// The first parameter is the list of HCL files
func findOrgsFromFilenames(hclFiles []string) map[string][]string {
    names := make(map[string][]string)

    for _, file := range hclFiles {
        dirs := strings.Split(file, "/")
        orgName := dirs[len(dirs)-3]
        names[orgName] = append(names[orgName], file)
    }
    return names
}

// Given a repository struct returned by the HCL parser, return a githubfoundations.RepositoryInput
func GetRepositoryInput(repo status.Repository) githubfoundations.RepositoryInput {
	return githubfoundations.RepositoryInput{
		Name: repo.Name,
		AdvanceSecurity: repo.AdvanceSecurity,
		AllowAutoMerge: repo.AllowAutoMerge,
		DefaultBranch: repo.DefaultBranch,
		DeleteHeadBranchOnMerge: repo.DeleteHeadBranchOnMerge,
		DependabotSecurityUpdates: repo.DependabotSecurityUpdates,
		Description: repo.Description,
		HasVulnerabilityAlerts: repo.HasVulnerabilityAlerts,
		Homepage: repo.Homepage,
		ProtectedBranches: repo.ProtectedBranches,
		RequiresWebCommitSignOff: repo.RequiresWebCommitSignOff,
		Topics: repo.Topics,
	}
}

// Setup the terragrunt options with defaults
func getOptions() *options.TerragruntOptions {
	tgOptions := options.NewTerragruntOptions()
	tgOptions.Env = map[string]string{
		"GCP_SECRET_MANAGER_PROJECT": os.Getenv("GCP_SECRET_MANAGER_PROJECT"),
		"GCP_TF_STATE_BUCKET_PROJECT": os.Getenv("GCP_SECRET_MANAGER_PROJECT"),
		"GCP_TF_STATE_BUCKET_NAME": os.Getenv("GCP_TF_STATE_BUCKET_NAME"),
		"GCP_TF_STATE_BUCKET_LOCATION": os.Getenv("GCP_TF_STATE_BUCKET_LOCATION"),
	}
	return tgOptions
}

// List all of the organizations managed by the tool
func FindManagedOrgs(orgsDir string) ([]string, error) {
	// Get the list of the dir names in the  directory
	dirs, err := os.ReadDir(orgsDir)
	if err != nil {
		log.Fatalf("Error in os.ReadDir: %s", err)
        return nil, err
	}

	var orgs []string
	for _, dir := range dirs {
		orgs = append(orgs, dir.Name())
	}

	return orgs, nil
}

// List all of the organizations + repository configs managed by the tool
func findOrgFiles(rootDir string, options *options.TerragruntOptions) (map[string][]string, error) {

	// Get the list of HCL files in the root directory
	hclFiles, err := config.FindConfigFilesInPath(rootDir, options)
    if err != nil {
        return nil, err
    }

    orgFiles := findOrgsFromFilenames(hclFiles)
	return orgFiles, nil
}

// List all of the repositories managed by the tool
func FindManagedRepos(ctx context.Context, reposDir string) (status.OrgSet, error) {
	options := getOptions()

	orgFiles, err := findOrgFiles(reposDir, options)
	if err != nil {
		log.Fatalf("Error in findOrgFiles: %s", err)
        return status.OrgSet{}, err
	}

	// Get the absolute path of the root directory
	absRootPath, err := filepath.Abs(reposDir)
	if err != nil {
		log.Fatalf("Error in filepath.Abs: %s", err)
        return status.OrgSet{}, err
	}

	var orgSet status.OrgSet
	orgSet.OrgProjectSets = make(map[string]status.OrgProjectSet)

	for org, files := range orgFiles {

		var repos status.OrgProjectSet
		repos.RepositorySets = make(map[string]githubfoundations.RepositorySetInput)
		orgSet.OrgProjectSets[org] = repos

		for _, file := range files {

			// If the file name ends with `../repositories/terragrunt.hcl`,
			// then it is a repository file
			if strings.HasSuffix(file, "repositories/terragrunt.hcl") {

				// Strip the trailing / from the reposDir
				replaceDir := strings.TrimSuffix(reposDir, "/")
				// Replace relative path with absolute path
				file = strings.Replace(file, replaceDir, absRootPath, 1)

				log.Printf("Working on file: %s\n", file)

				// Get the project name
				parts := strings.Split(file, "/")
				project := parts[len(parts)-4]

				// Parse the HCL file
				options.TerragruntConfigPath = file
				options.WorkingDir = path.Dir(file)
				parseCtx := config.NewParsingContext(ctx, options)
				parser := hclparse.NewParser()
				parsedHCL, err := parser.ParseFromFile(file)

				if err != nil {
					log.Fatalf(`Error in hclparse.NewParser().ParseFromFile: %s`, err)
					return orgSet, err
				}

				tfConfig, err := config.ParseConfig(parseCtx, parsedHCL, nil)
				if err != nil {
					log.Fatalf(`Error in config.ParseConfig: %s`, err)
					return orgSet, err
				}

				var inputs status.Inputs
				err = mapstructure.Decode(tfConfig.Inputs, &inputs)
				if err != nil {
					log.Fatalf(`Error in mapstructure.Decode: %s`, err)
					return orgSet, err
				}


				log.Printf("Repository Set has %d private repositories and %d public repositories", len(inputs.PrivateRepositories), len(inputs.PublicRepositories))
				var repoSet githubfoundations.RepositorySetInput
				for key, value := range inputs.DefaultRepositoryTeamPermissions {
					repoSet.DefaultRepositoryTeamPermissions = make(map[string]string)
					repoSet.DefaultRepositoryTeamPermissions[key] = value
				}

				for name, repo := range inputs.PrivateRepositories {
					// Coerce the repo into a githubfoundations.RepositoryInput
					repoInput := GetRepositoryInput(repo)
					repoInput.Name = name
					repoSet.PrivateRepositories = append(repoSet.PrivateRepositories, &repoInput)
				}
				for name, repo := range inputs.PublicRepositories {
					// Coerce the repo into a githubfoundations.RepositoryInput
					repoInput := GetRepositoryInput(repo)
					repoInput.Name = name
					repoSet.PublicRepositories = append(repoSet.PublicRepositories, &repoInput)
				}

				// Add the repoSet to the orgSet
				orgSet.OrgProjectSets[org].RepositorySets[project] = repoSet
			}
		}
	}
	return orgSet, nil
}
