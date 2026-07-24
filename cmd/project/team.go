package project

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newProjectTeamCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "team", Short: "Manage project team members"}
	internal.RequireSubcommand(cmd)
	addUser := &cobra.Command{
		Use: "add-user <shortName> <userId>", Short: "Add a user to the project team",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			registeredAsAgent, err := client.AddUserToProjectTeam(args[0], args[1])
			if err != nil {
				return err
			}
			output.PrintSuccess(cmd, "added user "+args[1]+" to project "+args[0])
			if registeredAsAgent {
				output.PrintSuccess(cmd, "registered user "+args[1]+" as a Helpdesk agent of project "+args[0])
			}
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
			output.PrintSuccess(cmd, "added group "+args[1]+" to project "+args[0])
			return nil
		},
	}
	listUsers := &cobra.Command{
		Use: "list-users <shortName>", Short: "List users in the project team",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			users, err := client.ListProjectTeamUsers(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, users)
		},
	}
	listGroups := &cobra.Command{
		Use: "list-groups <shortName>", Short: "List groups in the project team",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			groups, err := client.ListProjectTeamGroups(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, groups)
		},
	}
	removeUser := &cobra.Command{
		Use: "remove-user <shortName> <userId>", Short: "Remove a user from the project team",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if abort, err := internal.ConfirmOrAbort(cmd, fmt.Sprintf(
				"Remove user %s from project %s?", args[1], args[0])); abort {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.RemoveUserFromProjectTeam(args[0], args[1]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "removed user "+args[1]+" from project "+args[0])
			return nil
		},
	}
	removeUser.Flags().Bool("force", false, "Skip the confirmation prompt")
	removeGroup := &cobra.Command{
		Use: "remove-group <shortName> <groupId>", Short: "Remove a group from the project team",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if abort, err := internal.ConfirmOrAbort(cmd, fmt.Sprintf(
				"Remove group %s from project %s?", args[1], args[0])); abort {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.RemoveGroupFromProjectTeam(args[0], args[1]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "removed group "+args[1]+" from project "+args[0])
			return nil
		},
	}
	removeGroup.Flags().Bool("force", false, "Skip the confirmation prompt")
	cmd.AddCommand(addUser, addGroup, listUsers, listGroups, removeUser, removeGroup)
	return cmd
}
