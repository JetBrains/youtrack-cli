package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newUserDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <userId>",
		Short: "Delete a user account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd)
			if err != nil {
				return err
			}
			if err := client.DeleteUser(args[0]); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "deleted user "+args[0])
			return nil
		},
	}
}
