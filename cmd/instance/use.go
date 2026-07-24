package instance

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal/config"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <alias>",
		Short: "Switch the active instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			alias := args[0]
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if _, ok := cfg.Instances[alias]; !ok {
				return fmt.Errorf(
					"instance %q not found — run: youtrack instance list\nKnown: %s",
					alias, strings.Join(cfg.Aliases(), ", "),
				)
			}
			cfg.CurrentInstance = alias
			if err := cfg.Save(); err != nil {
				return err
			}
			output.PrintSuccess(cmd, fmt.Sprintf("switched to instance %q", alias))
			return nil
		},
	}
}
