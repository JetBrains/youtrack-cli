package instance

import (
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/cmd/auth"
)

func newLoginCmd() *cobra.Command {
	return auth.NewSSOLoginCmd()
}
