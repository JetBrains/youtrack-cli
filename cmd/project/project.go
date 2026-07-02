package project

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
	}

	cmd.AddGroup(
		&cobra.Group{ID: "work", Title: "Work with projects:"},
		&cobra.Group{ID: "config", Title: "Configure:"},
	)

	internal.AddToGroup(cmd, "work",
		newProjectListCmd(), newProjectViewCmd(), newProjectCreateCmd(),
		newProjectDeleteCmd(), newProjectArchiveCmd(),
	)
	internal.AddToGroup(cmd, "config",
		newProjectFieldCmd(), newProjectTeamCmd(), newProjectTemplateCmd(),
		newProjectWorkflowCmd(), newProjectTimetrackCmd(),
	)

	return cmd
}
