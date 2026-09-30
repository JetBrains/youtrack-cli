package internal

import (
	"reflect"
	"strings"
	"testing"

	"github.com/JetBrains/youtrack-cli/api"
)

func TestIssueColumns(t *testing.T) {
	attr := func(spec string) IssueColumn { return IssueColumn{Spec: spec} }
	custom := func(name string) IssueColumn { return IssueColumn{Spec: name, Custom: true} }
	cases := []struct {
		name string
		in   string
		out  []IssueColumn
	}{
		{"empty input has no columns", "", nil},
		{"blank input has no columns", "  ", nil},
		{"attributes stay attributes", "idReadable,summary", []IssueColumn{attr("idReadable"), attr("summary")}},
		{"unknown names are custom fields in order", "Priority,state,summary", []IssueColumn{custom("Priority"), custom("state"), attr("summary")}},
		{"attribute casing is canonicalized", "IDREADABLE,Summary,numberinproject", []IssueColumn{attr("idReadable"), attr("summary"), attr("numberInProject")}},
		{"dot path keeps its tail", "Project.shortName", []IssueColumn{attr("project.shortName")}},
		{"sub-field spec keeps its tail", "comments(id,text),customFields(name)", []IssueColumn{attr("comments(id,text)"), attr("customFields(name)")}},
		{"custom field names keep spaces and casing", " Fix versions , Due Date ", []IssueColumn{custom("Fix versions"), custom("Due Date")}},
		{"empty entries are dropped", "summary,,Priority,", []IssueColumn{attr("summary"), custom("Priority")}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IssueColumns(c.in); !reflect.DeepEqual(got, c.out) {
				t.Errorf("IssueColumns(%q) = %+v, want %+v", c.in, got, c.out)
			}
		})
	}
}

func TestIssueColumnsSplit(t *testing.T) {
	cols := IssueColumns("idReadable,Priority,reporter,Fix versions,project.shortName,Due-Date")
	if got, want := IssueRequestFields(cols), "idReadable,Priority,reporter(id,login,fullName),project.shortName"; got != want {
		t.Errorf("IssueRequestFields = %q, want %q", got, want)
	}
	if got, want := IssueCustomFieldNames(cols), []string{"Priority", "Fix versions", "Due-Date"}; !reflect.DeepEqual(got, want) {
		t.Errorf("IssueCustomFieldNames = %v, want %v", got, want)
	}
	if got := IssueRequestFields(IssueColumns("Fix versions")); got != "" {
		t.Errorf("IssueRequestFields with only a spaced custom field = %q, want empty", got)
	}
}

// Every attribute api.Issue models must count as an attribute, or --fields would look it up as a custom field.
func TestIssueAttributesCoverIssueStruct(t *testing.T) {
	typ := reflect.TypeOf(api.Issue{})
	for i := 0; i < typ.NumField(); i++ {
		name, _, _ := strings.Cut(typ.Field(i).Tag.Get("json"), ",")
		if name == "" || name == "-" {
			continue
		}
		if _, ok := issueAttributes[strings.ToLower(name)]; !ok {
			t.Errorf("api.Issue field %q is missing from issueAttributes", name)
		}
	}
}
