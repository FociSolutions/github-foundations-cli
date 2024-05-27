/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package list

import (
	"errors"
	"fmt"
	"gh_foundations/internal/pkg/functions"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var OrgsCmd = &cobra.Command{
	Use:   "orgs",
	Short: "List managed organizations.",
	Long: `List managed organizations. This command will list all organizations.\n
	found in the state file.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("requires the path of the \"organizations\" directory")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {

		orgsDir := args[0]

		orgs, err := functions.FindManagedOrgs(orgsDir)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		var orgOut string = "[]"

		if len(orgs) > 0 {
			orgOut = fmt.Sprintf("['%s']", strings.Join(orgs, "', '"))
		}

		fmt.Println(orgOut)

	},
}

func init() {
	os.Setenv("GCP_SECRET_MANAGER_PROJECT", "blahblah")
	os.Setenv("GCP_TF_STATE_BUCKET_PROJECT", "blahblahblah")
	os.Setenv("GCP_TF_STATE_BUCKET_NAME", "blahblahblahblah")
	os.Setenv("GCP_TF_STATE_BUCKET_LOCATION", "blahblahblahblahblah")
}
