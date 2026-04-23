package user

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newUserPinProjectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pin-project <userId> <projectId>",
		Short: "Pin a project for a user",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := internal.GetClient(cmd)
			if err != nil {
				return err
			}
			if err := client.PinProjectForUser(args[0], args[1]); err != nil {
				return err
			}
			output.PrintSuccess(cmd, "pinned project "+args[1]+" for user "+args[0])
			return nil
		},
	}
}
