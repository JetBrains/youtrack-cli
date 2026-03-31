package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal/output"
	"github.com/JetBrains/youtrack-cli/internal"
)

func newRoleCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "role", Short: "Manage roles and permissions"}
	cmd.AddCommand(
		newRoleListCmd(),
		newRoleAddCmd(),
		newRoleDeleteCmd(),
		newRoleUpdateCmd(),
		newRoleGrantCmd(),
		newRolePermissionsCmd(),
	)
	return cmd
}

func newRoleListCmd() *cobra.Command {
	return &cobra.Command{
		Use: "list", Short: "List all roles",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			roles, err := client.ListRoles()
			if err != nil {
				return err
			}
			return output.Print(cmd, roles)
		},
	}
}

func newRoleAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "add", Short: "Create a new role",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			description, _ := cmd.Flags().GetString("description")
			permStr, _ := cmd.Flags().GetString("permissions")
			var permIDs []string
			if permStr != "" {
				permIDs = strings.Split(permStr, ",")
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			role, err := client.AddRole(name, description, permIDs)
			if err != nil {
				return err
			}
			return output.Print(cmd, role)
		},
	}
	cmd.Flags().String("name", "", "Role name (required)")
	cmd.Flags().String("description", "", "Role description")
	cmd.Flags().String("permissions", "", "Comma-separated permission IDs")
	cmd.MarkFlagRequired("name")
	return cmd
}

func newRoleDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use: "delete <id>", Short: "Delete a role by ID",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.DeleteRole(args[0]); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "deleted role "+args[0])
			return nil
		},
	}
}

func newRoleUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "update <id>", Short: "Update role permissions",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			permStr, _ := cmd.Flags().GetString("permissions")
			permIDs := strings.Split(permStr, ",")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.UpdateRole(args[0], permIDs)
		},
	}
	cmd.Flags().String("permissions", "", "Comma-separated permission IDs (required)")
	cmd.MarkFlagRequired("permissions")
	return cmd
}

func newRoleGrantCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "grant", Short: "Grant a role to a user or group",
		RunE: func(cmd *cobra.Command, args []string) error {
			roleID, _ := cmd.Flags().GetString("role")
			userID, _ := cmd.Flags().GetString("user")
			groupID, _ := cmd.Flags().GetString("group")
			projectID, _ := cmd.Flags().GetString("project")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if userID != "" {
				return client.GrantRoleToUser(userID, roleID, projectID)
			}
			if groupID != "" {
				return client.GrantRoleToGroup(groupID, roleID, projectID)
			}
			return fmt.Errorf("specify --user or --group")
		},
	}
	cmd.Flags().String("role", "", "Role ID (required)")
	cmd.Flags().String("user", "", "User ID to grant role to")
	cmd.Flags().String("group", "", "Group ID to grant role to")
	cmd.Flags().String("project", "", "Scope to this project ID (optional)")
	cmd.MarkFlagRequired("role")
	return cmd
}

func newRolePermissionsCmd() *cobra.Command {
	return &cobra.Command{
		Use: "permissions", Short: "List available permissions",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			perms, err := client.GetPermissions()
			if err != nil {
				return err
			}
			return output.Print(cmd, perms)
		},
	}
}
