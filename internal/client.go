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
//  3. Active instance from ~/.youtrack-cli-config.json
//  4. YOUTRACK_BASE_URL / YOUTRACK_TOKEN env vars (legacy single-instance fallback)
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

	// Tier 3: active config instance fills any slot not already set by flags.
	if baseURL == "" || token == "" {
		cfg, err := config.Load()
		if err == nil && len(cfg.Instances) > 0 {
			inst, alias := cfg.ActiveInstance()
			switch {
			case inst != nil:
				if baseURL == "" {
					baseURL = inst.URL
				}
				if token == "" {
					token = inst.Token
				}
			case alias != "":
				// currentInstance is set but the alias is missing from the map (stale pointer).
				return nil, fmt.Errorf(
					"active instance %q not found in config — run: youtrack instance use <alias>\nKnown: %s",
					alias, strings.Join(cfg.Aliases(), ", "),
				)
			default:
				// Config has instances but none is selected as active.
				return nil, fmt.Errorf(
					"no active instance — run: youtrack instance use <alias>\nKnown: %s",
					strings.Join(cfg.Aliases(), ", "),
				)
			}
		}
	}

	// Tier 4: env vars (legacy single-instance fallback).
	if baseURL == "" {
		baseURL = os.Getenv("YOUTRACK_BASE_URL")
	}
	if token == "" {
		token = os.Getenv("YOUTRACK_TOKEN")
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
	c.Fields = fields
	return c, nil
}
