package instance

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/cmd/auth"
)

func newAddCmd() *cobra.Command {
	cmd := auth.NewGetTokenCmd()
	cmd.Use = "add [url-or-alias]"
	cmd.Short = "Add a new instance (same as auth login)"
	return cmd
}
