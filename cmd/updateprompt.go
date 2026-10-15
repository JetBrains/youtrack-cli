package cmd

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/config"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

// maybePromptForUpdate offers the newer release cached by the last background update
// check before the command runs, as Codex CLI does. Only a person at a terminal sees
// it (see internal.UpdatePromptEnabled), and only for install methods that
// "youtrack update" can upgrade. A failed or skipped update never stops the command.
func maybePromptForUpdate(cmd *cobra.Command) {
	switch cmd.Name() {
	case "update", "help", "completion", cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd:
		return
	}
	if cmd.HasParent() && cmd.Parent().Name() == "completion" {
		return
	}
	if quiet, _ := cmd.Root().PersistentFlags().GetBool("quiet"); quiet || output.Format(cmd) != "table" || !internal.UpdatePromptEnabled() {
		return
	}
	cfg, err := config.Load()
	if err != nil || !internal.PendingUpdate(internal.Version, cfg.LatestVersion, cfg.DismissedVersion) {
		return
	}
	latest := cfg.LatestVersion
	method := internal.DetectInstallMethodContext(cmd.Context())
	if _, _, ok := method.AutoUpgradeCommand(); !ok {
		return
	}

	errOut := cmd.ErrOrStderr()
	faint := color.New(color.Faint).SprintFunc()
	_, _ = fmt.Fprintf(errOut, "%s %s %s %s\n", color.New(color.Bold).Sprint("Update available!"),
		faint(internal.FormatVersion(internal.Version)), faint("→"), color.GreenString(internal.FormatVersion(latest)))
	_, _ = fmt.Fprintf(errOut, "%s https://github.com/JetBrains/youtrack-cli/releases/tag/v%s\n\n", faint("Release notes:"), latest)
	_, _ = fmt.Fprintf(errOut, "  1. Update now (runs `%s`)\n  2. Skip\n  3. Skip until next version\n\nChoose [1-3, default 2]: ", method.UpgradeCommand())
	answer, err := internal.ReadLineContext(cmd.Context(), bufio.NewReader(cmd.InOrStdin()))
	if err != nil {
		_, _ = fmt.Fprintln(errOut)
		return
	}
	switch strings.TrimSpace(answer) {
	case "1":
		// The upgrade writes to stderr so stdout stays clean for the command's own output.
		if _, err := internal.RunAutoUpgradeContext(cmd.Context(), method, latest, errOut, errOut); err != nil {
			output.PrintWarn(cmd, fmt.Sprintf("update failed: %v; run: %s", err, method.UpgradeCommand()))
			return
		}
		_, _ = fmt.Fprintf(errOut, "%s Updated to %s. This command runs on the current version.\n\n",
			color.GreenString("✓"), internal.FormatVersion(latest))
	case "3":
		internal.DismissUpdate(latest)
		_, _ = fmt.Fprintln(errOut)
	default:
		_, _ = fmt.Fprintln(errOut)
	}
}
