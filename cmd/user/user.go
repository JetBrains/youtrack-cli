package user

import "github.com/spf13/cobra"

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage users",
	}
	cmd.AddCommand(newUserListCmd())
	cmd.AddCommand(newUserGetCmd())
	cmd.AddCommand(newUserCurrentCmd())
	cmd.AddCommand(newUserAddCmd())
	cmd.AddCommand(newUserDeleteCmd())
	cmd.AddCommand(newUserBanCmd())
	cmd.AddCommand(newUserUnbanCmd())
	cmd.AddCommand(newUserTimezoneCmd())
	cmd.AddCommand(newUserLocaleCmd())
	cmd.AddCommand(newUserDraftsCmd())
	cmd.AddCommand(newUserProfileCmd())
	cmd.AddCommand(newUserPinProjectCmd())
	return cmd
}
