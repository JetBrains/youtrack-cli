package issue

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newIssueFieldCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "field", Short: "Manage custom field values on an issue"}
	internal.RequireSubcommand(cmd)
	cmd.AddCommand(newIssueFieldViewCmd())
	cmd.AddCommand(newIssueFieldSetCmd())
	return cmd
}

func newIssueFieldViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "view <issueId> <fieldName>",
		Aliases: []string{"get"},
		Short:   "View a custom field value on an issue",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			field, err := client.GetCustomFieldValue(args[0], args[1])
			if err != nil {
				return err
			}
			return output.Print(cmd, field)
		},
	}
}

func newIssueFieldSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <issueId> <fieldName> <value>",
		Short: "Set a custom field value on an issue",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			fieldType, _ := cmd.Flags().GetString("type")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.SetCustomField(args[0], args[1], args[2], fieldType); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "updated field "+args[1])
			return nil
		},
	}
	cmd.Flags().String("type", "", "Field type: period, text, user, int, string (default: enum)")
	return cmd
}
