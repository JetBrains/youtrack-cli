package issue

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newIssueAssistCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "assist <query> <issueId>...",
		Short: "Apply AI-assisted command suggestions to issues",
		Long: `Apply YouTrack AI command suggestions to one or more issues.
Unlike 'suggest', which only shows candidate commands, 'assist' executes them.

Example:
  youtrack issue assist "State In Progress" FOO-42
  youtrack issue assist "Priority Critical assignee me" FOO-1 FOO-2`,
		Args: cobra.MinimumNArgs(2),
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
