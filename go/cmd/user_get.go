package cmd

import (
	"youtrack-cli/cmd/output"

	"github.com/spf13/cobra"
)

func newUserGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <login>",
		Short: "Get user by login",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			user, err := client.GetUser(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, user)
		},
	}
}
