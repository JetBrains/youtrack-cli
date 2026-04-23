package instance

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/cmd/auth"
)

func newAddCmd() *cobra.Command {
	cmd := auth.NewGetTokenCmd()
	cmd.Use = "add"
	cmd.Short = "Add a new instance (authenticate and save credentials)"
	return cmd
}
