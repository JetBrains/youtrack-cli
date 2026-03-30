package cmd

import (
	"github.com/JetBrains/youtrack-cli/cmd/output"
	"github.com/spf13/cobra"
)

func newProjectFieldCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "field", Short: "Manage project custom fields"}
	list := &cobra.Command{
		Use: "list <shortName>", Short: "List custom fields in a project",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
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
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			return client.AddProjectCustomField(args[0], fieldID)
		},
	}
	add.Flags().String("field", "", "Custom field ID (required)")
	add.MarkFlagRequired("field")
	cmd.AddCommand(list, add)
	return cmd
}
