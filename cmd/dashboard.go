package cmd

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newDashboardCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "dashboard", Short: "Manage dashboards"}
	internal.RequireSubcommand(cmd)
	cmd.AddCommand(
		newDashboardListCmd(),
		newDashboardCreateCmd(),
		newDashboardDeleteCmd(),
		newDashboardUpdateCmd(),
	)
	return cmd
}

func newDashboardListCmd() *cobra.Command {
	return &cobra.Command{
		Use: "list", Short: "List all dashboards",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			dashboards, err := client.ListDashboards()
			if err != nil {
				return err
			}
			return output.Print(cmd, dashboards)
		},
	}
}

func newDashboardCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "create", Aliases: []string{"add"}, Short: "Create a new dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			dashboard, err := client.AddDashboard(name)
			if err != nil {
				return err
			}
			return output.Print(cmd, dashboard)
		},
	}
	cmd.Flags().String("name", "", "Dashboard name (required)")
	internal.MustMarkRequired(cmd, "name")
	return cmd
}

func newDashboardDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use: "delete <id>", Short: "Delete a dashboard",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.DeleteDashboard(args[0]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "deleted dashboard "+args[0])
			return nil
		},
	}
}

func newDashboardUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "update <id>", Short: "Rename a dashboard",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			dashboard, err := client.UpdateDashboard(args[0], name)
			if err != nil {
				return err
			}
			return output.Print(cmd, dashboard)
		},
	}
	cmd.Flags().String("name", "", "New name (required)")
	internal.MustMarkRequired(cmd, "name")
	return cmd
}
