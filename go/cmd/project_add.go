package cmd

import (
	"youtrack-cli/api"
	"youtrack-cli/cmd/output"

	"github.com/spf13/cobra"
)

func newProjectAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a new project",
		RunE: func(cmd *cobra.Command, args []string) error {
			shortName, _ := cmd.Flags().GetString("shortName")
			name, _ := cmd.Flags().GetString("name")
			leader, _ := cmd.Flags().GetString("leader")

			client, err := getClient(cmd)
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
			})
			if err != nil {
				return err
			}
			return output.Print(cmd, project)
		},
	}
	cmd.Flags().String("shortName", "", "Project short name / key (required)")
	cmd.Flags().String("name", "", "Project full name (required)")
	cmd.Flags().String("leader", "", "Leader login (defaults to current user)")
	cmd.MarkFlagRequired("shortName")
	cmd.MarkFlagRequired("name")
	return cmd
}
