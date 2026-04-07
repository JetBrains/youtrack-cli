package project

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
)

func newProjectWorkflowCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "workflow", Short: "Manage project workflows"}
	add := &cobra.Command{
		Use: "add <shortName>", Short: "Attach a workflow to a project",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workflowID, _ := cmd.Flags().GetString("workflow")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.AttachWorkflowToProject(args[0], workflowID); err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "attached workflow "+workflowID+" to project "+args[0])
			return err
		},
	}
	add.Flags().String("workflow", "", "Workflow ID (required)")
	internal.MustMarkRequired(add, "workflow")
	cmd.AddCommand(add)
	return cmd
}
