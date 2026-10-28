package api

import (
	"encoding/json"
	"testing"
)

func schema(t *testing.T, name, valueType string, multi bool) ProjectFieldSchema {
	t.Helper()
	var s ProjectFieldSchema
	raw := `{"field":{"name":"` + name + `","fieldType":{"valueType":"` + valueType + `","isMultiValue":` +
		map[bool]string{true: "true", false: "false"}[multi] + `}}}`
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestBuildIssueCustomField(t *testing.T) {
	cases := []struct {
		name, valueType string
		multi           bool
		value, want     string
	}{
		{"Type", "enum", false, "Task", `{"name":"Type","$type":"SingleEnumIssueCustomField","value":{"name":"Task"}}`},
		{"Tags", "enum", true, "UI", `{"name":"Tags","$type":"MultiEnumIssueCustomField","value":[{"name":"UI"}]}`},
		{"Tags", "enum", true, "", `{"name":"Tags","$type":"MultiEnumIssueCustomField","value":[]}`},
		{"State", "state", false, "Open", `{"name":"State","$type":"StateIssueCustomField","value":{"name":"Open"}}`},
		{"Assignee", "user", false, "alice.qa", `{"name":"Assignee","$type":"SingleUserIssueCustomField","value":{"login":"alice.qa"}}`},
		{"Assignee", "user", false, "", `{"name":"Assignee","$type":"SingleUserIssueCustomField","value":null}`},
		{"Reviewers", "user", true, "bob", `{"name":"Reviewers","$type":"MultiUserIssueCustomField","value":[{"login":"bob"}]}`},
		{"Subsystem", "ownedField", false, "UI", `{"name":"Subsystem","$type":"SingleOwnedIssueCustomField","value":{"name":"UI"}}`},
		{"Fix versions", "version", true, "1.0", `{"name":"Fix versions","$type":"MultiVersionIssueCustomField","value":[{"name":"1.0"}]}`},
		{"Fixed in build", "build", false, "42", `{"name":"Fixed in build","$type":"SingleBuildIssueCustomField","value":{"name":"42"}}`},
		{"Team", "group", false, "Core", `{"name":"Team","$type":"SingleGroupIssueCustomField","value":{"name":"Core"}}`},
		{"Notes", "text", false, "hi", `{"name":"Notes","$type":"TextIssueCustomField","value":{"text":"hi"}}`},
		{"Estimation", "period", false, "1d 2h", `{"name":"Estimation","$type":"PeriodIssueCustomField","value":{"presentation":"1d 2h"}}`},
		{"Ref", "string", false, "abc", `{"name":"Ref","$type":"SimpleIssueCustomField","value":"abc"}`},
		{"Count", "integer", false, "7", `{"name":"Count","$type":"SimpleIssueCustomField","value":7}`},
		{"Score", "float", false, "1.5", `{"name":"Score","$type":"SimpleIssueCustomField","value":1.5}`},
		{"Due", "date", false, "2026-01-31", `{"name":"Due","$type":"DateIssueCustomField","value":1769860800000}`},
		{"At", "date and time", false, "2026-01-31", `{"name":"At","$type":"SimpleIssueCustomField","value":1769817600000}`},
		{"At", "date and time", false, "2026-01-31T12:00:00Z", `{"name":"At","$type":"SimpleIssueCustomField","value":1769860800000}`},
		{"At", "date and time", false, "1769860800000", `{"name":"At","$type":"SimpleIssueCustomField","value":1769860800000}`},
	}
	for _, c := range cases {
		t.Run(c.name+"/"+c.value, func(t *testing.T) {
			field, err := BuildIssueCustomField(schema(t, c.name, c.valueType, c.multi), c.value)
			if err != nil {
				t.Fatal(err)
			}
			got, err := json.Marshal(field)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != c.want {
				t.Errorf("got  %s\nwant %s", got, c.want)
			}
		})
	}
}

func TestBuildIssueCustomFieldRejectsBadValues(t *testing.T) {
	for _, c := range []struct{ valueType, value string }{
		{"integer", "seven"},
		{"float", "1,5"},
		{"date", "31.01.2026"},
		{"unknown", "x"},
	} {
		if _, err := BuildIssueCustomField(schema(t, "F", c.valueType, false), c.value); err == nil {
			t.Errorf("%s %q: expected an error", c.valueType, c.value)
		}
	}
}

func TestFindProjectFieldIgnoresCase(t *testing.T) {
	schemas := []ProjectFieldSchema{schema(t, "Fix versions", "version", true)}
	if _, ok := FindProjectField(schemas, "fix VERSIONS"); !ok {
		t.Error("expected a case-insensitive match")
	}
	if _, ok := FindProjectField(schemas, "Fix version"); ok {
		t.Error("expected no match for a different name")
	}
}
