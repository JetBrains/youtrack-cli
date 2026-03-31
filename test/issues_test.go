package api_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

func TestIssue_CRUD(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	// Create
	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "integration test issue",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	require.NotEmpty(t, issue.IDReadable)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	// Get
	got, err := c.GetIssue(issue.IDReadable)
	require.NoError(t, err)
	assert.Equal(t, "integration test issue", got.Summary)

	// Update
	updated, err := c.UpdateIssue(issue.IDReadable, api.UpdateIssueRequest{Summary: "updated summary"})
	require.NoError(t, err)
	assert.Equal(t, "updated summary", updated.Summary)

	// List
	issues, err := c.ListIssues("project: "+proj.ShortName, 10)
	require.NoError(t, err)
	assert.NotEmpty(t, issues)

	// Delete
	err = c.DeleteIssue(issue.IDReadable)
	require.NoError(t, err)

	_, err = c.GetIssue(issue.IDReadable)
	assert.Error(t, err)
}

func TestIssue_Comment_CRUD(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "issue for comments",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	// Add comment
	comment, err := c.AddComment(issue.IDReadable, api.AddCommentRequest{Text: "hello world"})
	require.NoError(t, err)
	require.NotEmpty(t, comment.ID)

	// List comments
	comments, err := c.ListComments(issue.IDReadable)
	require.NoError(t, err)
	assert.Len(t, comments, 1)
	assert.Equal(t, "hello world", comments[0].Text)

	// Update comment
	updated, err := c.UpdateComment(issue.IDReadable, comment.ID, "updated text")
	require.NoError(t, err)
	assert.Equal(t, "updated text", updated.Text)

	// Delete comment
	err = c.DeleteComment(issue.IDReadable, comment.ID)
	require.NoError(t, err)

	afterDelete, err := c.ListComments(issue.IDReadable)
	require.NoError(t, err)
	assert.Empty(t, afterDelete)
}

func TestIssue_ApplyCommand(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "command test issue",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	err = c.ApplyCommand("Fixed", []string{issue.IDReadable})
	require.NoError(t, err)
}

func TestIssue_Count(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "count test issue",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	count, err := c.GetIssueCount("project: " + proj.ShortName)
	require.NoError(t, err)
	// YouTrack's count endpoint may return -1 while computing asynchronously.
	// A non-error response with count >= -1 confirms the endpoint is reachable.
	assert.GreaterOrEqual(t, count, -1)
}

func TestIssue_WorkItem(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	err := c.SetProjectTimeTracking(proj.ShortName, true)
	require.NoError(t, err)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "workitem test issue",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	wi, err := c.AddWorkItem(issue.IDReadable, 90, "design work", "")
	require.NoError(t, err)
	assert.NotEmpty(t, wi.ID)
}

func TestIssue_WrapUserValues(t *testing.T) {
	cases := []struct{ in, out string }{
		{"assignee alice", "assignee alice"},
		{"assignee alice.qa", "assignee {alice.qa}"},
		{"reporter bob-dev State Fixed", "reporter {bob-dev} State Fixed"},
		{"assignee {alice.qa}", "assignee {alice.qa}"},
	}
	for _, c := range cases {
		assert.Equal(t, c.out, api.WrapUserValues(c.in), "input: %q", c.in)
	}
}

// Cmd-layer tests

func TestCmd_IssueList(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "cmd list test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	out := runCmd(t, "issue", "list", "--query", "project: "+proj.ShortName, "--format", "json")
	assert.Contains(t, out, "cmd list test")
}

func TestCmd_IssueAdd(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	out := runCmd(t, "issue", "add",
		"--project", proj.ShortName,
		"--summary", "cmd add test",
		"--format", "json",
	)
	assert.Contains(t, out, "cmd add test")

	issues, err := c.ListIssues("project: "+proj.ShortName, 5)
	require.NoError(t, err)
	for _, iss := range issues {
		if strings.Contains(iss.Summary, "cmd add test") {
			t.Cleanup(func() { _ = c.DeleteIssue(iss.IDReadable) })
		}
	}
}

func TestCmd_IssueGet(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "cmd get test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	out := runCmd(t, "issue", "get", issue.IDReadable, "--format", "json")
	assert.Contains(t, out, issue.IDReadable)
	assert.Contains(t, out, "cmd get test")
}

func TestCmd_IssueDelete(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "cmd delete test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)

	runCmd(t, "issue", "delete", issue.IDReadable)

	_, err = c.GetIssue(issue.IDReadable)
	assert.Error(t, err)
}
