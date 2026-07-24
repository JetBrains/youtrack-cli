package project

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newProjectFieldCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "field", Short: "Manage project custom fields"}
	internal.RequireSubcommand(cmd)
	list := &cobra.Command{
		Use: "list <shortName>", Short: "List custom fields in a project",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			fields, err := client.GetProjectCustomFields(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, fields)
		},
	}
	add := &cobra.Command{
		Use: "add <shortName>", Short: "Add a custom field to a project",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fieldID, _ := cmd.Flags().GetString("field")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.AddProjectCustomField(args[0], fieldID)
		},
	}
	add.Flags().String("field", "", "Custom field ID (required)")
	internal.MustMarkRequired(add, "field")
	remove := &cobra.Command{
		Use: "remove <shortName> <projectFieldId>", Short: "Remove a custom field from a project",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if abort, err := internal.ConfirmOrAbort(cmd, fmt.Sprintf(
				"Remove project field %s from project %s?", args[1], args[0])); abort {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.RemoveProjectCustomField(args[0], args[1]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "removed project field "+args[1]+" from project "+args[0])
			return nil
		},
	}
	remove.Flags().Bool("force", false, "Skip the confirmation prompt")
	cmd.AddCommand(list, add, remove)
	return cmd
}
