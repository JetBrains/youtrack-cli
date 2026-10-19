package api_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

func TestCmd_IssueFieldValues(t *testing.T) {
	c := newTestClient(t)
	project := createTempProject(t, c)
	name := "Choices-" + fmt.Sprint(time.Now().UnixNano())
	field, err := c.AddCustomField(name, api.FieldTypeEnumSingle)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteCustomField(field.ID) })
	require.NoError(t, c.AddProjectCustomField(project.ShortName, field.ID))

	var projectFields []struct {
		Field  api.CustomField `json:"field"`
		Bundle api.Bundle      `json:"bundle"`
	}
	require.NoError(t, c.Get("/api/admin/projects/"+project.ID+"/customFields?fields=field(id),bundle(id)&$top=500", &projectFields))
	var bundleID string
	for _, projectField := range projectFields {
		if projectField.Field.ID == field.ID {
			bundleID = projectField.Bundle.ID
		}
	}
	require.NotEmpty(t, bundleID)
	for i := range 12 {
		require.NoError(t, c.AddBundleValue("enum", bundleID, fmt.Sprintf("Choice-%02d", i)))
	}
	var archived struct {
		Archived bool `json:"archived"`
	}
	require.NoError(t, c.Post("/api/admin/customFieldSettings/bundles/enum/"+bundleID+"/values?fields=archived", map[string]any{
		"$type": "EnumBundleElement", "name": "Archived-choice", "archived": true,
	}, &archived))
	require.True(t, archived.Archived)
	textName := name + "-text"
	textField, err := c.AddCustomField(textName, api.FieldTypeText)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteCustomField(textField.ID) })
	require.NoError(t, c.AddProjectCustomField(project.ShortName, textField.ID))
	issue, err := c.AddIssue(api.AddIssueRequest{Summary: "field choices test", Project: &api.Project{ID: project.ID}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })
	issueField, err := c.GetCustomFieldValue(issue.IDReadable, name)
	require.NoError(t, err)

	t.Run("JSON keeps fields and opaque field ID works", func(t *testing.T) {
		values := runJSON[[]map[string]any](t, "issue", "field", "values", issue.IDReadable, issueField.ID, "--limit", "2", "--json")
		require.Len(t, values, 2)
		assert.Equal(t, "Choice-00", values[0]["name"])
		assert.Equal(t, false, values[0]["archived"])
		assert.Equal(t, "EnumBundleElement", values[0]["$type"])
	})

	t.Run("pages do not repeat values", func(t *testing.T) {
		first := runJSON[[]api.IssueFieldValue](t, "issue", "field", "values", issue.IDReadable, name, "--limit", "2", "--json")
		next := runJSON[[]api.IssueFieldValue](t, "issue", "field", "values", issue.IDReadable, name, "--limit", "2", "--offset", "2", "--json")
		require.Len(t, first, 2)
		require.Len(t, next, 2)
		assert.Equal(t, "Choice-02", next[0].Name)
		for _, a := range first {
			for _, b := range next {
				assert.NotEqual(t, a.ID, b.ID)
			}
		}
	})

	t.Run("archived bundle values are not advertised", func(t *testing.T) {
		values := runJSON[[]api.IssueFieldValue](t, "issue", "field", "values", issue.IDReadable, name, "--limit", "50", "--json")
		require.Len(t, values, 12)
		for _, value := range values {
			assert.NotEqual(t, "Archived-choice", value.Name)
		}
	})

	t.Run("table plain and projections", func(t *testing.T) {
		for _, format := range []string{"table", "plain"} {
			out := runCmd(t, "issue", "field", "values", issue.IDReadable, name, "--limit", "2", "--format", format)
			assert.Contains(t, out, "Choice-00")
			assert.Contains(t, out, "Choice-01")
			assert.NotContains(t, out, "Choice-02")
			out = runCmd(t, "issue", "field", "values", issue.IDReadable, name, "--limit", "2", "--format", format, "--fields", "name,archived")
			assert.Contains(t, out, "false")
			assert.Contains(t, out, "Choice-00")
		}
	})

	t.Run("bad enum value retains typed error and bounds choices", func(t *testing.T) {
		out := runCmdExpectError(t, "issue", "field", "set", issue.IDReadable, name, "not-a-choice", "--json")
		var result struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &result))
		assert.Equal(t, "api_error", result.Error.Code, "errors.As must still find the original HTTP 400")
		assert.Contains(t, result.Error.Message, "HTTP 400")
		assert.Contains(t, result.Error.Message, "accessible values")
		assert.Contains(t, result.Error.Message, "Choice-00")
		assert.Contains(t, result.Error.Message, "Choice-09")
		assert.NotContains(t, result.Error.Message, "Choice-10")
		assert.Contains(t, result.Error.Message, "...")
		assert.Contains(t, result.Error.Message, "issue field values")
	})

	t.Run("state field choices", func(t *testing.T) {
		values := runJSON[[]api.IssueFieldValue](t, "issue", "field", "values", issue.IDReadable, "State", "--limit", "2", "--json")
		require.NotEmpty(t, values)
		assert.NotEmpty(t, values[0].Name)
	})

	t.Run("rejects invalid pages", func(t *testing.T) {
		for _, args := range [][]string{{"--limit", "0"}, {"--offset", "-1"}} {
			out := runCmdExpectError(t, append([]string{"issue", "field", "values", issue.IDReadable, name}, args...)...)
			assert.Contains(t, out, "limit must be positive and offset must not be negative")
		}
	})

	t.Run("missing field is reported", func(t *testing.T) {
		out := runCmdExpectError(t, "issue", "field", "values", issue.IDReadable, "no-such-field")
		assert.Contains(t, out, "not found on issue")
	})

	t.Run("scalar fields do not advertise choices", func(t *testing.T) {
		out := runCmdExpectError(t, "issue", "field", "values", issue.IDReadable, textName)
		assert.Contains(t, out, "does not expose selectable values")
	})

	t.Run("missing issue preserves server failure", func(t *testing.T) {
		out := runCmdExpectError(t, "issue", "field", "values", project.ShortName+"-999999", name)
		assert.Contains(t, out, "HTTP 404")
		assert.NotContains(t, out, "accessible values")
	})
}
