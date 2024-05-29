package functions

import (
	githubfoundations "gh_foundations/internal/pkg/types/github_foundations"
	"gh_foundations/internal/pkg/types/status"
	"gh_foundations/internal/pkg/types/terragrunt"
	"log"
	"os"
	"path/filepath"
	"strings"
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
func findOrgFiles(rootDir string) (map[string][]string, error) {

	// Get the list of HCL files in the root directory
	// hclFiles, err := config.FindConfigFilesInPath(rootDir, options)
	var hclFiles []string
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// if the file ends with "repositories/terragrunt.hcl", then it is a repository fil
		if strings.HasSuffix(path, "repositories/terragrunt.hcl") {
			hclFiles = append(hclFiles, path)
		}

		return nil
	})
    if err != nil {
        return nil, err
    }

    orgFiles := findOrgsFromFilenames(hclFiles)
	return orgFiles, nil
}


// List all of the repositories managed by the tool
func FindManagedRepos(reposDir string) (status.OrgSet, error) {
	orgFiles, err := findOrgFiles(reposDir)
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

				hclFile := terragrunt.HCLFile {
					Path: file,
				}

				inputs, err := hclFile.GetInputsFromFile()
				if err != nil {
					log.Fatalf(`Error in getInputsFromFile: %s`, err)
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
					repoInput := repo.GetRepositoryInput()
					repoInput.Name = name
					repoSet.PrivateRepositories = append(repoSet.PrivateRepositories, &repoInput)
				}
				for name, repo := range inputs.PublicRepositories {
					// Coerce the repo into a githubfoundations.RepositoryInput
					repoInput := repo.GetRepositoryInput()
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
