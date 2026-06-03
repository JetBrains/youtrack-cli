package instance

import "github.com/spf13/cobra"

// NewCmd constructs the root "instance" command.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instance",
		Short: "Manage configured YouTrack instances",
	}
	cmd.AddCommand(newAddCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newUseCmd())
	cmd.AddCommand(newRemoveCmd())
	cmd.AddCommand(newUpdateTokenCmd())
	return cmd
}
