package cmd

import (
	"youtrack-cli/api"
	"youtrack-cli/cmd/output"

	"github.com/spf13/cobra"
)

func newUserAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a new user",
		RunE: func(cmd *cobra.Command, args []string) error {
			login, _ := cmd.Flags().GetString("login")
			fullName, _ := cmd.Flags().GetString("fullName")
			email, _ := cmd.Flags().GetString("email")

			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			user, err := client.AddUser(api.AddUserRequest{
				Login:    login,
				FullName: fullName,
				Email:    email,
			})
			if err != nil {
				return err
			}
			return output.Print(cmd, user)
		},
	}
	cmd.Flags().String("login", "", "User login (required)")
	cmd.Flags().String("fullName", "", "User full name (required)")
	cmd.Flags().String("email", "", "User email (required)")
	cmd.MarkFlagRequired("login")
	cmd.MarkFlagRequired("fullName")
	cmd.MarkFlagRequired("email")
	return cmd
}
