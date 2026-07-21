package user

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newUserDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <userId>",
		Short: "Delete a user account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if abort, err := internal.ConfirmOrAbort(cmd, fmt.Sprintf("Delete user %s?", args[0])); abort {
				return err
			}
			successor, _ := cmd.Flags().GetString("successor")
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.DeleteUser(args[0], successor); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "deleted user "+args[0])
			return nil
		},
	}
	cmd.Flags().Bool("force", false, "Skip the confirmation prompt")
	cmd.Flags().String("successor", "", "Login or id of the user that inherits references from the deleted user")
	internal.MustMarkRequired(cmd, "successor")
	return cmd
}
