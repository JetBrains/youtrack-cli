package auth

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/config"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

// AuthStatus is the JSON output of the auth status command. It never contains a token.
type AuthStatus struct {
	URL      string `json:"url"`
	Login    string `json:"login"`
	FullName string `json:"fullName"`
	Status   string `json:"status"`
	// Source is where the token came from: "flag" (--token), "env" (YOUTRACK_TOKEN), or "profile" (saved instance).
	Source          string `json:"source,omitempty"`
	Alias           string `json:"alias,omitempty"`
	AuthMethod      string `json:"authMethod,omitempty"` // "token" (permanent) or "oauth"
	ExpiresAt       string `json:"expiresAt,omitempty"`  // RFC 3339; OAuth access token only
	HasRefreshToken bool   `json:"hasRefreshToken"`
}

// InstanceStatus is one row in multi-instance JSON output from auth status. It never contains a token.
type InstanceStatus struct {
	Alias           string `json:"alias"`
	URL             string `json:"url"`
	Login           string `json:"login,omitempty"`
	FullName        string `json:"fullName,omitempty"`
	Status          string `json:"status"`
	Active          bool   `json:"active"`
	AuthMethod      string `json:"authMethod"`
	ExpiresAt       string `json:"expiresAt,omitempty"`
	HasRefreshToken bool   `json:"hasRefreshToken"`
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current authentication status",
		Long: `Check whether the CLI is authenticated and display the current user's info,
the auth method (permanent token or OAuth session), the access-token expiry, and
whether a refresh token is stored. Tokens themselves are never printed. Checking
an OAuth session refreshes its access token when it is about to expire.

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
			if len(cfg.Instances) > 0 && !hasCredentialOverride(cmd) {
				return runMultiInstanceStatus(cmd, cfg)
			}

			// Single-instance legacy path: read from flags / env vars.
			baseURL, _ := cmd.Root().PersistentFlags().GetString("base-url")
			if baseURL == "" {
				baseURL = os.Getenv("YOUTRACK_BASE_URL")
			}
			client, src, err := internal.GetClientWithSource(cmd)
			if err != nil {
				s := AuthStatus{
					URL:    baseURL,
					Status: fmt.Sprintf("not configured: %s", err.Error()),
				}
				// A saved OAuth session whose tokens were cleared still names its profile.
				describeSource(&s, src)
				if !jsonOutput(cmd) {
					printAuthStatus(cmd, &s)
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
				// Keep the source and session metadata: a rejected refresh (invalid_grant) clears the tokens, not the profile.
				describeSource(&s, src)
				if !jsonOutput(cmd) {
					printAuthStatus(cmd, &s)
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
			describeSource(&s, src)
			if !jsonOutput(cmd) {
				printAuthStatus(cmd, &s)
				return nil
			}
			return output.Print(cmd, s)
		},
	}
}

// describeSource fills the auth-method fields; for a saved profile the config is re-read so a refresh done by this very check is reflected.
// An unknown source (no token was resolved) leaves them empty.
func describeSource(s *AuthStatus, src internal.CredentialSource) {
	if src.Kind == "" {
		return
	}
	s.Source, s.Alias, s.AuthMethod = src.Kind, src.Alias, "token"
	if src.Kind != "profile" {
		return
	}
	if cfg, err := config.Load(); err == nil {
		inst := cfg.Instances[src.Alias]
		s.AuthMethod, s.ExpiresAt, s.HasRefreshToken = authMethodOf(inst.AuthMethod), formatExpiry(inst.TokenExpiresAt), inst.RefreshToken != ""
	}
}

func authMethodOf(method string) string {
	if method == config.AuthMethodOAuth {
		return "oauth"
	}
	return "token"
}

func formatExpiry(unix int64) string {
	if unix == 0 {
		return ""
	}
	return time.Unix(unix, 0).Format(time.RFC3339)
}

// hasCredentialOverride reports whether auth status should inspect the same
// effective credentials as ordinary commands instead of every saved instance.
func hasCredentialOverride(cmd *cobra.Command) bool {
	return flagChanged(cmd, "base-url") || flagChanged(cmd, "token") || flagChanged(cmd, "server") ||
		os.Getenv("YOUTRACK_BASE_URL") != "" || os.Getenv("YOUTRACK_TOKEN") != ""
}

func printAuthStatus(cmd *cobra.Command, s *AuthStatus) {
	w := cmd.OutOrStdout()
	switch {
	case s.Status == "authenticated":
		_, _ = fmt.Fprintf(w, "%s Authenticated\n", color.New(color.FgGreen).Sprint("✓"))
		_, _ = fmt.Fprintf(w, "  URL:   %s\n", s.URL)
		_, _ = fmt.Fprintf(w, "  User:  %s (%s)\n", output.Sanitize(s.FullName), output.Sanitize(s.Login))
		_, _ = fmt.Fprintf(w, "  Auth:  %s\n", describeAuth(s.AuthMethod, s.Source, s.Alias, s.ExpiresAt, s.HasRefreshToken))
	case strings.HasPrefix(s.Status, "invalid token"):
		_, _ = fmt.Fprintf(w, "%s Token invalid\n", color.New(color.FgRed).Sprint("✗"))
		_, _ = fmt.Fprintf(w, "  URL:   %s\n", s.URL)
		if s.AuthMethod != "" {
			_, _ = fmt.Fprintf(w, "  Auth:  %s\n", describeAuth(s.AuthMethod, s.Source, s.Alias, s.ExpiresAt, s.HasRefreshToken))
		}
		_, _ = fmt.Fprintf(w, "  Error: %s\n", output.Sanitize(strings.TrimPrefix(s.Status, "invalid token: ")))
	default:
		_, _ = fmt.Fprintf(w, "%s Not configured\n", color.New(color.FgYellow).Sprint("!"))
		if detail := strings.TrimPrefix(s.Status, "not configured: "); strings.Contains(detail, "OAuth session") {
			_, _ = fmt.Fprintf(w, "  Error: %s\n", output.Sanitize(detail))
		}
		_, _ = fmt.Fprintf(w, "  Run:   youtrack auth get-token\n")
	}
}

// describeAuth renders the auth method line for humans, e.g. "OAuth session (profile work), expires 2026-09-29 15:04, refresh token stored".
func describeAuth(method, source, alias, expiresAt string, hasRefresh bool) string {
	var b strings.Builder
	if method == "oauth" {
		b.WriteString("OAuth session")
	} else {
		b.WriteString("permanent token")
	}
	switch source {
	case "flag":
		b.WriteString(" (--token flag)")
	case "env":
		b.WriteString(" (YOUTRACK_TOKEN)")
	case "profile":
		if alias != "" {
			b.WriteString(" (profile " + output.Sanitize(alias) + ")")
		}
	}
	if t, err := time.Parse(time.RFC3339, expiresAt); err == nil {
		b.WriteString(", expires " + t.Format("2006-01-02 15:04"))
	}
	if method == "oauth" {
		if hasRefresh {
			b.WriteString(", refresh token stored (auto-renews)")
		} else {
			b.WriteString(", no refresh token")
		}
	}
	return b.String()
}

// runMultiInstanceStatus checks auth for every configured instance and prints results.
func runMultiInstanceStatus(cmd *cobra.Command, cfg *config.Config) error {
	aliases := cfg.Aliases()
	statuses := make([]InstanceStatus, 0, len(aliases))
	for _, alias := range aliases {
		if err := cmd.Context().Err(); err != nil {
			return err
		}
		inst := cfg.Instances[alias]
		s := InstanceStatus{
			Alias:  alias,
			URL:    inst.URL,
			Active: alias == cfg.CurrentInstance,
		}
		if inst.AuthMethod == config.AuthMethodOAuth && inst.Token == "" {
			s.Status = fmt.Sprintf("invalid token: OAuth session expired or was revoked — run: youtrack auth login --session %s", alias)
			statuses = append(statuses, s)
			continue
		}
		client, err := internal.NewSavedInstanceClient(cmd, alias, &inst)
		var user *api.User
		if err == nil {
			user, err = client.GetCurrentUser()
		}
		if err != nil {
			if contextErr := cmd.Context().Err(); contextErr != nil {
				return contextErr
			}
			s.Status = fmt.Sprintf("invalid token: %s", err.Error())
		} else {
			s.Status = "authenticated"
			s.Login = user.Login
			s.FullName = user.FullName
		}
		statuses = append(statuses, s)
	}
	// Re-read so expiries reflect any refresh performed by the checks above.
	if fresh, err := config.Load(); err == nil {
		cfg = fresh
	}
	for i := range statuses {
		inst := cfg.Instances[statuses[i].Alias]
		statuses[i].AuthMethod = authMethodOf(inst.AuthMethod)
		statuses[i].ExpiresAt = formatExpiry(inst.TokenExpiresAt)
		statuses[i].HasRefreshToken = inst.RefreshToken != ""
	}
	if !jsonOutput(cmd) {
		printMultiInstanceStatus(cmd, statuses)
		return nil
	}
	return output.Print(cmd, statuses)
}

func printMultiInstanceStatus(cmd *cobra.Command, statuses []InstanceStatus) {
	w := cmd.OutOrStdout()
	for i := range statuses {
		s := &statuses[i]
		activeMarker := ""
		if s.Active {
			activeMarker = " " + color.New(color.FgGreen).Sprint("[active]")
		}
		switch s.Status {
		case "authenticated":
			_, _ = fmt.Fprintf(w, "  %s %-20s %s  %s (%s)%s\n",
				color.New(color.FgGreen).Sprint("✓"),
				s.Alias, s.URL, output.Sanitize(s.FullName), output.Sanitize(s.Login), activeMarker)
			if s.AuthMethod == "oauth" {
				_, _ = fmt.Fprintf(w, "      %s\n", describeAuth(s.AuthMethod, "", "", s.ExpiresAt, s.HasRefreshToken))
			}
		default:
			_, _ = fmt.Fprintf(w, "  %s %-20s %s  %s%s\n",
				color.New(color.FgRed).Sprint("✗"),
				s.Alias, s.URL, output.Sanitize(s.Status), activeMarker)
		}
	}
}
