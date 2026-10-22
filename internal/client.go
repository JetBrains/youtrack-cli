package internal

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal/config"
)

// GetClient builds an API client using a four-tier resolution chain:
//  1. --base-url / --token flags (explicit per-command override, always wins)
//  2. --server flag (selects a named profile from config by alias)
//  3. YOUTRACK_BASE_URL / YOUTRACK_TOKEN env vars (shell/CI, process-scoped)
//  4. Active instance from the config file (set via `instance use`)
//
// A token taken from a saved profile (tiers 2 and 4) is only used when the resolved URL has the same origin as that profile.
func GetClient(cmd *cobra.Command) (*api.Client, error) {
	flags := cmd.Root().PersistentFlags()

	baseURL, _ := flags.GetString("base-url")
	token, _ := flags.GetString("token")
	verbose, _ := flags.GetBool("verbose")
	serverAlias, _ := flags.GetString("server")
	fields, _ := flags.GetString("fields")

	// savedTokenAlias/savedTokenURL record which saved profile supplied the token, so it is only ever sent to that profile's origin.
	var savedTokenAlias, savedTokenURL string

	// Tier 2: --server flag selects a named profile (per-invocation, does not persist).
	if (baseURL == "" || token == "") && serverAlias != "" {
		cfg, err := config.Load()
		if err != nil {
			return nil, fmt.Errorf("loading config: %w", err)
		}
		inst, ok := cfg.Instances[serverAlias]
		if !ok {
			return nil, fmt.Errorf(
				"unknown server profile %q — run: youtrack instance list\nKnown: %s",
				serverAlias, strings.Join(cfg.Aliases(), ", "),
			)
		}
		if baseURL == "" {
			baseURL = inst.URL
		}
		if token == "" {
			token = inst.Token
			savedTokenAlias, savedTokenURL = serverAlias, inst.TokenURL()
			warnIfTokenExpired(cmd, serverAlias, &inst)
		}
	}

	// Tier 3: env vars win over stored config — explicit, process-scoped, CI-safe.
	// Independent guards allow partial fills: one credential from env, the other from config.
	if baseURL == "" {
		baseURL = os.Getenv("YOUTRACK_BASE_URL")
	}
	if token == "" {
		token = os.Getenv("YOUTRACK_TOKEN")
	}

	// Tier 4: active config instance as convenience fallback.
	if baseURL == "" || token == "" {
		cfg, err := config.Load()
		if err == nil && len(cfg.Instances) > 0 {
			inst, alias := cfg.ActiveInstance()
			switch {
			case inst != nil: // Active instance found; fill any still-empty slot.
				if baseURL == "" {
					baseURL = inst.URL
				}
				if token == "" {
					token = inst.Token
					savedTokenAlias, savedTokenURL = alias, inst.TokenURL()
					warnIfTokenExpired(cmd, alias, inst)
				}
			case alias != "": // currentInstance is set but the alias is missing from the map (stale pointer).
				return nil, fmt.Errorf(
					"active instance %q not found in config — run: youtrack instance use <alias>\nKnown: %s",
					alias, strings.Join(cfg.Aliases(), ", "),
				)
			default: // Config has instances but none is selected as active.
				return nil, fmt.Errorf(
					"no active instance — run: youtrack instance use <alias>\nKnown: %s",
					strings.Join(cfg.Aliases(), ", "),
				)
			}
		}
	}

	if baseURL == "" {
		return nil, fmt.Errorf(
			"no instance configured — run: youtrack auth get-token\n" +
				"(or set YOUTRACK_BASE_URL + YOUTRACK_TOKEN, or use --base-url + --token flags)",
		)
	}
	if token == "" {
		return nil, fmt.Errorf(
			"YOUTRACK_TOKEN is not set — run: youtrack auth get-token\n" +
				"(or set YOUTRACK_TOKEN, or use --token flag)",
		)
	}

	// A saved token is bound to its instance: never forward it to a URL overridden via --base-url or YOUTRACK_BASE_URL.
	if savedTokenURL != "" && !sameOrigin(baseURL, savedTokenURL) {
		return nil, fmt.Errorf(
			"refusing to send the saved token for %q (%s) to %s — pass --token or set YOUTRACK_TOKEN for that instance, or select its profile with --server <alias>",
			savedTokenAlias, savedTokenURL, baseURL,
		)
	}

	if api.IsInsecureRemoteHTTP(baseURL) {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
			"warning: %s uses plain http — the token is sent unencrypted; switch the instance URL to https\n", baseURL)
	}

	api.UserAgent = "youtrack-cli/" + Version

	c := api.NewClient(baseURL, token).WithContext(cmd.Context())
	c.Verbose = verbose
	c.Fields = expandFields(fields)
	return c, nil
}

// RequireTokenURL refuses to sign in again for a saved profile whose config-file URL has another origin than the URL its keyring token is bound to.
// Renewing would send the password or a browser sign-in to a host set by whoever edited the config file, and bind the new token to it.
func RequireTokenURL(alias string, inst *config.Instance) error {
	tokenURL := inst.TokenURL()
	if tokenURL == inst.URL || sameOrigin(inst.URL, tokenURL) {
		return nil
	}
	path, _ := config.ConfigPath()
	return fmt.Errorf(
		"refusing to sign in for %q: the config file URL %s differs from %s, the URL its saved token was issued for — check %s; to sign in to %s on purpose, run: youtrack instance delete %s, then youtrack instance add",
		alias, inst.URL, tokenURL, path, inst.URL, alias,
	)
}

// sameOrigin reports whether two instance URLs share scheme, host and port (default ports normalized); unparsable or host-less URLs never match.
func sameOrigin(a, b string) bool {
	ua, errA := url.Parse(strings.TrimSpace(a))
	ub, errB := url.Parse(strings.TrimSpace(b))
	if errA != nil || errB != nil || ua.Hostname() == "" || ub.Hostname() == "" {
		return false
	}
	return strings.EqualFold(ua.Scheme, ub.Scheme) &&
		strings.EqualFold(ua.Hostname(), ub.Hostname()) &&
		originPort(ua) == originPort(ub)
}

// originPort returns the explicit port, or the scheme's default so "https://x" and "https://x:443" compare equal.
func originPort(u *url.URL) string {
	if p := u.Port(); p != "" {
		return p
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return "443"
	case "http":
		return "80"
	}
	return ""
}

// warnIfTokenExpired prints a stderr renewal hint when a stored token is past its expiry, before the follow-up request 401s.
func warnIfTokenExpired(cmd *cobra.Command, alias string, inst *config.Instance) {
	if inst.TokenExpiresAt == 0 || time.Now().Unix() < inst.TokenExpiresAt {
		return
	}
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
		"warning: token for %q expired at %s — run: youtrack instance login %s\n",
		alias, time.Unix(inst.TokenExpiresAt, 0).Format("2006-01-02 15:04"), alias)
}

// expandFields auto-expands bare nested entity names to include useful sub-fields.
// This ensures e.g. "--fields reporter" fetches reporter(id,login,fullName) from the API
// so the display layer has real data to work with, not just the $type discriminator.
// Splitting is paren-aware so "reporter(login,fullName)" is not mis-split on the inner comma.
func expandFields(fields string) string {
	if fields == "" {
		return ""
	}
	defaults := map[string]string{
		"reporter":   "reporter(id,login,fullName)",
		"updater":    "updater(id,login,fullName)",
		"draftOwner": "draftOwner(id,login,fullName)",
		"project":    "project(id,shortName,name)",
	}
	parts := SplitFields(fields)
	for i, p := range parts {
		// Only expand bare names (no sub-field spec already present).
		if expanded, ok := defaults[p]; ok {
			parts[i] = expanded
		}
	}
	return commentProjectionFields(strings.Join(parts, ","))
}

// commentProjectionFields retains deletion authority for narrow comment
// projections, including comments nested inside other projected entities.
func commentProjectionFields(fields string) string {
	parts := SplitFields(fields)
	for i, spec := range parts {
		open := strings.IndexByte(spec, '(')
		if open < 0 || !strings.HasSuffix(spec, ")") {
			continue
		}
		name := spec[:open]
		inner := commentProjectionFields(spec[open+1 : len(spec)-1])
		if name == "comments" || name == "pinnedComments" {
			hasDeleted := false
			for _, field := range SplitFields(inner) {
				if field == "deleted" {
					hasDeleted = true
					break
				}
			}
			if !hasDeleted {
				if inner != "" {
					inner += ","
				}
				inner += "deleted"
			}
		}
		parts[i] = name + "(" + inner + ")"
	}
	return strings.Join(parts, ",")
}

// SplitFields splits a comma-separated field spec while respecting nested
// parentheses, so "id,comments(id,text),summary" yields three entries rather than four.
// It serves both the request side (expandFields) and the output projection.
func SplitFields(s string) []string {
	var parts []string
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	return append(parts, strings.TrimSpace(s[start:]))
}
