package internal

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal/config"
)

// GetClient builds an API client using a four-tier resolution chain:
//  1. --base-url / --token flags (explicit per-command override, always wins)
//  2. --server flag (selects a named profile from config by alias)
//  3. YOUTRACK_BASE_URL / YOUTRACK_TOKEN env vars (explicit, process-scoped)
//  4. Active instance from ~/.youtrack-cli-config.json (convenience fallback)
func GetClient(cmd *cobra.Command) (*api.Client, error) {
	flags := cmd.Root().PersistentFlags()

	baseURL, _ := flags.GetString("base-url")
	token, _ := flags.GetString("token")
	verbose, _ := flags.GetBool("verbose")
	serverAlias, _ := flags.GetString("server")
	fields, _ := flags.GetString("fields")

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

	c := api.NewClient(baseURL, token)
	c.Verbose = verbose
	c.Fields = expandFields(fields)
	return c, nil
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
	parts := splitFieldsInternal(fields)
	for i, p := range parts {
		// Only expand bare names (no sub-field spec already present).
		if expanded, ok := defaults[p]; ok {
			parts[i] = expanded
		}
	}
	return strings.Join(parts, ",")
}

// splitFieldsInternal is a copy of the output package's splitFields kept here
// to avoid a cross-package import cycle. Both must stay in sync.
func splitFieldsInternal(s string) []string {
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
