package cmd

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newMailboxCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "mailbox", Short: "Manage mailbox integrations"}
	cmd.AddCommand(newMailboxListCmd(), newMailboxAddCmd(), newMailboxDeleteCmd(), newMailboxRuleCmd())
	return cmd
}

func newMailboxListCmd() *cobra.Command {
	return &cobra.Command{
		Use: "list", Short: "List mailbox servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			servers, err := client.ListMailboxServers()
			if err != nil {
				return err
			}
			return output.Print(cmd, servers)
		},
	}
}

func newMailboxAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "add", Short: "Add a mailbox server",
		RunE: func(cmd *cobra.Command, args []string) error {
			protocol, _ := cmd.Flags().GetString("protocol")
			host, _ := cmd.Flags().GetString("host")
			port, _ := cmd.Flags().GetInt("port")
			login, _ := cmd.Flags().GetString("login")
			password, _ := cmd.Flags().GetString("password")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			server, err := client.AddMailboxServer(api.AddMailboxServerRequest{
				Protocol: protocol, Host: host, Port: port, Login: login, Password: password,
			})
			if err != nil {
				return err
			}
			return output.Print(cmd, server)
		},
	}
	cmd.Flags().String("protocol", "IMAP", "Protocol (required)")
	cmd.Flags().String("host", "", "Host (required)")
	cmd.Flags().Int("port", 993, "Port")
	cmd.Flags().String("login", "", "Login (required)")
	cmd.Flags().String("password", "", "Password (required)")
	internal.MustMarkRequired(cmd, "host", "login", "password")
	return cmd
}

func newMailboxDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use: "delete <id>", Short: "Delete a mailbox server",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.DeleteMailboxServer(args[0]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "deleted mailbox server "+args[0])
			return nil
		},
	}
}

func newMailboxRuleCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "rule", Short: "Manage mailbox rules"}
	add := &cobra.Command{
		Use: "add", Short: "Add a mailbox rule",
		RunE: func(cmd *cobra.Command, args []string) error {
			mailboxID, _ := cmd.Flags().GetString("mailbox")
			projectID, _ := cmd.Flags().GetString("project")
			name, _ := cmd.Flags().GetString("name")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.AddMailboxRule(mailboxID, projectID, name); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "added mailbox rule "+name)
			return nil
		},
	}
	add.Flags().String("mailbox", "", "Mailbox ID (required)")
	add.Flags().String("project", "", "Project ID (required)")
	add.Flags().String("name", "", "Rule name (required)")
	internal.MustMarkRequired(add, "mailbox", "project", "name")
	cmd.AddCommand(add)
	return cmd
}
