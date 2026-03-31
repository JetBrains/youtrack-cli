package issue

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/cmd/output"
	"github.com/JetBrains/youtrack-cli/internal"
)

func newIssueWorkitemCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "workitem", Short: "Log work on an issue"}
	add := &cobra.Command{
		Use: "add <issueId>", Short: "Add a work item to an issue",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			minutes, _ := cmd.Flags().GetInt("minutes")
			text, _ := cmd.Flags().GetString("text")
			typeID, _ := cmd.Flags().GetString("type")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			item, err := client.AddWorkItem(args[0], minutes, text, typeID)
			if err != nil {
				return err
			}
			return output.Print(cmd, item)
		},
	}
	add.Flags().Int("minutes", 0, "Duration in minutes (required)")
	add.Flags().String("text", "", "Work item description")
	add.Flags().String("type", "", "Work item type ID")
	add.MarkFlagRequired("minutes")
	cmd.AddCommand(add)
	return cmd
}
