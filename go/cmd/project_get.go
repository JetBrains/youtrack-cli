package cmd

import (
	"youtrack-cli/cmd/output"

	"github.com/spf13/cobra"
)

func newProjectGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <shortName>",
		Short: "Get a project by short name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			project, err := client.GetProject(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, project)
		},
	}
}
