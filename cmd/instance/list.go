package instance

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
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
			w := cmd.OutOrStdout()
			for _, alias := range cfg.Aliases() {
				inst := cfg.Instances[alias]
				activeMarker := ""
				if alias == cfg.CurrentInstance {
					activeMarker = " " + color.New(color.FgGreen).Sprint("[active]")
				}
				c := api.NewClient(inst.URL, inst.Token)
				user, err := c.GetCurrentUser()
				if err != nil {
					_, _ = fmt.Fprintf(w, "  %s %-20s %s%s\n",
						color.New(color.FgRed).Sprint("✗"), alias, inst.URL, activeMarker)
				} else {
					_, _ = fmt.Fprintf(w, "  %s %-20s %s  %s (%s)%s\n",
						color.New(color.FgGreen).Sprint("✓"), alias, inst.URL, user.FullName, user.Login, activeMarker)
				}
			}
			return nil
		},
	}
}
