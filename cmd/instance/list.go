package instance

import (
	"fmt"
	"strconv"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/config"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

type instanceListItem struct {
	Alias    string `json:"alias"`
	URL      string `json:"url"`
	Login    string `json:"login,omitempty"`
	FullName string `json:"fullName,omitempty"`
	Status   string `json:"status"`
	Active   bool   `json:"active"`
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configured instances",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			items := make([]instanceListItem, 0, len(cfg.Instances))
			w := cmd.OutOrStdout()
			for _, alias := range cfg.Aliases() {
				if err := cmd.Context().Err(); err != nil {
					return err
				}
				inst := cfg.Instances[alias]
				item := instanceListItem{Alias: alias, URL: inst.URL, Active: alias == cfg.CurrentInstance}
				// OAuth sessions refresh on demand, so a session near expiry is not misreported as unavailable.
				user, err := internal.NewSavedInstanceClient(cmd, alias, &inst).GetCurrentUser()
				if err != nil {
					if contextErr := cmd.Context().Err(); contextErr != nil {
						return contextErr
					}
					item.Status = "unavailable"
				} else {
					item.Status = "authenticated"
					item.Login = user.Login
					item.FullName = user.FullName
				}
				items = append(items, item)
			}
			switch output.Format(cmd) {
			case "json":
				return output.Print(cmd, items)
			case "plain":
				for _, item := range items {
					_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
						item.Alias, item.URL, item.Login, item.Status, strconv.FormatBool(item.Active))
				}
				return nil
			}
			if len(items) == 0 {
				_, _ = fmt.Fprintln(w, "No instances configured. Run: youtrack auth get-token")
				return nil
			}
			for _, item := range items {
				activeMarker := ""
				if item.Active {
					activeMarker = " " + color.New(color.FgGreen).Sprint("[active]")
				}
				if item.Status != "authenticated" {
					_, _ = fmt.Fprintf(w, "  %s %-20s %s%s\n",
						color.New(color.FgRed).Sprint("✗"), item.Alias, item.URL, activeMarker)
				} else {
					_, _ = fmt.Fprintf(w, "  %s %-20s %s  %s (%s)%s\n",
						color.New(color.FgGreen).Sprint("✓"), item.Alias, item.URL, item.FullName, item.Login, activeMarker)
				}
			}
			return nil
		},
	}
}
