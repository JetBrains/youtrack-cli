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
	return &cobra.Command{
		Use:   "update",
		Short: "Check for a newer YouTrack CLI release and upgrade",
		Long: `Checks GitHub for the latest YouTrack CLI release. When a newer release
exists, detects the install method (Homebrew, Scoop, Chocolatey, WinGet, npm,
go install, or the install script) and upgrades through that same channel.
Install methods with no safe single-package upgrade path (apt, rpm, AUR) or an
undetected install fall back to printing manual instructions, plus a
gh-based alternative when the gh CLI is installed.

With --json or --plain, or when no newer release exists, only the status is
reported — no upgrade is attempted.`,
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
			canAutoUpgrade := humanFormat && term.IsTerminal(int(os.Stdout.Fd()))
			if !status.UpdateAvailable || !canAutoUpgrade {
				return output.Print(cmd, status)
			}
			return runUpgrade(cmd, status)
		},
	}
}

// runUpgrade attempts an automated upgrade through the detected install method,
// falling back to manual instructions when that isn't possible or fails.
func runUpgrade(cmd *cobra.Command, status *internal.UpdateStatus) error {
	out := cmd.OutOrStdout()
	errOut := cmd.ErrOrStderr()
	method := internal.DetectInstallMethodContext(cmd.Context())

	_, _ = fmt.Fprintf(out, "YouTrack CLI %s %s %s\n",
		color.New(color.Faint).Sprint(internal.FormatVersion(status.CurrentVersion)),
		color.New(color.Faint).Sprint("→"),
		color.New(color.FgGreen).Sprint(internal.FormatVersion(status.LatestVersion)))

	attempted, err := internal.RunAutoUpgradeContext(cmd.Context(), method, status.LatestVersion, out, errOut)
	if !attempted {
		printUpgradeFallback(out, method)
		return nil
	}
	if err != nil {
		output.PrintWarn(cmd, "auto-upgrade failed: "+err.Error())
		printUpgradeFallback(out, method)
		return nil //nolint:nilerr // deliberate: fall back to manual instructions instead of failing the command
	}
	output.PrintSuccess(cmd, "YouTrack CLI upgraded to "+internal.FormatVersion(status.LatestVersion))
	return nil
}

func printUpgradeFallback(out io.Writer, method internal.InstallMethod) {
	_, _ = fmt.Fprintf(out, "  %s  %s\n",
		color.New(color.Bold).Sprint("Upgrade:"), method.UpgradeCommand())
	if method != internal.InstallScript && internal.HasGH() {
		_, _ = fmt.Fprintf(out, "  %s  %s\n",
			color.New(color.Bold).Sprint("Alternative (via gh):"), internal.GHInstallScriptCommand)
	}
}
