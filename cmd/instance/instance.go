package instance

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
)

// NewCmd constructs the root "instance" command.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instance",
		Short: "Manage configured YouTrack instances",
	}
	internal.RequireSubcommand(cmd)
	cmd.AddCommand(newAddCmd())
	cmd.AddCommand(newLoginCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newUseCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newRefreshTokenCmd())
	return cmd
}
