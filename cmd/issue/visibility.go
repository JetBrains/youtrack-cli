package issue

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newIssueVisibilityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "visibility <issueId>",
		Short: "Set visibility on an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			groups, _ := cmd.Flags().GetStringSlice("groups")
			users, _ := cmd.Flags().GetStringSlice("users")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.UpdateIssueVisibility(args[0], groups, users); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "updated visibility for "+args[0])
			return nil
		},
	}
	cmd.Flags().StringSlice("groups", nil, "Permitted group IDs (comma-separated)")
	cmd.Flags().StringSlice("users", nil, "Permitted user IDs (comma-separated)")
	return cmd
}
