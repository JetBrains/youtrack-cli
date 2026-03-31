package project

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
)

func newProjectTeamCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "team", Short: "Manage project team members"}
	addUser := &cobra.Command{
		Use: "add-user <shortName> <userId>", Short: "Add a user to the project team",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.AddUserToProjectTeam(args[0], args[1]); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "added user "+args[1]+" to project "+args[0])
			return nil
		},
	}
	addGroup := &cobra.Command{
		Use: "add-group <shortName> <groupId>", Short: "Add a group to the project team",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.AddGroupToProjectTeam(args[0], args[1]); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "added group "+args[1]+" to project "+args[0])
			return nil
		},
	}
	cmd.AddCommand(addUser, addGroup)
	return cmd
}
