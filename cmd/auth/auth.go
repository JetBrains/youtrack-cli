package auth

import "github.com/spf13/cobra"

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication utilities",
	}
	cmd.AddCommand(NewGetTokenCmd())
	cmd.AddCommand(newLoginCmd())
	cmd.AddCommand(NewRefreshTokenCmd())
	cmd.AddCommand(newLogoutCmd())
	cmd.AddCommand(newStatusCmd())
	return cmd
}

func newLoginCmd() *cobra.Command {
	c := NewGetTokenCmd()
	c.Use = "login"
	c.Short = "Authenticate with a YouTrack instance (alias: get-token)"
	return c
}
