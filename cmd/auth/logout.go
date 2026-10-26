package auth

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/config"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

// LogoutResult is the JSON output of the auth logout command.
type LogoutResult struct {
	URL    string `json:"url"`
	Status string `json:"status"`
}

func newLogoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove saved credentials for an instance",
		Long: `Removes an instance from the config file (~/.config/youtrack/config.json) and
its tokens from the OS keyring.

For an OAuth session (created by "auth login --session"), logout removes the tokens
locally only: the refresh token stays valid at Hub until it expires. To revoke
it, open your profile in Hub (Authentication / Account Security).

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
			// Remove under the config lock on a fresh load, so a concurrent refresh cannot resurrect the session
			// and the session reported below is the latest one (a refresh may have changed it since the prompt).
			var purgeErr error
			err = func() error {
				unlock, err := config.Lock(cmd.Context())
				if err != nil {
					return err
				}
				defer unlock()
				fresh, err := config.Load()
				if err != nil {
					return err
				}
				cur, ok := fresh.Instances[targetAlias]
				if !ok {
					return fmt.Errorf("instance %q was removed by another youtrack process", targetAlias)
				}
				inst = cur
				if err := fresh.RemoveInstance(targetAlias); err != nil {
					return err
				}
				if fresh.CurrentInstance == targetAlias {
					fresh.CurrentInstance = ""
				}
				if err := fresh.Save(); err != nil {
					return err
				}
				// Purged only after the save succeeds, so a failed save keeps the profile usable, and before the lock is
				// released, so a concurrent login that recreates targetAlias keeps its new credentials.
				purgeErr = config.PurgeKeyringToken(targetAlias)
				return nil
			}()
			if err != nil {
				return err
			}
			if purgeErr != nil {
				output.PrintWarn(cmd, purgeErr.Error())
			}
			warnRefreshTokenStaysValid(cmd, targetAlias, &inst)
			res := LogoutResult{URL: inst.URL, Status: "logged out"}
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

// warnRefreshTokenStaysValid explains that logout of an OAuth session only removes the tokens locally: Hub offers the CLI no way to revoke them.
func warnRefreshTokenStaysValid(cmd *cobra.Command, alias string, inst *config.Instance) {
	if inst.AuthMethod == config.AuthMethodOAuth && inst.RefreshToken != "" {
		output.PrintWarn(cmd, fmt.Sprintf("the refresh token for %q was removed locally but stays valid at Hub until it expires; to revoke it, open your profile in Hub (Authentication / Account Security)", alias))
	}
}
