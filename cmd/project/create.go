package project

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newProjectCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"add"},
		Short:   "Create a new project",
		Example: "  youtrack project create --shortName FOO --name \"Foo Project\"\n  youtrack project create --shortName FOO --name Foo --leader john.doe",
		RunE: func(cmd *cobra.Command, args []string) error {
			shortName, _ := cmd.Flags().GetString("shortName")
			name, _ := cmd.Flags().GetString("name")
			leader, _ := cmd.Flags().GetString("leader")
			template, _ := cmd.Flags().GetString("template")

			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}

			// Default leader to current user if not specified
			if leader == "" {
				me, err := client.GetCurrentUser()
				if err != nil {
					return err
				}
				leader = me.Login
			}

			leaderUser, err := client.GetUser(leader)
			if err != nil {
				return err
			}

			project, err := client.AddProject(api.AddProjectRequest{
				ShortName: shortName,
				Name:      name,
				Leader:    &api.User{ID: leaderUser.ID},
			}, template)
			if err != nil {
				return err
			}
			return output.Print(cmd, project)
		},
	}
	cmd.Flags().String("shortName", "", "Project short name / key (required)")
	cmd.Flags().String("name", "", "Project full name (required)")
	cmd.Flags().String("leader", "", "Leader login (defaults to current user)")
	cmd.Flags().String("template", "", "Project template: default (empty/omit), helpdesk, taskManagement, scrum, kanban, projectManagement, demo, or marketing")
	internal.MustMarkRequired(cmd, "shortName", "name")
	return cmd
}
