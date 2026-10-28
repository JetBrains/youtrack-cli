package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

func newUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Check for a newer YouTrack CLI release and upgrade",
		Long: `Checks GitHub for the latest YouTrack CLI release. When a newer release
exists, detects the install method (Homebrew, Scoop, Chocolatey, WinGet, npm,
go install, or the install script) and upgrades through that same channel. An
install-script install re-runs the official installer, which downloads the new
release, verifies it, and replaces the binary in place. Methods with no safe
single-package upgrade path (apt, rpm, AUR) and an undetected install fall back
to printing manual instructions.

In a terminal the upgrade runs right away. With --json or --plain, when stdout
is not a terminal, or when no newer release exists, only the status is
reported — no upgrade is attempted, unless you pass --yes.

With --yes the upgrade runs without a terminal too, as when an app or script
runs the CLI, and a failed or unavailable upgrade exits with an error instead
of printing manual instructions. With JSON output the upgrade's own output goes
to stderr and stdout gets the status with "upgraded": true.`,
		Example: `  youtrack update
  youtrack update --json
  youtrack update --yes --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			humanFormat := output.Format(cmd) == "table"
			if humanFormat {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Checking for updates...")
			}
			status, err := internal.CheckNowContext(cmd.Context(), internal.Version)
			if err != nil {
				return fmt.Errorf("checking for updates: %w", err)
			}
			// Auto-upgrade must never fire on a redirected/logged invocation, even if
			// --format table was explicitly passed (which bypasses Format's own TTY
			// auto-detection) — require a real terminal on top of the resolved format.
			yes, _ := cmd.Flags().GetBool("yes")
			canAutoUpgrade := yes || humanFormat && term.IsTerminal(int(os.Stdout.Fd()))
			if !status.UpdateAvailable || !canAutoUpgrade {
				return output.Print(cmd, status)
			}
			return runUpgrade(cmd, status, yes)
		},
	}
	cmd.Flags().BoolP("yes", "y", false, "Upgrade even when not run from a terminal, and fail if the upgrade does not succeed")
	return cmd
}

// runUpgrade attempts an automated upgrade through the detected install method,
// falling back to manual instructions when that isn't possible or fails; with
// strict (--yes) those cases are errors instead.
func runUpgrade(cmd *cobra.Command, status *internal.UpdateStatus, strict bool) error {
	out := cmd.OutOrStdout()
	errOut := cmd.ErrOrStderr()
	human := output.Format(cmd) == "table"
	method := internal.DetectInstallMethodContext(cmd.Context())

	if human {
		_, _ = fmt.Fprintf(out, "YouTrack CLI %s %s %s\n",
			color.New(color.Faint).Sprint(internal.FormatVersion(status.CurrentVersion)),
			color.New(color.Faint).Sprint("→"),
			color.New(color.FgGreen).Sprint(internal.FormatVersion(status.LatestVersion)))
	}

	upgradeOut := out
	if !human {
		upgradeOut = errOut // stdout carries only the status document
	}
	attempted, err := internal.RunAutoUpgradeContext(cmd.Context(), method, status.LatestVersion, upgradeOut, errOut)
	switch {
	case strict && !attempted:
		return fmt.Errorf("automatic upgrade is not available for this install (manual upgrade: %s)", method.UpgradeCommand())
	case strict && err != nil:
		return fmt.Errorf("upgrade failed: %w (manual upgrade: %s)", err, method.UpgradeCommand())
	case !attempted:
		printUpgradeFallback(out, method)
		return nil
	case err != nil:
		output.PrintWarn(cmd, "auto-upgrade failed: "+err.Error())
		printUpgradeFallback(out, method)
		return nil //nolint:nilerr // deliberate: fall back to manual instructions instead of failing the command
	case human:
		output.PrintSuccess(cmd, "YouTrack CLI upgraded to "+internal.FormatVersion(status.LatestVersion))
		return nil
	default:
		status.Upgraded = true
		return output.Print(cmd, status)
	}
}

func printUpgradeFallback(out io.Writer, method internal.InstallMethod) {
	_, _ = fmt.Fprintf(out, "  %s  %s\n",
		color.New(color.Bold).Sprint("Upgrade:"), method.UpgradeCommand())
}
