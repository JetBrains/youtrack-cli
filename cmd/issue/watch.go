package issue

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newIssueWatchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "watch <issueId>",
		Short: "Watch an issue (receive notifications)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.WatchIssue(args[0]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "watching "+args[0])
			return nil
		},
	}
}

func newIssueUnwatchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unwatch <issueId>",
		Short: "Stop watching an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.UnwatchIssue(args[0]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "unwatched "+args[0])
			return nil
		},
	}
}
