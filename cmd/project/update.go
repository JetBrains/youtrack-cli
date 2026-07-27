package project

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newProjectUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <shortName>",
		Short: "Update project metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("shortName") && !cmd.Flags().Changed("name") &&
				!cmd.Flags().Changed("description") && !cmd.Flags().Changed("leader") &&
				!cmd.Flags().Changed("archived") {
				return fmt.Errorf("at least one of --shortName, --name, --description, --leader, or --archived must be set")
			}

			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			req := api.UpdateProjectRequest{}
			if cmd.Flags().Changed("shortName") {
				value, _ := cmd.Flags().GetString("shortName")
				req.ShortName = &value
			}
			if cmd.Flags().Changed("name") {
				value, _ := cmd.Flags().GetString("name")
				req.Name = &value
			}
			if cmd.Flags().Changed("description") {
				value, _ := cmd.Flags().GetString("description")
				req.Description = &value
			}
			if cmd.Flags().Changed("leader") {
				login, _ := cmd.Flags().GetString("leader")
				leader, err := client.GetUser(login)
				if err != nil {
					return err
				}
				req.Leader = &api.User{ID: leader.ID}
			}
			if cmd.Flags().Changed("archived") {
				value, _ := cmd.Flags().GetBool("archived")
				req.Archived = &value
			}
			project, err := client.UpdateProject(args[0], req)
			if err != nil {
				return err
			}
			return output.Print(cmd, project)
		},
	}
	cmd.Flags().String("shortName", "", "New project short name")
	cmd.Flags().String("name", "", "New project name")
	cmd.Flags().String("description", "", "New project description (empty clears it)")
	cmd.Flags().String("leader", "", "New project leader login")
	cmd.Flags().Bool("archived", false, "Archive or unarchive the project")
	return cmd
}
