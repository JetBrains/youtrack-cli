package issue

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newIssueSuggestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "suggest <issueId[,issueId...]> <query>",
		Short: "Get command suggestions for issues",
		Example: `  youtrack issue suggest FOO-42 "State "
  youtrack issue suggest FOO-1,FOO-2 "assignee "`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueIDs := strings.Split(args[0], ",")
			for i, id := range issueIDs {
				issueIDs[i] = strings.TrimSpace(id)
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			result, err := client.GetCommandAssist(args[1], issueIDs)
			if err != nil {
				return err
			}
			return output.Print(cmd, result)
		},
	}
}
