package instance

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal/config"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configured instances",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if len(cfg.Instances) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No instances configured. Run: youtrack auth get-token")
				return nil
			}
			aliases := cfg.Aliases()
			for _, alias := range aliases {
				inst := cfg.Instances[alias]
				active := ""
				if alias == cfg.CurrentInstance {
					active = " " + color.New(color.FgGreen).Sprint("[active]")
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %-20s %s%s\n", alias, inst.URL, active)
			}
			return nil
		},
	}
}
