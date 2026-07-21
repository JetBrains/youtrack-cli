package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newAppCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "app",
		Short: "Manage apps",
	}
	internal.RequireSubcommand(cmd)
	cmd.AddCommand(newAppListCmd())
	cmd.AddCommand(newAppDeleteCmd())
	cmd.AddCommand(newAppDeactivateCmd())
	cmd.AddCommand(newAppPluginCmd())
	return cmd
}

func newAppListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all apps",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			apps, err := client.ListApps()
			if err != nil {
				return err
			}
			return output.Print(cmd, apps)
		},
	}
}

func newAppDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an app by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if abort, err := internal.ConfirmOrAbort(cmd, fmt.Sprintf("Delete app %s?", args[0])); abort {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.DeleteApp(args[0])
		},
	}
	cmd.Flags().Bool("force", false, "Skip the confirmation prompt")
	return cmd
}

func newAppDeactivateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "deactivate <id>",
		Short: "Deactivate an app by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			return client.DeactivateApp(args[0])
		},
	}
}

func newAppPluginCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "plugin", Short: "Manage plugins"}
	internal.RequireSubcommand(cmd)
	list := &cobra.Command{
		Use: "list", Short: "List plugins",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			plugins, err := client.ListPlugins()
			if err != nil {
				return err
			}
			return output.Print(cmd, plugins)
		},
	}
	del := &cobra.Command{
		Use: "delete <id>", Short: "Delete a plugin",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if abort, err := internal.ConfirmOrAbort(cmd, fmt.Sprintf("Delete plugin %s?", args[0])); abort {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.DeletePlugin(args[0]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "deleted plugin "+args[0])
			return nil
		},
	}
	del.Flags().Bool("force", false, "Skip the confirmation prompt")
	cmd.AddCommand(list, del)
	return cmd
}
