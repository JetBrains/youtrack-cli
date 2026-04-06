package issue

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
)

func newIssueReactionCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "reaction", Short: "Manage reactions on issue comments"}
	cmd.AddCommand(newIssueReactionAddCmd())
	cmd.AddCommand(newIssueReactionDeleteCmd())
	return cmd
}

func newIssueReactionAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <issueId> <commentId> <reaction>",
		Short: "Add a reaction to an issue comment",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.AddReaction(args[0], args[1], args[2]); err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "added reaction "+args[2])
			return err
		},
	}
}

func newIssueReactionDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <issueId> <commentId> <reactionId>",
		Short: "Delete a reaction from an issue comment",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.DeleteReaction(args[0], args[1], args[2]); err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "deleted reaction "+args[2])
			return err
		},
	}
}
