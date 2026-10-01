package output

import "testing"

func TestCustomFieldCell(t *testing.T) {
	cases := []struct {
		name  string
		field map[string]any
		out   string
	}{
		{"missing field is empty", nil, ""},
		{"unset value is empty", map[string]any{"value": nil}, ""},
		{"unset value shows empty field text", map[string]any{"value": nil, "projectCustomField": map[string]any{"emptyFieldText": "Unassigned"}}, "Unassigned"},
		{"empty multi-value shows empty field text", map[string]any{"value": []any{}, "projectCustomField": map[string]any{"emptyFieldText": "No Fix versions"}}, "No Fix versions"},
		{"enum shows name", map[string]any{"value": map[string]any{"name": "Critical", "$type": "EnumBundleElement"}}, "Critical"},
		{"user shows login", map[string]any{"value": map[string]any{"login": "jane", "fullName": "Jane Doe", "name": "Jane Doe"}}, "jane"},
		{"text shows text", map[string]any{"value": map[string]any{"text": "notes", "$type": "TextFieldValue"}}, "notes"},
		{"period shows presentation", map[string]any{"value": map[string]any{"presentation": "1w 2d", "$type": "PeriodValue"}}, "1w 2d"},
		{"value without identifier is empty, not JSON", map[string]any{"value": map[string]any{"$type": "PeriodValue"}}, ""},
		{"multi-value joins names", map[string]any{"value": []any{map[string]any{"name": "1.0"}, map[string]any{"name": "2.0"}}}, "1.0, 2.0"},
		{"empty multi-value is empty", map[string]any{"value": []any{}}, ""},
		{"integer stays an integer", map[string]any{"value": float64(42)}, "42"},
		{"float keeps decimals", map[string]any{"value": 1.5}, "1.5"},
		{"string is verbatim", map[string]any{"value": "hello"}, "hello"},
		{"date field drops time of day", map[string]any{"$type": "DateIssueCustomField", "value": float64(1791892800000)}, "2026-10-13"},
		{"date-time field keeps time", map[string]any{"$type": "SimpleIssueCustomField", "value": float64(1791892800000)}, "2026-10-13 12:00:00"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := customFieldCell(c.field); got != c.out {
				t.Errorf("customFieldCell(%v) = %q, want %q", c.field, got, c.out)
			}
		})
	}
}
