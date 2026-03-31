package project

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/cmd/output"
	"github.com/JetBrains/youtrack-cli/internal"
)

func newProjectGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <shortName>",
		Short: "Get a project by short name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
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
