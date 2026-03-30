package cmd

import (
	"fmt"

	"github.com/JetBrains/youtrack-cli/cmd/output"
	"github.com/spf13/cobra"
)

func newGroupCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "group", Short: "Manage user groups"}
	cmd.AddCommand(newGroupListCmd(), newGroupGetCmd(), newGroupAddCmd(), newGroupDeleteCmd())
	return cmd
}

func newGroupListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "list", Short: "List groups",
		RunE: func(cmd *cobra.Command, args []string) error {
			query, _ := cmd.Flags().GetString("query")
			client, err := getClient(cmd)
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
		Use: "get <name>", Short: "Get a group by name",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
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
		Use: "add", Short: "Create a new group",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			client, err := getClient(cmd)
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
	cmd.MarkFlagRequired("name")
	return cmd
}

func newGroupDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use: "delete <id>", Short: "Delete a group by ID",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			if err := client.DeleteGroup(args[0]); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "deleted group "+args[0])
			return nil
		},
	}
}
