package issue

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newIssueListCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Search issues by query",
		RunE: func(cmd *cobra.Command, args []string) error {
			query, _ := cmd.Flags().GetString("query")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			issues, err := client.ListIssues(query, limit)
			if err != nil {
				return err
			}
			if len(issues) == 0 {
				output.PrintTip(cmd, "No issues found — try a different --query or check your permissions")
			}
			return output.Print(cmd, issues)
		},
	}
	cmd.Flags().String("query", "", "YouTrack query string (required)")
	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum number of issues to return")
	internal.MustMarkRequired(cmd, "query")
	return cmd
}
