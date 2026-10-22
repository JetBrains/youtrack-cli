package internal

import (
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// OutputFormat resolves --json, --plain, explicit --format, and finally stdout's
// terminal status, in that order. Keeping it here lets confirmations and entity
// output share the same rules without an import cycle.
func OutputFormat(cmd *cobra.Command) string {
	flags := cmd.Root().PersistentFlags()
	if jsonFlag, _ := flags.GetBool("json"); jsonFlag {
		return "json"
	}
	if plainFlag, _ := flags.GetBool("plain"); plainFlag {
		return "plain"
	}
	if flags.Changed("format") {
		format, _ := flags.GetString("format")
		return format
	}
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return "json"
	}
	return "table"
}
