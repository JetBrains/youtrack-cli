package project

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newProjectArchiveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "archive <shortName>",
		Short: "Archive a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.ArchiveProject(args[0]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "archived project "+args[0])
			return nil
		},
	}
}
