package issue

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newIssueViewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "view <issueId>",
		Aliases: []string{"get"},
		Short:   "View an issue by ID",
		Example: "  youtrack issue view FOO-42\n  youtrack issue view FOO-42 --comments 5\n  youtrack issue view FOO-42 -f idReadable,summary,reporter.login",
		Args:    cobra.ExactArgs(1),
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
				fields, _ := cmd.Root().PersistentFlags().GetString("fields")
				if fields == "" {
					output.PrintIssueDetail(cmd.OutOrStdout(), issue)
				} else if err := output.Print(cmd, issue); err != nil {
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
			}
			issue, err := client.GetIssue(args[0])
			if err != nil {
				return err
			}
			n, _ := cmd.Flags().GetInt("comments")
			if output.Format(cmd) == "json" && n > 0 {
				comments, err := client.ListRecentComments(args[0], n)
				if err != nil {
					return err
				}
				data, err := json.Marshal(issue)
				if err != nil {
					return err
				}
				var result map[string]any
				if err := json.Unmarshal(data, &result); err != nil {
					return err
				}
				result["comments"] = comments
				return output.Print(cmd, result)
			}
			if err := output.Print(cmd, issue); err != nil {
				return err
			}
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
