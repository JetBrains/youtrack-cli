package project

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newProjectWorkflowCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "workflow", Short: "Manage project workflows"}
	internal.RequireSubcommand(cmd)
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
			output.PrintSuccess(cmd, "attached workflow "+workflowID+" to project "+args[0])
			return nil
		},
	}
	add.Flags().String("workflow", "", "Workflow ID (required)")
	internal.MustMarkRequired(add, "workflow")
	cmd.AddCommand(add)
	return cmd
}
