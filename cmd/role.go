package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newRoleCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "role", Short: "Manage roles and permissions"}
	internal.RequireSubcommand(cmd)
	cmd.AddCommand(
		newRoleListCmd(),
		newRoleViewCmd(),
		newRoleCreateCmd(),
		newRoleDeleteCmd(),
		newRoleUpdateCmd(),
		newRoleGrantCmd(),
		newRoleRevokeCmd(),
		newRoleAssignmentsCmd(),
		newRolePermissionsCmd(),
		newRolePermissionCmd(),
	)
	return cmd
}

func newRoleViewCmd() *cobra.Command {
	return &cobra.Command{
		Use: "view <id>", Aliases: []string{"get"}, Short: "View a role by ID",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			role, err := client.GetRole(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, role)
		},
	}
}

func newRolePermissionCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "permission", Short: "Manage permissions on a role"}
	internal.RequireSubcommand(cmd)
	list := &cobra.Command{
		Use: "list <roleId>", Short: "List permissions for a role",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			role, err := client.GetRole(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, role.Permissions)
		},
	}
	add := &cobra.Command{
		Use: "add <roleId> <permId>", Short: "Add a permission to a role",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.AddRolePermission(args[0], args[1]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "added permission "+args[1]+" to role "+args[0])
			return nil
		},
	}
	remove := &cobra.Command{
		Use: "remove <roleId> <permId>", Short: "Remove a permission from a role",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if abort, err := internal.ConfirmOrAbort(cmd, fmt.Sprintf(
				"Remove permission %s from role %s?", args[1], args[0])); abort {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.RemoveRolePermission(args[0], args[1]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "removed permission "+args[1]+" from role "+args[0])
			return nil
		},
	}
	remove.Flags().Bool("force", false, "Skip the confirmation prompt")
	cmd.AddCommand(list, add, remove)
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

func newRoleCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "create", Aliases: []string{"add"}, Short: "Create a new role",
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
	internal.MustMarkRequired(cmd, "name")
	return cmd
}

func newRoleDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "delete <id>", Short: "Delete a role by ID",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if abort, err := internal.ConfirmOrAbort(cmd, fmt.Sprintf("Delete role %s?", args[0])); abort {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.DeleteRole(args[0]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "deleted role "+args[0])
			return nil
		},
	}
	cmd.Flags().Bool("force", false, "Skip the confirmation prompt")
	return cmd
}

func newRoleUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "update <id>", Short: "Update role permissions",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if abort, err := internal.ConfirmOrAbort(cmd, fmt.Sprintf(
				"Replace all permissions on role %s?", args[0])); abort {
				return err
			}
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
	cmd.Flags().Bool("force", false, "Skip the confirmation prompt")
	internal.MustMarkRequired(cmd, "permissions")
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
	internal.MustMarkRequired(cmd, "role")
	return cmd
}

func newRoleRevokeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "revoke <assignedRoleID>", Short: "Revoke a role assignment by ID",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if abort, err := internal.ConfirmOrAbort(cmd, fmt.Sprintf(
				"Revoke role assignment %s?", args[0])); abort {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.RevokeRole(args[0]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "revoked role assignment "+args[0])
			return nil
		},
	}
	cmd.Flags().Bool("force", false, "Skip the confirmation prompt")
	return cmd
}

func newRoleAssignmentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "assignments", Short: "List role assignments, optionally filtered by a holder ID",
		Long: "List role assignments. --holder accepts any holder ID: user, group, team, or organization. " +
			"--user and --group are aliases kept for readability; the YouTrack API does not distinguish them.",
		RunE: func(cmd *cobra.Command, args []string) error {
			holderID, _ := cmd.Flags().GetString("holder")
			userID, _ := cmd.Flags().GetString("user")
			groupID, _ := cmd.Flags().GetString("group")
			specified := 0
			for _, v := range []string{holderID, userID, groupID} {
				if v != "" {
					specified++
				}
			}
			if specified > 1 {
				return fmt.Errorf("specify only one of --holder, --user, or --group")
			}
			if holderID == "" {
				if userID != "" {
					holderID = userID
				} else {
					holderID = groupID
				}
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			rows, err := client.ListAssignedRoles(holderID)
			if err != nil {
				return err
			}
			return output.Print(cmd, rows)
		},
	}
	cmd.Flags().String("holder", "", "Filter by holder ID (user, group, team, or organization)")
	cmd.Flags().String("user", "", "Alias for --holder")
	cmd.Flags().String("group", "", "Alias for --holder")
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
