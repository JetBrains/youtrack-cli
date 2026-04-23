package project

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newProjectDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <shortName>",
		Short: "Delete a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			confirmed, err := internal.Confirm(cmd, fmt.Sprintf("Delete project %s and all its issues?", args[0]))
			if err != nil {
				return err
			}
			if !confirmed {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Canceled.")
				return nil
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.DeleteProject(args[0]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "deleted project "+args[0])
			return nil
		},
	}
	cmd.Flags().Bool("force", false, "Skip the confirmation prompt")
	return cmd
}
