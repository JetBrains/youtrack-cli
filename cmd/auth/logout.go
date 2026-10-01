package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/config"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

// LogoutResult is the JSON output of the auth logout command.
type LogoutResult struct {
	URL    string `json:"url"`
	Status string `json:"status"`
	// Revoked reports whether an OAuth session was revoked at Hub (best-effort; false for permanent tokens).
	Revoked bool `json:"revoked,omitempty"`
}

// revokeTimeout bounds best-effort revocation so an unreachable Hub never blocks logout.
const revokeTimeout = 10 * time.Second

func newLogoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove saved credentials for an instance",
		Long: `Removes an instance from ~/.youtrack-cli-config.json and its tokens from
the OS keyring.

For an OAuth session (created by "instance login"), the refresh and access
tokens are also revoked at Hub when it advertises a revocation endpoint. This is
best-effort: logout succeeds even when revocation fails. You can always revoke
access yourself in Hub (your profile → Authentication / Account Security).

A permanent token remains valid on the YouTrack server until deleted manually
from your profile (Users → your account → Authentication tab).

Use --json for machine-readable output.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			targetAlias, _ := cmd.Flags().GetString("instance")

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if len(cfg.Instances) == 0 {
				res := LogoutResult{Status: "not configured"}
				if !jsonOutput(cmd) {
					output.PrintSuccess(cmd, "not logged in — nothing to remove")
					return nil
				}
				return output.Print(cmd, res)
			}

			if targetAlias == "" {
				targetAlias = cfg.CurrentInstance
			}
			if targetAlias == "" {
				return fmt.Errorf("no active instance — specify one with --instance <alias>")
			}
			inst, ok := cfg.Instances[targetAlias]
			if !ok {
				return fmt.Errorf("instance %q not found — run: youtrack instance list", targetAlias)
			}
			confirmed, err := internal.Confirm(cmd, fmt.Sprintf("Remove credentials for %s (%s)?", targetAlias, inst.URL))
			if err != nil {
				return err
			}
			if !confirmed {
				res := LogoutResult{URL: inst.URL, Status: "canceled"}
				if !jsonOutput(cmd) {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Canceled.")
					return nil
				}
				return output.Print(cmd, res)
			}
			revoked := revokeOAuthSession(cmd, targetAlias, &inst)
			wasActive := cfg.CurrentInstance == targetAlias
			if err := cfg.RemoveInstance(targetAlias); err != nil {
				return err
			}
			if wasActive {
				cfg.CurrentInstance = ""
			}
			if err := cfg.Save(); err != nil {
				return err
			}
			if err := config.PurgeKeyringToken(targetAlias); err != nil {
				output.PrintWarn(cmd, err.Error())
			}
			res := LogoutResult{URL: inst.URL, Status: "logged out", Revoked: revoked}
			if !jsonOutput(cmd) {
				output.PrintSuccess(cmd, "logged out from "+inst.URL)
				return nil
			}
			return output.Print(cmd, res)
		},
	}
	cmd.Flags().Bool("force", false, "Skip the confirmation prompt")
	cmd.Flags().String("instance", "", "Alias of the instance to log out from (defaults to active instance)")
	return cmd
}

// revokeOAuthSession revokes a saved OAuth session's tokens at Hub (RFC 7009), best-effort: failures only warn, never block logout.
func revokeOAuthSession(cmd *cobra.Command, alias string, inst *config.Instance) bool {
	if inst.AuthMethod != config.AuthMethodOAuth || (inst.RefreshToken == "" && inst.Token == "") {
		return false
	}
	if inst.RevocationEndpoint == "" || inst.ClientID == "" {
		output.PrintWarn(cmd, fmt.Sprintf("Hub advertises no token revocation endpoint — the refresh token for %q stays valid at Hub; to revoke it, remove the application's access in your Hub profile", alias))
		return false
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), revokeTimeout)
	defer cancel()
	revoked := false
	// Revoking the refresh token ends the whole grant; the access token is revoked too in case Hub treats them separately.
	for _, t := range []struct{ token, hint string }{{inst.RefreshToken, "refresh_token"}, {inst.Token, "access_token"}} {
		if t.token == "" {
			continue
		}
		if err := api.RevokeOAuthTokenContext(ctx, inst.RevocationEndpoint, inst.ClientID, t.token, t.hint); err != nil {
			output.PrintWarn(cmd, fmt.Sprintf("could not revoke the %s for %q at Hub (%v) — removing it locally anyway", strings.ReplaceAll(t.hint, "_", " "), alias, err))
			continue
		}
		if t.hint == "refresh_token" || inst.RefreshToken == "" {
			revoked = true
		}
	}
	return revoked
}
