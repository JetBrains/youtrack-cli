package cmd

import "github.com/spf13/cobra"

func newIssueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue",
		Short: "Manage issues",
	}
	cmd.AddCommand(newIssueGetCmd())
	cmd.AddCommand(newIssueListCmd())
	cmd.AddCommand(newIssueAddCmd())
	cmd.AddCommand(newIssueUpdateCmd())
	cmd.AddCommand(newIssueDeleteCmd())
	cmd.AddCommand(newIssueCommentCmd())
	cmd.AddCommand(newIssueCommandCmd())
	return cmd
}
