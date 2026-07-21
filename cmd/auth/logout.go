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
		Long: `Removes an instance from ~/.youtrack-cli-config.json.

The token remains valid on the YouTrack server until deleted manually
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
			config.PurgeKeyringToken(targetAlias)
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
