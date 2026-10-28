package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
)

// Format resolves the effective output format, shared with confirmation output.
func Format(cmd *cobra.Command) string {
	return internal.OutputFormat(cmd)
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
	names := internal.SplitFields(fields)
	return writeProjection(w, data, headerNames(names), withHeader, func(obj map[string]any) []string {
		vals := make([]string, len(names))
		for i, name := range names {
			vals[i] = resolveField(obj, name)
		}
		return vals
	})
}

// maskDeletedCommentText only changes decoded copies used for human output.
// Type discriminators identify nested comments without changing unrelated
// objects that happen to expose deleted and text attributes.
func maskDeletedCommentText(value any, comment bool) {
	switch node := value.(type) {
	case map[string]any:
		entityType, _ := node["$type"].(string)
		if comment || entityType == "IssueComment" || entityType == "ArticleComment" {
			if deleted, _ := node["deleted"].(bool); deleted {
				node["text"] = "[deleted]"
				node["textPreview"] = "[deleted]"
			}
		}
		for _, child := range node {
			maskDeletedCommentText(child, false)
		}
	case []any:
		for _, child := range node {
			maskDeletedCommentText(child, comment)
		}
	}
}

// writeProjection JSON round-trips data (a slice or a single object) to maps and writes one tab-separated row per
// object, with cells computed by row and an optional header line.
func writeProjection(w io.Writer, data any, headers []string, withHeader bool, row func(map[string]any) []string) error {
	w = safeWriter{w}

	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	commentRows := false
	switch data.(type) {
	case []api.Comment, *api.Comment:
		commentRows = true
	}

	writeRow := func(obj map[string]any) {
		maskDeletedCommentText(obj, commentRows)
		vals := row(obj)
		if withHeader {
			// Collapse newlines so the tab-separated row stays on one line,
			// but do not truncate — callers control width via terminal/pager.
			for i := range vals {
				vals[i] = firstLine(vals[i])
			}
		}
		_, _ = fmt.Fprintln(w, strings.Join(vals, "\t"))
	}

	var arr []map[string]any
	if json.Unmarshal(b, &arr) == nil {
		if withHeader {
			_, _ = fmt.Fprintln(w, strings.Join(headers, "\t"))
		}
		for _, item := range arr {
			writeRow(item)
		}
		return nil
	}

	var obj map[string]any
	if json.Unmarshal(b, &obj) == nil {
		if withHeader {
			_, _ = fmt.Fprintln(w, strings.Join(headers, "\t"))
		}
		writeRow(obj)
		return nil
	}

	return fmt.Errorf("cannot project fields on type %T", data)
}

// PrintIssueColumns renders issues with the columns chosen by `issue list --fields`: attribute columns resolve like
// any --fields projection, custom-field columns show the value of the issue's custom field with that name.
// JSON output is the issues unchanged, exactly as Print would emit them.
func PrintIssueColumns(cmd *cobra.Command, issues []api.Issue, cols []internal.IssueColumn) error {
	w := cmd.OutOrStdout()
	format := Format(cmd)
	if format == "json" {
		return printJSON(w, issues)
	}

	specs := make([]string, len(cols))
	for i, c := range cols {
		specs[i] = c.Spec
	}
	headers := headerNames(specs)
	for i, c := range cols {
		if c.Custom {
			headers[i] = strings.ToUpper(c.Spec)
		}
	}
	row := func(obj map[string]any) []string {
		vals := make([]string, len(cols))
		for i, c := range cols {
			if c.Custom {
				if field := findCustomField(obj, c.Spec); field != nil {
					vals[i] = customFieldCell(field)
					continue
				}
			}
			// Attributes, and custom names no custom field matched — those may be attributes the server returned.
			vals[i] = resolveField(obj, c.Spec)
		}
		return vals
	}

	if format == "plain" {
		return writeProjection(w, issues, headers, false, row)
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	err := writeProjection(tw, issues, headers, true, row)
	_ = tw.Flush()
	return err
}

// findCustomField returns the custom field named name (case-insensitive) from an issue's customFields, or nil.
func findCustomField(issue map[string]any, name string) map[string]any {
	fields, _ := issue["customFields"].([]any)
	for _, f := range fields {
		if m, ok := f.(map[string]any); ok {
			if n, _ := m["name"].(string); strings.EqualFold(n, name) {
				return m
			}
		}
	}
	return nil
}

// customFieldCell renders an issue custom field for a table cell; date-only fields drop the time of day, and unset
// fields show the project's empty field text ("Unassigned"), like the issue list UI.
func customFieldCell(field map[string]any) string {
	if field == nil {
		return ""
	}
	if ms, ok := field["value"].(float64); ok && field["$type"] == "DateIssueCustomField" {
		return time.UnixMilli(int64(ms)).UTC().Format("2006-01-02")
	}
	if s := customFieldValue(field["value"]); s != "" {
		return s
	}
	pcf, _ := field["projectCustomField"].(map[string]any)
	empty, _ := pcf["emptyFieldText"].(string)
	return empty
}

// customFieldValue converts a custom field value to a display string: text and period values show their text or
// presentation, entity values (enum, state, user, version) their identifier, multi-value fields join with ", ",
// and date fields render as timestamps.
func customFieldValue(val any) string {
	switch v := val.(type) {
	case nil:
		return ""
	case map[string]any:
		for _, key := range []string{"presentation", "text"} {
			if s, ok := v[key].(string); ok && s != "" {
				return s
			}
		}
		if s := compactObject(v); s[0] != '{' {
			return s
		}
		return ""
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			if s := customFieldValue(item); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, ", ")
	default:
		return formatValue(v)
	}
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
	subNames := internal.SplitFields(innerSpec)
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
// Sub-field specs are uppercased and preserved: "comments(id,text)" → "COMMENTS(ID,TEXT)".
// When both "id" and "idReadable" are present, idReadable is labeled KEY
// to distinguish the human-readable project key from the internal opaque ID.
func headerNames(names []string) []string {
	hasID := false
	for _, n := range names {
		if k, _ := parseFieldSpec(n); k == "id" {
			hasID = true
			break
		}
	}
	out := make([]string, len(names))
	for i, n := range names {
		outerKey, innerSpec := parseFieldSpec(n)
		var label string
		switch outerKey {
		case "idReadable":
			if hasID {
				label = "KEY"
			} else {
				label = "ID"
			}
		default:
			label = strings.ToUpper(outerKey)
		}
		if innerSpec != "" {
			label += "(" + strings.ToUpper(innerSpec) + ")"
		}
		out[i] = label
	}
	return out
}

// firstLine returns the first non-empty line of s with surrounding whitespace
// stripped. Used in projected table output to keep rows single-line without
// truncating the value.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
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
