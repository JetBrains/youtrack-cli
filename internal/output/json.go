package output

import (
	"encoding/json"
	"io"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// Format resolves the effective output format for the command using a priority chain:
//  1. --json flag → "json"
//  2. --plain flag → "plain"
//  3. --format explicitly set by user → that value
//  4. stdout is not a TTY → "json" (auto-detect pipes/redirects)
//  5. default → "table"
func Format(cmd *cobra.Command) string {
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

// Print writes data using the resolved output format.
func Print(cmd *cobra.Command, data any) error {
	w := cmd.OutOrStdout()
	switch Format(cmd) {
	case "plain":
		return PrintPlain(w, data)
	case "json":
		return printJSON(w, data)
	default:
		return PrintTable(w, data)
	}
}

func printJSON(w io.Writer, data any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}
