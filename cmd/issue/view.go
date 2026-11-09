package issue

import (
	"encoding/json"
	"errors"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newIssueViewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "view <issueId>",
		Aliases: []string{"get"},
		Short:   "View an issue by ID",
		Long: "View an issue by ID.\n\n" +
			"--fields adds to the default projection; pass --only-fields to fetch just the listed fields instead.\n" +
			"--no-description leaves the description out of the default projection.",
		Example: "  youtrack issue view FOO-42\n  youtrack issue view FOO-42 --comments 5\n  youtrack issue view FOO-42 -f idReadable,summary,reporter.login\n" +
			"  youtrack issue view FOO-42 --no-description --json\n  youtrack issue view FOO-42 --only-fields -f idReadable,summary --json",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			noDescription, _ := cmd.Flags().GetBool("no-description")
			onlyFields, _ := cmd.Flags().GetBool("only-fields")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			projection := api.IssueViewFields(false, !noDescription)
			if onlyFields {
				if client.Fields == "" {
					return errors.New("--only-fields requires --fields")
				}
				// Clearing Client.Fields keeps it from being merged into this and later requests again.
				projection, client.Fields = client.Fields, ""
			}
			if output.Format(cmd) == "plain" && !onlyFields {
				issue, err := client.GetIssueProjection(args[0], api.IssueViewFields(true, !noDescription))
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
					return printComments(cmd, comments)
				}
				return nil
			}
			issue, err := client.GetIssueProjection(args[0], projection)
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
				return printComments(cmd, comments)
			}
			return nil
		},
	}
	cmd.Flags().Int("comments", 0, "Show N most recent comments")
	cmd.Flags().Bool("no-description", false, "Leave the description out of the output")
	cmd.Flags().Bool("only-fields", false, "Fetch only the fields listed in --fields instead of adding them to the default projection")
	cmd.MarkFlagsMutuallyExclusive("no-description", "only-fields")
	return cmd
}

// printComments ignores --fields, which projects the issue rather than its comments.
func printComments(cmd *cobra.Command, comments []api.Comment) error {
	if output.Format(cmd) == "plain" {
		return output.PrintPlain(cmd.OutOrStdout(), comments)
	}
	return output.PrintTable(cmd.OutOrStdout(), comments)
}
