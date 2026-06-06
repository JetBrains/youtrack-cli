package issue

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newIssueAssignCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "assign <issueId> <login>",
		Short:   "Assign an issue to a user",
		Example: "  youtrack issue assign FOO-42 alice",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.ApplyCommand("assignee "+args[1], []string{args[0]}); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "assigned "+args[0]+" to "+args[1])
			return nil
		},
	}
}

func newIssueUnassignCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unassign <issueId>",
		Short: "Remove assignee from an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.ApplyCommand("assignee Unassigned", []string{args[0]}); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "unassigned "+args[0])
			return nil
		},
	}
}
