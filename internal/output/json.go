package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

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
			tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
			err := printProjected(tw, data, fields, true)
			_ = tw.Flush()
			return err
		}
		return PrintTable(w, data)
	}
}

func printJSON(w io.Writer, data any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// splitFields splits a comma-separated field spec string while respecting
// nested parentheses, so "id,comments(id,text),summary" yields three entries
// rather than four.
func splitFields(s string) []string {
	var parts []string
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	parts = append(parts, strings.TrimSpace(s[start:]))
	return parts
}

// parseFieldSpec splits a spec like "comments(id,text)" into outer key
// "comments" and inner spec "id,text". Returns the spec unchanged as key
// with empty innerSpec when no parentheses are present.
func parseFieldSpec(spec string) (key, innerSpec string) {
	i := strings.IndexByte(spec, '(')
	if i < 0 {
		return spec, ""
	}
	return spec[:i], strings.TrimSuffix(spec[i+1:], ")")
}

// printProjected renders only the fields named in the comma-separated `fields`
// string. It works on any type by doing a JSON round-trip to a map, then
// resolving dot-separated paths (e.g. "project.shortName") or sub-field
// specs (e.g. "comments(id,text)").
func printProjected(w io.Writer, data any, fields string, withHeader bool) error {
	names := splitFields(fields)

	b, err := json.Marshal(data)
	if err != nil {
		return err
	}

	writeRow := func(obj map[string]any) {
		vals := make([]string, len(names))
		for i, name := range names {
			v := resolveField(obj, name)
			if withHeader {
				v = singleLine(v, 72)
			}
			vals[i] = v
		}
		_, _ = fmt.Fprintln(w, strings.Join(vals, "\t"))
	}

	var arr []map[string]any
	if json.Unmarshal(b, &arr) == nil {
		if withHeader {
			_, _ = fmt.Fprintln(w, strings.Join(headerNames(names), "\t"))
		}
		for _, item := range arr {
			writeRow(item)
		}
		return nil
	}

	var obj map[string]any
	if json.Unmarshal(b, &obj) == nil {
		if withHeader {
			_, _ = fmt.Fprintln(w, strings.Join(headerNames(names), "\t"))
		}
		writeRow(obj)
		return nil
	}

	return fmt.Errorf("cannot project fields on type %T", data)
}

// resolveField resolves a field spec against a map. Supports:
//   - plain key:              "summary"
//   - dot path:               "project.shortName"
//   - sub-field spec:         "comments(id,text)"  — applies inner spec to each element
func resolveField(m map[string]any, spec string) string {
	// Dot-path navigation takes priority: "project.shortName"
	if key, rest, hasDot := strings.Cut(spec, "."); hasDot {
		val := m[key]
		if sub, ok := val.(map[string]any); ok {
			return resolveField(sub, rest)
		}
		return ""
	}

	outerKey, innerSpec := parseFieldSpec(spec)
	val := m[outerKey]
	if val == nil {
		return ""
	}
	if innerSpec == "" {
		return formatValue(val)
	}

	// Sub-field spec: apply innerSpec to the resolved value.
	subNames := splitFields(innerSpec)
	switch v := val.(type) {
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			if sub, ok := item.(map[string]any); ok {
				if s := resolveSubFields(sub, subNames); s != "" {
					parts = append(parts, s)
				}
			}
		}
		return strings.Join(parts, ", ")
	case map[string]any:
		return resolveSubFields(v, subNames)
	}
	return formatValue(val)
}

// resolveSubFields resolves multiple sub-field names against a map and joins
// the non-empty results with a space.
func resolveSubFields(m map[string]any, subNames []string) string {
	vals := make([]string, 0, len(subNames))
	for _, sn := range subNames {
		if sv := resolveField(m, sn); sv != "" {
			vals = append(vals, sv)
		}
	}
	return strings.Join(vals, " ")
}

// headerNames converts field specs to display column headers.
// Sub-field specs are stripped: "comments(id,text)" → "COMMENTS".
// When both "id" and "idReadable" are present, idReadable is labeled KEY
// to distinguish the human-readable project key from the internal opaque ID.
func headerNames(names []string) []string {
	// Resolve outer keys first (strip sub-field specs) for the hasID check.
	keys := make([]string, len(names))
	for i, n := range names {
		keys[i], _ = parseFieldSpec(n)
	}
	hasID := false
	for _, k := range keys {
		if k == "id" {
			hasID = true
			break
		}
	}
	out := make([]string, len(keys))
	for i, k := range keys {
		switch k {
		case "idReadable":
			if hasID {
				out[i] = "KEY"
			} else {
				out[i] = "ID"
			}
		default:
			out[i] = strings.ToUpper(k)
		}
	}
	return out
}

// singleLine collapses a multi-line string to its first non-empty line,
// then truncates to max runes with "…" if needed.
func singleLine(s string, max int) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	r := []rune(s)
	if len(r) > max {
		return string(r[:max-1]) + "…"
	}
	return s
}

// formatValue converts a JSON-unmarshalled value to a display string.
// Nested objects are compacted to their most useful identifier field.
// Arrays of named objects (tags, links) join their identifiers; arrays of
// opaque objects (comments, attachments) show a count instead.
// float64 integers in the millisecond-epoch range are rendered as timestamps.
func formatValue(val any) string {
	switch v := val.(type) {
	case map[string]any:
		return compactObject(v)
	case []any:
		return formatSlice(v)
	case float64:
		n := int64(v)
		if float64(n) != v {
			return strconv.FormatFloat(v, 'f', -1, 64)
		}
		// millisecond epoch: 2001-09-09 to 2286-11-20
		if n > 1_000_000_000_000 && n < 10_000_000_000_000 {
			return time.UnixMilli(n).UTC().Format("2006-01-02 15:04:05")
		}
		return strconv.FormatInt(n, 10)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// formatSlice renders an array field for table display.
// If every element compacts to a meaningful identifier, they are joined with ", ".
// If any element falls back to raw JSON (no recognizable identifier key), the
// whole array is represented as a count — e.g. "3" for three comments.
func formatSlice(items []any) string {
	if len(items) == 0 {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		switch m := item.(type) {
		case map[string]any:
			s := compactObject(m)
			if len(s) > 0 && s[0] == '{' {
				// compactObject fell back to JSON — no useful identifier available;
				// show a count for the whole array instead of unreadable output.
				return strconv.Itoa(len(items))
			}
			parts = append(parts, s)
		default:
			parts = append(parts, fmt.Sprintf("%v", item))
		}
	}
	return strings.Join(parts, ", ")
}

// compactObject extracts a human-readable identifier from a nested API object.
// It tries common identifier keys in priority order; falls back to compact JSON.
func compactObject(m map[string]any) string {
	for _, key := range []string{"login", "name", "shortName", "fullName", "idReadable", "id"} {
		if s, ok := m[key].(string); ok && s != "" {
			return s
		}
	}
	b, _ := json.Marshal(m)
	return string(b)
}
