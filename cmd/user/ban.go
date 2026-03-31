package user

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
)

func newUserBanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ban <userId>",
		Short: "Ban a user account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.BanUser(args[0], true); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "banned user "+args[0])
			return nil
		},
	}
}

func newUserUnbanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unban <userId>",
		Short: "Unban a user account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.BanUser(args[0], false); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "unbanned user "+args[0])
			return nil
		},
	}
}
