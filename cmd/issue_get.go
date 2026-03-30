package cmd

import (
	"github.com/JetBrains/youtrack-cli/cmd/output"

	"github.com/spf13/cobra"
)

func newIssueGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <issueId>",
		Short: "Get issue by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			issue, err := client.GetIssue(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, issue)
		},
	}
}
