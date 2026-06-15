package issue

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
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
	add.Flags().Int("minutes", 0, "Work duration in minutes, e.g. 90 for 1h30m (required)")
	add.Flags().String("text", "", "Work item description")
	add.Flags().String("type", "", "Work item type ID")
	internal.MustMarkRequired(add, "minutes")
	cmd.AddCommand(add)
	getCmd := &cobra.Command{
		Use: "get <issueId> <workItemId>", Short: "Get a work item by ID",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			item, err := client.GetWorkItem(args[0], args[1])
			if err != nil {
				return err
			}
			return output.Print(cmd, item)
		},
	}
	cmd.AddCommand(getCmd)
	updateCmd := &cobra.Command{
		Use: "update <issueId> <workItemId>", Short: "Update a work item",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			minutes, _ := cmd.Flags().GetInt("minutes")
			text, _ := cmd.Flags().GetString("text")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			item, err := client.UpdateWorkItem(args[0], args[1], minutes, text)
			if err != nil {
				return err
			}
			return output.Print(cmd, item)
		},
	}
	updateCmd.Flags().Int("minutes", 0, "New work duration in minutes")
	updateCmd.Flags().String("text", "", "New work item description")
	cmd.AddCommand(updateCmd)
	return cmd
}
