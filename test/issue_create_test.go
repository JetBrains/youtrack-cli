package api_test

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

func requireProjectField(t *testing.T, c *api.Client, proj *api.Project, name string) {
	t.Helper()
	fields, err := c.GetProjectCustomFields(proj.ShortName)
	require.NoError(t, err)
	for _, f := range fields {
		if f.Field != nil && f.Field.Name == name {
			require.NoError(t, c.Post(fmt.Sprintf("/api/admin/projects/%s/customFields/%s", proj.ID, f.ID),
				map[string]any{"canBeEmpty": false}, nil))
			return
		}
	}
	t.Fatalf("project %s has no field %s", proj.ShortName, name)
}

func issueFieldValues(t *testing.T, c *api.Client, id string) map[string]string {
	t.Helper()
	var result struct {
		CustomFields []struct {
			Name  string `json:"name"`
			Value any    `json:"value"`
		} `json:"customFields"`
	}
	require.NoError(t, c.Get("/api/issues/"+url.PathEscape(id)+"?fields=customFields(name,value(name,login))", &result))
	values := map[string]string{}
	for _, f := range result.CustomFields {
		switch v := f.Value.(type) {
		case map[string]any:
			values[f.Name] = fmt.Sprint(v["name"])
			if login, ok := v["login"]; ok {
				values[f.Name] = fmt.Sprint(login)
			}
		case nil:
		default:
			values[f.Name] = fmt.Sprint(v)
		}
	}
	return values
}

// A field the project requires must be accepted when it is passed to create, which needs it in the creating request.
func TestCmd_IssueCreateWithRequiredField(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	requireProjectField(t, c, proj, "Assignee")
	me, err := c.GetCurrentUser()
	require.NoError(t, err)

	out, err := runCmdFail(t, "issue", "create", "--project", proj.ShortName, "--summary", "missing required field")
	require.Error(t, err, "the server should reject an issue without the required field: %s", out)
	assert.Contains(t, out, "Assignee")

	issue := runJSON[api.Issue](t, "issue", "create", "--project", proj.ShortName, "--summary", "required field",
		"--custom-fields", `{"Assignee":"me","Priority":"Major"}`, "--json")
	t.Cleanup(func() { _ = c.DeleteIssue(issue.ID) })
	values := issueFieldValues(t, c, issue.IDReadable)
	assert.Equal(t, me.Login, values["Assignee"])
	assert.Equal(t, "Major", values["Priority"])
}

func TestCmd_IssueCreateTypeAndTags(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	tag, err := c.AddTag(api.AddTagRequest{Name: "e2e-create-" + uniqueProjectShortName("t")})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteTag(tag.ID) })

	issue := runJSON[api.Issue](t, "issue", "create", "--project", proj.ShortName, "--summary", "type and tags",
		"--type", "Task", "--tag", tag.Name, "--json")
	t.Cleanup(func() { _ = c.DeleteIssue(issue.ID) })

	assert.Equal(t, "Task", issueFieldValues(t, c, issue.IDReadable)["Type"])
	tags, err := c.ListIssueTags(issue.IDReadable)
	require.NoError(t, err)
	require.Len(t, tags, 1)
	assert.Equal(t, tag.ID, tags[0].ID)
}

// Inputs that cannot be resolved fail before the create request, so nothing is created or burned.
func TestCmd_IssueCreateRejectsUnresolvableInput(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	for _, args := range [][]string{
		{"--tag", "no-such-tag-e2e"},
		{"--parent", proj.ShortName + "-999"},
		{"--type", "Task", "--custom-fields", `{"type":"Bug"}`},
	} {
		out, err := runCmdFail(t, append([]string{"issue", "create", "--project", proj.ShortName, "--summary", "rejected"}, args...)...)
		assert.Error(t, err, "%v should fail: %s", args, out)
	}
	first := runJSON[api.Issue](t, "issue", "create", "--project", proj.ShortName, "--summary", "first", "--json")
	t.Cleanup(func() { _ = c.DeleteIssue(first.ID) })
	assert.Equal(t, proj.ShortName+"-1", first.IDReadable, "rejected creates must not have used an issue number")
}
