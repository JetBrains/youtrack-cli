package issue

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newIssueGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <issueId>",
		Short: "Get issue by ID (alias: view)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
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
