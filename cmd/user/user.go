package user

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage users",
	}
	internal.RequireSubcommand(cmd)

	cmd.AddGroup(
		&cobra.Group{ID: "work", Title: "Work with users:"},
		&cobra.Group{ID: "state", Title: "Account state:"},
		&cobra.Group{ID: "profile", Title: "Profile & preferences:"},
	)

	internal.AddToGroup(cmd, "work",
		newUserListCmd(), newUserViewCmd(), newUserCurrentCmd(),
		newUserCreateCmd(), newUserDeleteCmd(),
	)
	internal.AddToGroup(cmd, "state",
		newUserBanCmd(), newUserUnbanCmd(),
	)
	internal.AddToGroup(cmd, "profile",
		newUserProfileCmd(), newUserTimezoneCmd(), newUserLocaleCmd(),
		newUserTagCmd(), newUserDraftsCmd(),
	)

	return cmd
}
