package cmd

import (
	"github.com/JetBrains/youtrack-cli/cmd/output"

	"github.com/spf13/cobra"
)

func newUserCurrentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Get the currently authenticated user",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
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
