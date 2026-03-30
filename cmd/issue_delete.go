package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newIssueDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <issueId>",
		Short: "Delete an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			if err := client.DeleteIssue(args[0]); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "deleted "+args[0])
			return nil
		},
	}
}
