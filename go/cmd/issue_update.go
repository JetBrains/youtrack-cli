package cmd

import (
	"youtrack-cli/api"
	"youtrack-cli/cmd/output"

	"github.com/spf13/cobra"
)

func newIssueUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <issueId>",
		Short: "Update issue summary or description",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			summary, _ := cmd.Flags().GetString("summary")
			description, _ := cmd.Flags().GetString("description")

			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			issue, err := client.UpdateIssue(args[0], api.UpdateIssueRequest{
				Summary:     summary,
				Description: description,
			})
			if err != nil {
				return err
			}
			return output.Print(cmd, issue)
		},
	}
	cmd.Flags().String("summary", "", "New summary")
	cmd.Flags().String("description", "", "New description")
	return cmd
}
