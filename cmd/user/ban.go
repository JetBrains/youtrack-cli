package user

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newUserBanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ban <userId>",
		Short: "Ban a user account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if abort, err := internal.ConfirmOrAbort(cmd, fmt.Sprintf("Ban user %s?", args[0])); abort {
				return err
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.BanUser(args[0], true); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "banned user "+args[0])
			return nil
		},
	}
	cmd.Flags().Bool("force", false, "Skip the confirmation prompt")
	return cmd
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
			output.PrintSuccess(cmd, "unbanned user "+args[0])
			return nil
		},
	}
}
