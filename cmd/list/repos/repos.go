/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package list

import (
	"errors"
	"fmt"
	"gh_foundations/internal/pkg/functions"
	"gh_foundations/internal/pkg/types/status"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var ghas bool

var ReposCmd = &cobra.Command{
	Use:   "repos",
	Short: "List managed repositories.",
	Long: `List managed repositories. This command will list all repositories.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("requires the path of the \"projects\" directory")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {

		reposDir := args[0]

		ctx := cmd.Context()

		orgSet, err := functions.FindManagedRepos(ctx, reposDir)
		if err != nil {
			log.Fatalf("Error in findManagedRepos: %s", err)
		}
		
		if ghas {
			orgSet = orgSet.WithGHASEnabled()
			log.Printf("Found %d repositories with GHAS enabled\n", len(flattenRepos(orgSet)))
		}
		var repos string
		repos = ""
		for _, repo := range flattenRepos(orgSet) {
			if repos != "" {
				repos = fmt.Sprintf("%s, '%s'", repos, repo)
			} else {
				repos = fmt.Sprintf("['%s'", repo)
			}
		}
		repos = fmt.Sprintf("%s]", repos)

		fmt.Println(repos)
	},
}

func init() {
	os.Setenv("GCP_SECRET_MANAGER_PROJECT", "blahblah")
	os.Setenv("GCP_TF_STATE_BUCKET_PROJECT", "blahblahblah")
	os.Setenv("GCP_TF_STATE_BUCKET_NAME", "blahblahblahblah")
	os.Setenv("GCP_TF_STATE_BUCKET_LOCATION", "blahblahblahblahblah")

	ReposCmd.Flags().BoolVarP(&ghas, "ghas", "g", false, "List repositories with GHAS enabled")
}

// Return only the names of the repositories managed by the tool
func flattenRepos(org status.OrgSet) []string {
	var repoNames []string

	for org, projects := range org.OrgProjectSets {
		for _, repoSet := range projects.RepositorySets {
			for _, repo := range repoSet.PrivateRepositories {
				repoNames = append(repoNames, fmt.Sprintf("%s/%s", org, repo.Name))
			}
			for _, repo := range repoSet.PublicRepositories {
				repoNames = append(repoNames, fmt.Sprintf("%s/%s", org, repo.Name))
			}
		}
	}
	return repoNames
}