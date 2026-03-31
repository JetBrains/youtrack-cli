package user

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal/output"
	"github.com/JetBrains/youtrack-cli/internal"
)

func newUserCurrentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Get the currently authenticated user",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			user, err := client.GetCurrentUser()
			if err != nil {
				return err
			}
			return output.Print(cmd, user)
		},
	}
}
