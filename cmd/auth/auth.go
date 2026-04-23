package auth

import "github.com/spf13/cobra"

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication utilities",
	}
	cmd.AddCommand(newGetTokenCmd())
	cmd.AddCommand(newStatusCmd())
	return cmd
}
