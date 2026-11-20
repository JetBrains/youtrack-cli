package api_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

func TestCmd_IssueFieldList(t *testing.T) {
	c := newTestClient(t)
	project := createTempProject(t, c)
	attachDefaultProjectFields(t, c, project, "State")
	suffix := fmt.Sprint(time.Now().UnixNano())
	required := "Severity-" + suffix
	multi := "Labels-" + suffix
	text := "Notes-" + suffix
	due := "Due-" + suffix
	addField := func(name, fieldType string) string {
		field, err := c.AddCustomField(name, fieldType)
		require.NoError(t, err)
		t.Cleanup(func() { _ = c.DeleteCustomField(field.ID) })
		require.NoError(t, c.AddProjectCustomField(project.ShortName, field.ID))
		return field.ID
	}
	requiredID := addField(required, api.FieldTypeEnumSingle)
	addField(multi, api.FieldTypeEnumMulti)
	addField(text, api.FieldTypeText)
	addField(due, api.FieldTypeDate)

	var projectFields []struct {
		ID     string          `json:"id"`
		Type   string          `json:"$type"`
		Field  api.CustomField `json:"field"`
		Bundle api.Bundle      `json:"bundle"`
	}
	require.NoError(t, c.Get("/api/admin/projects/"+project.ID+"/customFields?fields=id,$type,field(id),bundle(id)&$top=500", &projectFields))
	var projectFieldID, projectFieldType, bundleID string
	for _, pf := range projectFields {
		if pf.Field.ID == requiredID {
			projectFieldID, projectFieldType, bundleID = pf.ID, pf.Type, pf.Bundle.ID
		}
	}
	require.NotEmpty(t, bundleID)
	require.NoError(t, c.AddBundleValue("enum", bundleID, "High"))

	issue, err := c.AddIssue(api.AddIssueRequest{Summary: "field list test", Project: &api.Project{ID: project.ID}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })
	awaitIssueFields(t, c, issue.IDReadable, required, multi, text, due, "State")
	require.NoError(t, c.Post("/api/admin/projects/"+project.ID+"/customFields/"+projectFieldID,
		map[string]any{"$type": projectFieldType, "canBeEmpty": false, "emptyFieldText": "No severity"}, nil))
	runCmd(t, "issue", "field", "set", issue.IDReadable, required, "High")
	// issue field set has no date type; a date field takes epoch milliseconds (noon UTC for a date-only field).
	dueMillis := time.Date(2026, 1, 31, 12, 0, 0, 0, time.UTC).UnixMilli()
	runCmd(t, "issue", "field", "set", issue.IDReadable, due, fmt.Sprint(dueMillis), "--type", "int")
	runCmd(t, "issue", "field", "set", issue.IDReadable, text, "first line\nsecond\tline", "--type", "text")

	byName := func(fields []api.IssueFieldInfo) map[string]api.IssueFieldInfo {
		out := map[string]api.IssueFieldInfo{}
		for _, f := range fields {
			out[f.Name] = f
		}
		return out
	}

	t.Run("JSON says type, multi, required, choices and value", func(t *testing.T) {
		fields := byName(runJSON[[]api.IssueFieldInfo](t, "issue", "field", "list", issue.IDReadable, "--json"))
		sev := fields[required]
		assert.NotEmpty(t, sev.ID)
		assert.Equal(t, "enum", sev.ValueType)
		assert.False(t, sev.MultiValue)
		assert.True(t, sev.Required)
		assert.True(t, sev.HasChoices)
		assert.Equal(t, "No severity", sev.EmptyText)
		assert.Contains(t, fmt.Sprint(sev.Value), "High")

		assert.True(t, fields[multi].MultiValue)
		assert.False(t, fields[multi].Required)
		assert.Equal(t, "text", fields[text].ValueType)
		assert.False(t, fields[text].HasChoices)
		assert.Equal(t, "state", fields["State"].ValueType)
		assert.True(t, fields["State"].HasChoices)
	})

	// row matches a table or plain line whose cells are exactly these; the last cell ends the line.
	row := func(cells ...string) string {
		quoted := make([]string, len(cells))
		for i, cell := range cells {
			quoted[i] = regexp.QuoteMeta(cell)
		}
		return `(?m)^` + strings.Join(quoted, `\s+`) + `\s*$`
	}

	t.Run("table and plain show name, type, required and value", func(t *testing.T) {
		for _, format := range []string{"table", "plain"} {
			out := runCmd(t, "issue", "field", "list", issue.IDReadable, "--format", format)
			if format == "table" {
				assert.Regexp(t, row("NAME", "TYPE", "REQUIRED", "VALUE"), out)
			}
			assert.Regexp(t, row(required, "enum", "yes", "High"), out, format)
			assert.Regexp(t, row(multi, "enum[*]", "no"), out, "%s: an unset field has an empty value", format)
			assert.Regexp(t, row(due, "date", "no", "2026-01-31"), out, format)
			assert.Regexp(t, row(text, "text", "no", "first line second line"), out, "%s: a multi-line value stays on its row", format)
			assert.NotRegexp(t, `(?m)^second`, out, format)
		}
	})

	t.Run("a draft shows the fields a new issue needs", func(t *testing.T) {
		draft, err := c.CreateDraft(project.ID, "", "")
		require.NoError(t, err)
		t.Cleanup(func() { _ = c.DeleteDraft(draft.ID) })
		fields := byName(runJSON[[]api.IssueFieldInfo](t, "issue", "field", "list", draft.ID, "--json"))
		require.Contains(t, fields, required)
		assert.True(t, fields[required].Required)
		assert.Nil(t, fields[required].Value)
		assert.Regexp(t, row(required, "enum", "yes"), runCmd(t, "issue", "field", "list", draft.ID, "--format", "plain"),
			"a required field without a value does not look set")
	})

	t.Run("an unknown issue fails", func(t *testing.T) {
		out := runCmdExpectError(t, "issue", "field", "list", project.ShortName+"-999999", "--json")
		assert.Contains(t, out, "404")
	})
}
