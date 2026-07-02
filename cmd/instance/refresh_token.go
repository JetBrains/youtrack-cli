package instance

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/cmd/auth"
)

func newRefreshTokenCmd() *cobra.Command {
	cmd := auth.NewRefreshTokenCmd()
	cmd.Use = "refresh-token <alias>"
	cmd.Aliases = []string{"update-token"}
	cmd.Short = "Replace the saved token for an instance"
	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("alias is required")
		}
		if len(args) > 1 {
			return fmt.Errorf("accepts 1 alias, received %d", len(args))
		}
		return nil
	}
	cmd.Long = `Replaces the saved token for a configured instance using any available
authentication method:

  1) Basic auth   — login + password via Hub API
  2) Web browser  — open the token page in a browser, then paste the token
  3) Paste token  — enter a token directly without opening a browser

The instance URL is read from the config — no URL prompt. On success the
config entry is updated in-place (alias and URL are preserved).

Use --format json for machine-readable output.`
	return cmd
}
