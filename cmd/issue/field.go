package issue

import (
	"errors"

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
		Use:   "set <issueId> <fieldName> [value]",
		Short: "Set a custom field value on an issue",
		Long: "Set a custom field value on an issue.\n\n" +
			"Pass the value as the third argument, or repeat --value to set several values of a multi-value field;\n" +
			"the --value list replaces the field's current values. --clear empties the field.",
		Example: "  youtrack issue field set FOO-42 Priority Critical\n" +
			"  youtrack issue field set FOO-42 Subsystem --value \"Platform. Auth & Licensing\" --value UI\n" +
			"  youtrack issue field set FOO-42 Assignee --clear",
		Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			fieldType, _ := cmd.Flags().GetString("type")
			values, _ := cmd.Flags().GetStringArray("value")
			clearField, _ := cmd.Flags().GetBool("clear")
			given := 0
			for _, set := range []bool{len(args) == 3, len(values) > 0, clearField} {
				if set {
					given++
				}
			}
			if given != 1 {
				return errors.New("pass exactly one of a value argument, --value, or --clear")
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if len(args) == 3 {
				values = []string{args[2]}
			}
			err = client.SetCustomFieldValues(args[0], args[1], values, fieldType)
			if err != nil {
				return err
			}
			action := "updated field "
			if clearField {
				action = "cleared field "
			}
			output.PrintSuccess(cmd, action+args[1])
			return nil
		},
	}
	cmd.Flags().String("type", "", "Field type: period, text, user, int, string (default: enum)")
	cmd.Flags().StringArray("value", nil, "Value to set; repeat for multi-value fields")
	cmd.Flags().Bool("clear", false, "Clear the field")
	return cmd
}
