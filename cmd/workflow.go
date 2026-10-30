package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
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
	var name string
	cmd := &cobra.Command{
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
			return output.Print(cmd, filterWorkflowsByName(workflows, name))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Show only workflows whose name contains this text (case-insensitive)")
	return cmd
}

func filterWorkflowsByName(workflows []api.Workflow, substr string) []api.Workflow {
	if substr == "" {
		return workflows
	}
	needle := strings.ToLower(substr)
	matched := []api.Workflow{}
	for _, wf := range workflows {
		if strings.Contains(strings.ToLower(wf.Name), needle) {
			matched = append(matched, wf)
		}
	}
	return matched
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
	cmd := &cobra.Command{
		Use: "delete <id>", Short: "Delete a workflow",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if abort, err := internal.ConfirmOrAbort(cmd, fmt.Sprintf("Delete workflow %s?", args[0])); abort {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.DeleteWorkflow(args[0]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "deleted workflow "+args[0])
			return nil
		},
	}
	cmd.Flags().Bool("force", false, "Skip the confirmation prompt")
	return cmd
}

func newWorkflowRestoreCmd() *cobra.Command {
	return &cobra.Command{
		Use: "restore", Short: "Restore default workflows",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.RestoreDefaultWorkflows(); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "restored default workflows")
			return nil
		},
	}
}

func newWorkflowRuleCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "rule", Short: "Manage workflow rules"}
	internal.RequireSubcommand(cmd)
	cmd.AddCommand(newWorkflowRuleCreateCmd(), newWorkflowRuleViewCmd(), newWorkflowRuleUpdateCmd())
	return cmd
}

func newWorkflowRuleCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "create <workflowId>", Aliases: []string{"add"}, Short: "Create a rule in a workflow",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ruleType, _ := cmd.Flags().GetString("type")
			name, _ := cmd.Flags().GetString("name")
			scriptFile, _ := cmd.Flags().GetString("script-file")
			script, err := readRuleScript(cmd, scriptFile)
			if err != nil {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			rule, err := client.AddWorkflowRule(args[0], ruleType, name, script)
			if err != nil {
				return err
			}
			return output.Print(cmd, rule)
		},
	}
	cmd.Flags().String("type", "WorkflowRule", "Rule entity type sent as $type; YouTrack determines the resulting type from the script")
	cmd.Flags().String("name", "", "Rule name: lowercase letters, digits, -, _, . and ~ (required)")
	cmd.Flags().String("script-file", "", "Path to the rule's JavaScript source, or - for stdin (required)")
	internal.MustMarkRequired(cmd, "name", "script-file")
	return cmd
}

func newWorkflowRuleViewCmd() *cobra.Command {
	return &cobra.Command{
		Use: "view <workflowId> <ruleId>", Short: "View a workflow rule, including its script",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			rule, err := client.GetWorkflowRule(args[0], args[1])
			if err != nil {
				return err
			}
			return output.Print(cmd, rule)
		},
	}
}

func newWorkflowRuleUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <workflowId> <ruleId>",
		Short: "Replace a workflow rule's script and verify it was stored unchanged",
		Long: `Replace a workflow rule's script, then read the rule back and compare the stored
script with the one that was sent. The command exits non-zero when they differ.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			scriptFile, _ := cmd.Flags().GetString("script-file")
			script, err := readRuleScript(cmd, scriptFile)
			if err != nil {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			workflowID, ruleID := args[0], args[1]
			if err := client.UpdateWorkflowRuleScript(workflowID, ruleID, script); err != nil {
				return err
			}
			stored, err := client.GetWorkflowRule(workflowID, ruleID)
			if err != nil {
				return fmt.Errorf("updated rule %s but could not read it back to verify: %w", ruleID, err)
			}
			if script != stored.Script {
				return fmt.Errorf("rule %s script read back after update does not match the script sent: sent %d bytes, stored %d bytes",
					ruleID, len(script), len(stored.Script))
			}
			output.PrintSuccess(cmd, fmt.Sprintf("updated rule %s (%s) and verified the stored script", stored.Name, stored.ID))
			return nil
		},
	}
	cmd.Flags().String("script-file", "", "Path to the new JavaScript source, or - for stdin (required)")
	internal.MustMarkRequired(cmd, "script-file")
	return cmd
}

func readRuleScript(cmd *cobra.Command, path string) (string, error) {
	data, err := internal.ReadInput(cmd, path)
	if err != nil {
		return "", fmt.Errorf("--script-file: %w", err)
	}
	script := string(data)
	if strings.TrimSpace(script) == "" {
		return "", fmt.Errorf("--script-file: script is empty")
	}
	return script, nil
}
