package instance

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/cmd/auth"
)

// newLoginCmd is "auth login" under the instance group.
func newLoginCmd() *cobra.Command {
	cmd := auth.NewGetTokenCmd()
	cmd.Use = "login [url-or-alias]"
	cmd.Short = "Sign in to an instance and save its token (same as auth login)"
	return cmd
}
