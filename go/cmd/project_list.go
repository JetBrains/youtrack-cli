package cmd

import (
	"youtrack-cli/cmd/output"

	"github.com/spf13/cobra"
)

func newProjectListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			projects, err := client.ListProjects()
			if err != nil {
				return err
			}
			return output.Print(cmd, projects)
		},
	}
}
