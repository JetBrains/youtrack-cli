package instance

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/cmd/auth"
)

func newUpdateTokenCmd() *cobra.Command {
	cmd := auth.NewRefreshTokenCmd()
	cmd.Use = "update-token <alias>"
	cmd.Short = "Replace the saved token for an instance"
	cmd.Args = cobra.ExactArgs(1)
	return cmd
}
