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

// successResult is the JSON success object emitted in JSON mode so stdout stays valid JSON.
type successResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// PrintSuccess confirms a write on cmd's stdout — {"status":"ok","message":"..."} in JSON mode, "✓ msg" otherwise; never suppressed by --quiet.
func PrintSuccess(cmd *cobra.Command, msg string) {
	if Format(cmd) == "json" {
		_ = printJSON(cmd.OutOrStdout(), successResult{Status: "ok", Message: msg})
		return
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", successIcon, msg)
}

// PrintWarn writes "! msg\n" to cmd's stderr.
// Suppressed when --quiet is set.
func PrintWarn(cmd *cobra.Command, msg string) {
	if isQuiet(cmd) {
		return
	}
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "%s %s\n", warnIcon, msg)
}

// PrintTip writes "  Tip: msg\n" to cmd's stderr.
// Suppressed when --quiet is set.
func PrintTip(cmd *cobra.Command, msg string) {
	if isQuiet(cmd) {
		return
	}
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  Tip: %s\n", msg)
}

// isQuiet reports whether --quiet is set on the root command.
func isQuiet(cmd *cobra.Command) bool {
	q, _ := cmd.Root().PersistentFlags().GetBool("quiet")
	return q
}
