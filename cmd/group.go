package cmd

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newGroupCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "group", Short: "Manage user groups"}
	cmd.AddCommand(newGroupListCmd(), newGroupGetCmd(), newGroupViewCmd(), newGroupAddCmd(), newGroupCreateCmd(), newGroupDeleteCmd())
	return cmd
}

func newGroupViewCmd() *cobra.Command {
	c := newGroupGetCmd()
	c.Use = "view <name>"
	c.Short = "View a group by name"
	return c
}

func newGroupCreateCmd() *cobra.Command {
	c := newGroupAddCmd()
	c.Use = "create"
	c.Short = "Create a new group"
	return c
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

func newGroupGetCmd() *cobra.Command {
	return &cobra.Command{
		Use: "get <name>", Short: "Get a group by name (alias: view)",
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

func newGroupAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "add", Short: "Create a new group (alias: create)",
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
	cmd.Flags().String("successor", "", "ID or name of the group that inherits references from the deleted group (optional)")
	return cmd
}
