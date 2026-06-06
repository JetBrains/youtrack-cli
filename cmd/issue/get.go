package issue

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newIssueGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <issueId>",
		Short: "Get issue by ID (alias: view)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if output.Format(cmd) == "plain" {
				issue, err := client.GetIssueDetail(args[0])
				if err != nil {
					return err
				}
				output.PrintIssueDetail(cmd.OutOrStdout(), issue)
				n, _ := cmd.Flags().GetInt("comments")
				if n > 0 {
					comments, err := client.ListRecentComments(args[0], n)
					if err != nil {
						return err
					}
					return output.Print(cmd, comments)
				}
				return nil
			}
			issue, err := client.GetIssue(args[0])
			if err != nil {
				return err
			}
			if err := output.Print(cmd, issue); err != nil {
				return err
			}
			n, _ := cmd.Flags().GetInt("comments")
			if n > 0 {
				comments, err := client.ListRecentComments(args[0], n)
				if err != nil {
					return err
				}
				return output.Print(cmd, comments)
			}
			return nil
		},
	}
	cmd.Flags().Int("comments", 0, "Show N most recent comments")
	return cmd
}
