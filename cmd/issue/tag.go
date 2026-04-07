package issue

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
)

func newIssueTagCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tag <issueId> <tagId>",
		Short: "Tag an issue",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.TagIssue(args[0], args[1]); err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "tagged issue "+args[0]+" with "+args[1])
			return err
		},
	}
}
