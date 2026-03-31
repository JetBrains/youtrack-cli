package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/cmd/output"
	"github.com/JetBrains/youtrack-cli/internal"
)

func newVCSCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "vcs", Short: "Manage VCS integrations"}
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
	return &cobra.Command{
		Use: "delete <id>", Short: "Delete a VCS server",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.DeleteVCSServer(args[0]); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "deleted VCS server "+args[0])
			return nil
		},
	}
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
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.DeleteVCSProcessor(args[0]); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "deleted VCS processor "+args[0])
			return nil
		},
	}
	cmd.AddCommand(list, del)
	return cmd
}
