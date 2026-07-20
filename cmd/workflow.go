package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newWorkflowCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "workflow", Short: "Manage workflows"}
	internal.RequireSubcommand(cmd)
	cmd.AddCommand(newWorkflowListCmd(), newWorkflowCreateCmd(), newWorkflowDeleteCmd(), newWorkflowRestoreCmd(), newWorkflowRuleCmd())
	return cmd
}

func newWorkflowListCmd() *cobra.Command {
	return &cobra.Command{
		Use: "list", Short: "List all workflows",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			workflows, err := client.ListWorkflows()
			if err != nil {
				return err
			}
			return output.Print(cmd, workflows)
		},
	}
}

func newWorkflowCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "create", Aliases: []string{"add"}, Short: "Create a new workflow",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			title, _ := cmd.Flags().GetString("title")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			workflow, err := client.AddWorkflow(name, title)
			if err != nil {
				return err
			}
			return output.Print(cmd, workflow)
		},
	}
	cmd.Flags().String("name", "", "Workflow name (required)")
	cmd.Flags().String("title", "", "Workflow title")
	internal.MustMarkRequired(cmd, "name")
	return cmd
}

func newWorkflowDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use: "delete <id>", Short: "Delete a workflow",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.DeleteWorkflow(args[0]); err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "deleted workflow "+args[0])
			return err
		},
	}
}

func newWorkflowRestoreCmd() *cobra.Command {
	return &cobra.Command{
		Use: "restore", Short: "Restore default workflows",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.RestoreDefaultWorkflows()
		},
	}
}

func newWorkflowRuleCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "rule", Short: "Manage workflow rules"}
	internal.RequireSubcommand(cmd)
	create := &cobra.Command{
		Use: "create <workflowId>", Aliases: []string{"add"}, Short: "Create a rule in a workflow",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ruleType, _ := cmd.Flags().GetString("type")
			name, _ := cmd.Flags().GetString("name")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.AddWorkflowRule(args[0], ruleType, name); err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "created rule "+name)
			return err
		},
	}
	create.Flags().String("type", "", "Rule type (required)")
	create.Flags().String("name", "", "Rule name (required)")
	internal.MustMarkRequired(create, "type", "name")
	cmd.AddCommand(create)
	return cmd
}
