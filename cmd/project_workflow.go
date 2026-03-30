package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newProjectWorkflowCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "workflow", Short: "Manage project workflows"}
	add := &cobra.Command{
		Use: "add <shortName>", Short: "Attach a workflow to a project",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workflowID, _ := cmd.Flags().GetString("workflow")
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			if err := client.AttachWorkflowToProject(args[0], workflowID); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "attached workflow "+workflowID+" to project "+args[0])
			return nil
		},
	}
	add.Flags().String("workflow", "", "Workflow ID (required)")
	add.MarkFlagRequired("workflow")
	cmd.AddCommand(add)
	return cmd
}
