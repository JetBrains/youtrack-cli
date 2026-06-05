package user

import "github.com/spf13/cobra"

func newUserCreateCmd() *cobra.Command {
	cmd := newUserAddCmd()
	cmd.Use = "create"
	cmd.Short = "Create a new user"
	return cmd
}
