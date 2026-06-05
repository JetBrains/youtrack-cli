package issue

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newIssueVoteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "vote <issueId>",
		Short: "Vote for an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.VoteIssue(args[0]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "voted for "+args[0])
			return nil
		},
	}
}

func newIssueUnvoteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unvote <issueId>",
		Short: "Remove vote from an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.UnvoteIssue(args[0]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "removed vote from "+args[0])
			return nil
		},
	}
}
