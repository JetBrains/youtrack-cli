package output

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
)

// Print writes data to stdout using the format specified by the global --format flag.
func Print(cmd *cobra.Command, data any) error {
	format, _ := cmd.Root().PersistentFlags().GetString("format")
	if format == "table" {
		return PrintTable(data)
	}
	return printJSON(data)
}

func printJSON(data any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}
