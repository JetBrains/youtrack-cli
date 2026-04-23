package output

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	successIcon = color.New(color.FgGreen).Sprint("✓")
	warnIcon    = color.New(color.FgYellow).Sprint("!")
)

// PrintSuccess writes "✓ msg\n" to cmd's stdout.
// Not suppressed by --quiet — success confirmations are always shown.
func PrintSuccess(cmd *cobra.Command, msg string) {
	fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", successIcon, msg)
}

// PrintWarn writes "! msg\n" to cmd's stderr.
// Suppressed when --quiet is set.
func PrintWarn(cmd *cobra.Command, msg string) {
	if isQuiet(cmd) {
		return
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "%s %s\n", warnIcon, msg)
}

// PrintTip writes "  Tip: msg\n" to cmd's stderr.
// Suppressed when --quiet is set.
func PrintTip(cmd *cobra.Command, msg string) {
	if isQuiet(cmd) {
		return
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "  Tip: %s\n", msg)
}

// isQuiet reports whether --quiet is set on the root command.
func isQuiet(cmd *cobra.Command) bool {
	q, _ := cmd.Root().PersistentFlags().GetBool("quiet")
	return q
}
