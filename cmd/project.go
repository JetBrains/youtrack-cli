package cmd

import "github.com/spf13/cobra"

func newProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
	}
	cmd.AddCommand(newProjectListCmd())
	cmd.AddCommand(newProjectGetCmd())
	cmd.AddCommand(newProjectAddCmd())
	cmd.AddCommand(newProjectDeleteCmd())
	return cmd
}
