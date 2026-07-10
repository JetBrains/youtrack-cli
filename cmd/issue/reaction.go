package issue

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
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
			output.PrintSuccess(cmd, "added reaction "+args[2])
			return nil
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
			output.PrintSuccess(cmd, "deleted reaction "+args[2])
			return nil
		},
	}
}
