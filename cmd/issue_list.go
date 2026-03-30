package cmd

import (
	"github.com/JetBrains/youtrack-cli/cmd/output"

	"github.com/spf13/cobra"
)

func newIssueListCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Search issues by query",
		RunE: func(cmd *cobra.Command, args []string) error {
			query, _ := cmd.Flags().GetString("query")
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			issues, err := client.ListIssues(query, limit)
			if err != nil {
				return err
			}
			return output.Print(cmd, issues)
		},
	}
	cmd.Flags().String("query", "", "YouTrack query string (required)")
	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum number of issues to return")
	cmd.MarkFlagRequired("query")
	return cmd
}
