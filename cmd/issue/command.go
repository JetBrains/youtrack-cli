package issue

import (
	"fmt"
	"strings"

	"github.com/JetBrains/youtrack-cli/internal"

	"github.com/spf13/cobra"
)

func newIssueCommandCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "command <issueId[,issueId...]> <query>",
		Short: "Apply a YouTrack command to one or more issues",
		Example: `  youtrack issue command FOO-42 "State Fixed"
  youtrack issue command FOO-1,FOO-2 "assignee alice Priority Critical"`,
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
			if err := client.ApplyCommand(args[1], issueIDs); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "command applied to "+args[0])
			return nil
		},
	}
}
