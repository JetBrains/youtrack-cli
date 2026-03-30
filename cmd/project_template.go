package cmd

import (
	"fmt"

	"github.com/JetBrains/youtrack-cli/cmd/output"
	"github.com/spf13/cobra"
)

func newProjectTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "template", Short: "Manage project templates"}
	list := &cobra.Command{
		Use: "list", Short: "List project templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			templates, err := client.GetProjectTemplates()
			if err != nil {
				return err
			}
			return output.Print(cmd, templates)
		},
	}
	convert := &cobra.Command{
		Use: "convert <shortName>", Short: "Convert a project to a template",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			if err := client.ConvertProjectToTemplate(args[0]); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "converted project "+args[0]+" to template")
			return nil
		},
	}
	cmd.AddCommand(list, convert)
	return cmd
}
