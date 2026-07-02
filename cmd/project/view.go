package project

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newProjectViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "view <shortName>",
		Aliases: []string{"get"},
		Short:   "View a project by short name",
		Example: "  youtrack project view FOO",
		Args:    cobra.ExactArgs(1),
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
