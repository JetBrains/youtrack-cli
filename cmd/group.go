package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newGroupCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "group", Short: "Manage user groups"}
	internal.RequireSubcommand(cmd)
	cmd.AddCommand(newGroupListCmd(), newGroupViewCmd(), newGroupCreateCmd(), newGroupDeleteCmd(), newGroupMemberCmd(), newGroupRoleCmd())
	return cmd
}

func newGroupListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "list", Short: "List groups",
		RunE: func(cmd *cobra.Command, args []string) error {
			query, _ := cmd.Flags().GetString("query")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			groups, err := client.ListGroups(query)
			if err != nil {
				return err
			}
			return output.Print(cmd, groups)
		},
	}
	cmd.Flags().String("query", "", "Filter query")
	return cmd
}

func newGroupViewCmd() *cobra.Command {
	return &cobra.Command{
		Use: "view <name>", Aliases: []string{"get"}, Short: "View a group by name",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			group, err := client.GetGroup(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, group)
		},
	}
}

func newGroupCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "create", Aliases: []string{"add"}, Short: "Create a new group",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			group, err := client.AddGroup(name)
			if err != nil {
				return err
			}
			return output.Print(cmd, group)
		},
	}
	cmd.Flags().String("name", "", "Group name (required)")
	internal.MustMarkRequired(cmd, "name")
	return cmd
}

func newGroupDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "delete <group>", Short: "Delete a group by ID or name",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if abort, err := internal.ConfirmOrAbort(cmd, fmt.Sprintf("Delete group %s?", args[0])); abort {
				return err
			}
			successor, _ := cmd.Flags().GetString("successor")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.DeleteGroup(args[0], successor); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "deleted group "+args[0])
			return nil
		},
	}
	cmd.Flags().String("successor", "", "ID or name of the group that inherits references from the deleted group")
	cmd.Flags().Bool("force", false, "Skip the confirmation prompt")
	internal.MustMarkRequired(cmd, "successor")
	return cmd
}

func newGroupMemberCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "member", Short: "Manage group members"}
	internal.RequireSubcommand(cmd)
	list := &cobra.Command{
		Use: "list <group-id>", Short: "List members of a group",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			members, err := client.ListGroupMembers(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, members)
		},
	}
	cmd.AddCommand(list)
	return cmd
}

func newGroupRoleCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "role", Short: "Manage group project roles"}
	internal.RequireSubcommand(cmd)
	list := &cobra.Command{
		Use: "list <group-id>", Short: "List project roles assigned to a group",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			roles, err := client.ListGroupProjectRoles(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, roles)
		},
	}
	add := &cobra.Command{
		Use: "add <group-id>", Short: "Add a project role to a group",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			roleID, _ := cmd.Flags().GetString("role")
			projectID, _ := cmd.Flags().GetString("project")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.AddGroupProjectRole(args[0], roleID, projectID); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "role assigned to group")
			return nil
		},
	}
	add.Flags().String("role", "", "Role ID (required)")
	add.Flags().String("project", "", "Project ID (optional, for project-scoped role)")
	internal.MustMarkRequired(add, "role")
	cmd.AddCommand(list, add)
	return cmd
}
