package auth

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication utilities",
	}
	internal.RequireSubcommand(cmd)
	cmd.AddCommand(newLoginCmd())
	cmd.AddCommand(NewRefreshTokenCmd())
	cmd.AddCommand(newLogoutCmd())
	cmd.AddCommand(newStatusCmd())
	return cmd
}

func newLoginCmd() *cobra.Command {
	c := NewGetTokenCmd()
	c.Use = "login [url-or-alias]"
	c.Aliases = []string{"get-token"}
	c.Short = "Authenticate with a YouTrack instance"
	return c
}
