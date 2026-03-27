package output

import (
	"encoding/json"
	"io"

	"github.com/spf13/cobra"
)

// Print writes data using the format specified by the global --format flag.
func Print(cmd *cobra.Command, data any) error {
	w := cmd.OutOrStdout()
	format, _ := cmd.Root().PersistentFlags().GetString("format")
	if format == "table" {
		return PrintTable(w, data)
	}
	return printJSON(w, data)
}

func printJSON(w io.Writer, data any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}
