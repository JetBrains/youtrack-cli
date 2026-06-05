package project

import "github.com/spf13/cobra"

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
	}
	cmd.AddCommand(newProjectListCmd())
	cmd.AddCommand(newProjectGetCmd())
	cmd.AddCommand(newProjectViewCmd())
	cmd.AddCommand(newProjectAddCmd())
	cmd.AddCommand(newProjectCreateCmd())
	cmd.AddCommand(newProjectDeleteCmd())
	cmd.AddCommand(newProjectArchiveCmd())
	cmd.AddCommand(newProjectTemplateCmd())
	cmd.AddCommand(newProjectTeamCmd())
	cmd.AddCommand(newProjectFieldCmd())
	cmd.AddCommand(newProjectWorkflowCmd())
	cmd.AddCommand(newProjectTimetrackCmd())
	return cmd
}
