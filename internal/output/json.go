package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

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
	fields, _ := cmd.Root().PersistentFlags().GetString("fields")
	switch Format(cmd) {
	case "plain":
		if fields != "" {
			return printProjected(w, data, fields, false)
		}
		return PrintPlain(w, data)
	case "json":
		return printJSON(w, data)
	default:
		if fields != "" {
			return printProjected(w, data, fields, true)
		}
		return PrintTable(w, data)
	}
}

func printJSON(w io.Writer, data any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// printProjected renders only the fields named in the comma-separated `fields`
// string. It works on any type by doing a JSON round-trip to a map, then
// resolving dot-separated paths (e.g. "project.shortName").
func printProjected(w io.Writer, data any, fields string, withHeader bool) error {
	names := strings.Split(fields, ",")
	for i, n := range names {
		names[i] = strings.TrimSpace(n)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return err
	}

	writeRow := func(obj map[string]any) {
		vals := make([]string, len(names))
		for i, name := range names {
			vals[i] = resolveField(obj, name)
		}
		_, _ = fmt.Fprintln(w, strings.Join(vals, "\t"))
	}

	var arr []map[string]any
	if json.Unmarshal(b, &arr) == nil {
		if withHeader {
			_, _ = fmt.Fprintln(w, strings.Join(names, "\t"))
		}
		for _, item := range arr {
			writeRow(item)
		}
		return nil
	}

	var obj map[string]any
	if json.Unmarshal(b, &obj) == nil {
		if withHeader {
			_, _ = fmt.Fprintln(w, strings.Join(names, "\t"))
		}
		writeRow(obj)
		return nil
	}

	return fmt.Errorf("cannot project fields on type %T", data)
}

// resolveField walks a dot-separated path ("project.shortName") through a nested map.
func resolveField(m map[string]any, path string) string {
	key, rest, nested := strings.Cut(path, ".")
	val := m[key]
	if val == nil {
		return ""
	}
	if nested {
		if sub, ok := val.(map[string]any); ok {
			return resolveField(sub, rest)
		}
		return ""
	}
	return fmt.Sprintf("%v", val)
}
