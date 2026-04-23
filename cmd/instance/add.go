package instance

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/cmd/auth"
)

func newAddCmd() *cobra.Command {
	cmd := auth.NewGetTokenCmd()
	cmd.Use = "add"
	cmd.Short = "Add a new instance (authenticate and save credentials)"

	// instance add always prompts for a URL — adding an instance implies a new target.
	// Wrap RunE to clear YOUTRACK_BASE_URL unless --base-url was explicitly provided,
	// so the interactive URL prompt always fires.
	wrapped := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if !cmd.Root().PersistentFlags().Changed("base-url") {
			prev := os.Getenv("YOUTRACK_BASE_URL")
			_ = os.Unsetenv("YOUTRACK_BASE_URL")
			defer func() {
				if prev != "" {
					_ = os.Setenv("YOUTRACK_BASE_URL", prev)
				}
			}()
		}
		return wrapped(cmd, args)
	}
	return cmd
}
