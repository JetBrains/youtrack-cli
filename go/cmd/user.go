package cmd

import "github.com/spf13/cobra"

func newUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage users",
	}
	cmd.AddCommand(newUserListCmd())
	cmd.AddCommand(newUserGetCmd())
	cmd.AddCommand(newUserCurrentCmd())
	cmd.AddCommand(newUserAddCmd())
	return cmd
}
