package cmd

import (
	"fmt"

	"github.com/JetBrains/youtrack-cli/cmd/output"
	"github.com/spf13/cobra"
)

func newIssueFieldCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "field", Short: "Manage custom field values on an issue"}
	cmd.AddCommand(newIssueFieldGetCmd())
	cmd.AddCommand(newIssueFieldSetCmd())
	return cmd
}

func newIssueFieldGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <issueId> <fieldName>",
		Short: "Get a custom field value on an issue",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
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
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			if err := client.SetCustomField(args[0], args[1], args[2], fieldType); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "updated field "+args[1])
			return nil
		},
	}
	cmd.Flags().String("type", "", "Field type: period, text, user, int, string (default: enum)")
	return cmd
}
