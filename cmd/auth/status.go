package auth

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/config"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

// AuthStatus is the JSON output of the auth status command.
type AuthStatus struct {
	URL      string `json:"url"`
	Login    string `json:"login"`
	FullName string `json:"fullName"`
	Status   string `json:"status"`
}

// InstanceStatus is one row in multi-instance JSON output from auth status.
type InstanceStatus struct {
	Alias    string `json:"alias"`
	URL      string `json:"url"`
	Login    string `json:"login,omitempty"`
	FullName string `json:"fullName,omitempty"`
	Status   string `json:"status"`
	Active   bool   `json:"active"`
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current authentication status",
		Long: `Check whether the CLI is authenticated and display the current user's info.

This command always exits 0; it is an informational probe, not a gate.
Check the "status" field for the authentication result:
  "authenticated"       — token is valid
  "invalid token: ..."  — token rejected by the server
  "not configured: ..." — YOUTRACK_BASE_URL or YOUTRACK_TOKEN is not set

Use --json for machine-readable output.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			// Multi-instance path: config exists with at least one instance.
			// Skip if the caller explicitly overrode the URL or token via flags —
			// explicit flags mean "check this specific instance", not all instances.
			if len(cfg.Instances) > 0 && !flagChanged(cmd, "base-url") && !flagChanged(cmd, "token") {
				return runMultiInstanceStatus(cmd, cfg)
			}

			// Single-instance legacy path: read from flags / env vars.
			baseURL, _ := cmd.Root().PersistentFlags().GetString("base-url")
			if baseURL == "" {
				baseURL = os.Getenv("YOUTRACK_BASE_URL")
			}
			client, err := internal.GetClient(cmd)
			if err != nil {
				s := AuthStatus{
					URL:    baseURL,
					Status: fmt.Sprintf("not configured: %s", err.Error()),
				}
				if !jsonOutput(cmd) {
					printAuthStatus(cmd, s)
					return nil
				}
				return output.Print(cmd, s)
			}
			user, err := client.GetCurrentUser()
			if err != nil {
				s := AuthStatus{
					URL:    client.BaseURL,
					Status: fmt.Sprintf("invalid token: %s", err.Error()),
				}
				if !jsonOutput(cmd) {
					printAuthStatus(cmd, s)
					return nil
				}
				return output.Print(cmd, s)
			}
			s := AuthStatus{
				URL:      client.BaseURL,
				Login:    user.Login,
				FullName: user.FullName,
				Status:   "authenticated",
			}
			if !jsonOutput(cmd) {
				printAuthStatus(cmd, s)
				return nil
			}
			return output.Print(cmd, s)
		},
	}
}

func printAuthStatus(cmd *cobra.Command, s AuthStatus) {
	w := cmd.OutOrStdout()
	switch {
	case s.Status == "authenticated":
		_, _ = fmt.Fprintf(w, "%s Authenticated\n", color.New(color.FgGreen).Sprint("✓"))
		_, _ = fmt.Fprintf(w, "  URL:   %s\n", s.URL)
		_, _ = fmt.Fprintf(w, "  User:  %s (%s)\n", s.FullName, s.Login)
	case strings.HasPrefix(s.Status, "invalid token"):
		_, _ = fmt.Fprintf(w, "%s Token invalid\n", color.New(color.FgRed).Sprint("✗"))
		_, _ = fmt.Fprintf(w, "  URL:   %s\n", s.URL)
		_, _ = fmt.Fprintf(w, "  Error: %s\n", strings.TrimPrefix(s.Status, "invalid token: "))
	default:
		_, _ = fmt.Fprintf(w, "%s Not configured\n", color.New(color.FgYellow).Sprint("!"))
		_, _ = fmt.Fprintf(w, "  Run:   youtrack auth get-token\n")
	}
}

// runMultiInstanceStatus checks auth for every configured instance and prints results.
func runMultiInstanceStatus(cmd *cobra.Command, cfg *config.Config) error {
	aliases := cfg.Aliases()
	statuses := make([]InstanceStatus, 0, len(aliases))
	for _, alias := range aliases {
		inst := cfg.Instances[alias]
		s := InstanceStatus{
			Alias:  alias,
			URL:    inst.URL,
			Active: alias == cfg.CurrentInstance,
		}
		c := api.NewClient(inst.URL, inst.Token)
		user, err := c.GetCurrentUser()
		if err != nil {
			s.Status = fmt.Sprintf("invalid token: %s", err.Error())
		} else {
			s.Status = "authenticated"
			s.Login = user.Login
			s.FullName = user.FullName
		}
		statuses = append(statuses, s)
	}
	if !jsonOutput(cmd) {
		printMultiInstanceStatus(cmd, statuses)
		return nil
	}
	return output.Print(cmd, statuses)
}

func printMultiInstanceStatus(cmd *cobra.Command, statuses []InstanceStatus) {
	w := cmd.OutOrStdout()
	for _, s := range statuses {
		activeMarker := ""
		if s.Active {
			activeMarker = " " + color.New(color.FgGreen).Sprint("[active]")
		}
		switch s.Status {
		case "authenticated":
			_, _ = fmt.Fprintf(w, "  %s %-20s %s  %s (%s)%s\n",
				color.New(color.FgGreen).Sprint("✓"),
				s.Alias, s.URL, s.FullName, s.Login, activeMarker)
		default:
			_, _ = fmt.Fprintf(w, "  %s %-20s %s  %s%s\n",
				color.New(color.FgRed).Sprint("✗"),
				s.Alias, s.URL, s.Status, activeMarker)
		}
	}
}
