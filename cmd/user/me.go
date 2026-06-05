package user

import "github.com/spf13/cobra"

func newUserMeCmd() *cobra.Command {
	cmd := newUserCurrentCmd()
	cmd.Use = "me"
	cmd.Short = "Show the currently authenticated user"
	return cmd
}
