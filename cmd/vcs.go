package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newVCSCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "vcs", Short: "Manage VCS integrations"}
	internal.RequireSubcommand(cmd)
	cmd.AddCommand(newVCSListCmd(), newVCSDeleteCmd(), newVCSConfigsCmd(), newVCSProcessorCmd())
	return cmd
}

func newVCSListCmd() *cobra.Command {
	return &cobra.Command{
		Use: "list", Short: "List VCS hosting servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			servers, err := client.ListVCSServers()
			if err != nil {
				return err
			}
			return output.Print(cmd, servers)
		},
	}
}

func newVCSDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "delete <id>", Short: "Delete a VCS server",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if abort, err := internal.ConfirmOrAbort(cmd, fmt.Sprintf("Delete VCS server %s?", args[0])); abort {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.DeleteVCSServer(args[0]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "deleted VCS server "+args[0])
			return nil
		},
	}
	cmd.Flags().Bool("force", false, "Skip the confirmation prompt")
	return cmd
}

func newVCSConfigsCmd() *cobra.Command {
	return &cobra.Command{
		Use: "configs <serverId>", Short: "List VCS configurations for a server",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			configs, err := client.GetVCSConfigurations(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, configs)
		},
	}
}

func newVCSProcessorCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "processor", Short: "Manage VCS processors"}
	internal.RequireSubcommand(cmd)
	list := &cobra.Command{
		Use: "list", Short: "List VCS processors",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			procs, err := client.ListVCSProcessors()
			if err != nil {
				return err
			}
			return output.Print(cmd, procs)
		},
	}
	del := &cobra.Command{
		Use: "delete <id>", Short: "Delete a VCS processor",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if abort, err := internal.ConfirmOrAbort(cmd, fmt.Sprintf("Delete VCS processor %s?", args[0])); abort {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.DeleteVCSProcessor(args[0]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "deleted VCS processor "+args[0])
			return nil
		},
	}
	del.Flags().Bool("force", false, "Skip the confirmation prompt")
	cmd.AddCommand(list, del)
	return cmd
}
