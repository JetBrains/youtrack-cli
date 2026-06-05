package user

import "github.com/spf13/cobra"

func newUserViewCmd() *cobra.Command {
	cmd := newUserGetCmd()
	cmd.Use = "view <login>"
	cmd.Short = "View user profile"
	return cmd
}
