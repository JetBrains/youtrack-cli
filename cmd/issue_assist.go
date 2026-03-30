package cmd

import (
	"github.com/JetBrains/youtrack-cli/cmd/output"
	"github.com/spf13/cobra"
)

func newIssueAssistCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "assist <query> <issueId>...",
		Short: "Get command suggestions for issues",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
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
