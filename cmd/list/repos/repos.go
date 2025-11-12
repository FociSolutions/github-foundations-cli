/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package list

import (
	"encoding/json"
	"errors"
	"fmt"
	"gh_foundations/internal/pkg/functions"
	"gh_foundations/internal/pkg/types/status"
	"io"
	"log"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var ghas bool
var verbose bool
var outputFormat string

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

		// Suppress all logs unless verbose requested
		if !verbose {
			log.SetOutput(io.Discard)
		}
		orgSet, err := functions.FindManagedRepos(reposDir)
		if err != nil {
			log.Printf("list repos: error discovering repositories: %v", err)
			fmt.Println("[]")
			return
		}

		repoList := flattenRepos(orgSet)

		if ghas {
			orgSet = orgSet.WithGHASEnabled()
			repoList = flattenRepos(orgSet)
			log.Printf("Found %d repositories with GHAS enabled\n", len(repoList))
		}

		sort.Strings(repoList)
		if outputFormat == "json" {
			data, _ := json.Marshal(repoList) // data is always slice of strings, marshal can't fail
			fmt.Println(string(data))
			return
		}
		// Default legacy format
		if len(repoList) == 0 {
			fmt.Println("[]")
			return
		}
		fmt.Printf("['%s']\n", strings.Join(repoList, "', '"))
	},
}

func init() {
	ReposCmd.Flags().BoolVarP(&ghas, "ghas", "g", false, "List repositories with GHAS enabled")
	ReposCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show diagnostic logs while listing")
	ReposCmd.Flags().StringVarP(&outputFormat, "format", "f", "legacy", "Output format: legacy | json")
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
