package issue

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newIssueWorkitemCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "workitem", Short: "Log work on an issue"}
	internal.RequireSubcommand(cmd)
	create := &cobra.Command{
		Use: "create <issueId>", Aliases: []string{"add"}, Short: "Create a work item on an issue",
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
	create.Flags().Int("minutes", 0, "Work duration in minutes, e.g. 90 for 1h30m (required)")
	create.Flags().String("text", "", "Work item description")
	create.Flags().String("type", "", "Work item type ID")
	internal.MustMarkRequired(create, "minutes")
	cmd.AddCommand(create)
	viewCmd := &cobra.Command{
		Use: "view <issueId> <workItemId>", Aliases: []string{"get"}, Short: "View a work item by ID",
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
	cmd.AddCommand(viewCmd)
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
