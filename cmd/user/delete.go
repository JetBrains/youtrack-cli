package user

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
)

func newUserDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <userId>",
		Short: "Delete a user account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			successor, _ := cmd.Flags().GetString("successor")

			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.DeleteUser(args[0], successor); err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "deleted user "+args[0])
			return err
		},
	}
	cmd.Flags().String("successor", "", "Login or id of the user that inherits references from the deleted user")
	internal.MustMarkRequired(cmd, "successor")
	return cmd
}
