package issue

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/cmd/output"
	"github.com/JetBrains/youtrack-cli/internal"
)

func newIssueAssistCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "assist <query> <issueId>...",
		Short: "Get command suggestions for issues",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			result, err := client.GetCommandAssist(args[0], args[1:])
			if err != nil {
				return err
			}
			return output.Print(cmd, result)
		},
	}
	return cmd
}
