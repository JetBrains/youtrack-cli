package api_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

func TestCmd_IssueFieldsProjection(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	me, err := c.GetCurrentUser()
	require.NoError(t, err)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary:     "fields projection test",
		Description: "test description content",
		Project:     &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	_, err = c.AddComment(issue.IDReadable, api.AddCommentRequest{Text: "test comment"})
	require.NoError(t, err)

	t.Run("scalar fields render as plain values without map[] or scientific notation", func(t *testing.T) {
		out := runCmd(t, "issue", "list",
			"--query", "project: "+proj.ShortName,
			"--fields", "idReadable,summary,created,updated,numberInProject,commentsCount,votes,isDraft",
			"--format", "plain",
		)
		assert.Contains(t, out, issue.IDReadable)
		assert.Contains(t, out, "fields projection test")
		assert.Contains(t, out, "2026-") // created rendered as date, not epoch
		assert.NotContains(t, out, "map[")
		assert.NotContains(t, out, "e+12") // no scientific notation for timestamps
	})

	t.Run("reporter auto-expands and displays login", func(t *testing.T) {
		out := runCmd(t, "issue", "list",
			"--query", "project: "+proj.ShortName,
			"--fields", "idReadable,reporter",
			"--format", "plain",
		)
		assert.Contains(t, out, me.Login)
		assert.NotContains(t, out, "map[")
	})

	t.Run("project auto-expands and displays shortName", func(t *testing.T) {
		out := runCmd(t, "issue", "list",
			"--query", "project: "+proj.ShortName,
			"--fields", "idReadable,project",
			"--format", "plain",
		)
		assert.Contains(t, out, proj.ShortName)
		assert.NotContains(t, out, "map[")
	})

	t.Run("comments array displays as count not raw map", func(t *testing.T) {
		out := runCmd(t, "issue", "list",
			"--query", "project: "+proj.ShortName,
			"--fields", "idReadable,commentsCount,comments",
			"--format", "plain",
		)
		assert.Contains(t, out, issue.IDReadable)
		assert.NotContains(t, out, "map[")
		// commentsCount is a scalar int — 1 after adding the comment above
		assert.Contains(t, out, "1")
	})

	t.Run("id and idReadable together use KEY header to avoid duplicate ID columns", func(t *testing.T) {
		out := runCmd(t, "issue", "list",
			"--query", "project: "+proj.ShortName,
			"--fields", "id,idReadable",
			"--format", "table",
		)
		lines := strings.Split(strings.TrimSpace(out), "\n")
		require.NotEmpty(t, lines)
		header := lines[0]
		assert.Contains(t, header, "KEY")
		assert.Equal(t, 1, strings.Count(header, "ID"), "expected exactly one ID column, got: %q", header)
	})

	t.Run("description renders on single line in table format", func(t *testing.T) {
		out := runCmd(t, "issue", "list",
			"--query", "project: "+proj.ShortName,
			"--fields", "idReadable,description",
			"--format", "table",
		)
		assert.Contains(t, out, "test description content")
		assert.NotContains(t, out, "map[")
	})
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

	runCmd(t, "issue", "delete", issue.IDReadable, "--force")

	_, err = c.GetIssue(issue.IDReadable)
	assert.Error(t, err)
}

func TestCmd_IssueUpdate(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "cmd update original",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	out := runCmd(t, "issue", "update", issue.IDReadable,
		"--summary", "cmd update revised",
		"--description", "updated through cli",
		"--format", "json",
	)
	assert.Contains(t, out, "cmd update revised")

	got, err := c.GetIssue(issue.IDReadable)
	require.NoError(t, err)
	assert.Equal(t, "cmd update revised", got.Summary)
	assert.Equal(t, "updated through cli", got.Description)
}

func TestCmd_IssueCommentCRUD(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "cmd comment test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	out := runCmd(t, "issue", "comment", "add", issue.IDReadable,
		"--text", "hello from cli",
		"--format", "json",
	)
	assert.Contains(t, out, "hello from cli")

	var comment struct {
		ID   string `json:"id"`
		Text string `json:"text"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &comment))
	require.NotEmpty(t, comment.ID)

	out = runCmd(t, "issue", "comment", "list", issue.IDReadable, "--format", "json")
	assert.Contains(t, out, comment.ID)
	assert.Contains(t, out, "hello from cli")

	out = runCmd(t, "issue", "comment", "update", issue.IDReadable, comment.ID,
		"--text", "updated from cli",
		"--format", "json",
	)
	assert.Contains(t, out, "updated from cli")

	comments, err := c.ListComments(issue.IDReadable)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "updated from cli", comments[0].Text)

	out = runCmd(t, "issue", "comment", "delete", issue.IDReadable, comment.ID, "--force")
	assert.Contains(t, out, "deleted comment "+comment.ID)

	comments, err = c.ListComments(issue.IDReadable)
	require.NoError(t, err)
	assert.Empty(t, comments)
}

func TestCmd_IssueWorkitemAdd(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	out := runCmd(t, "project", "timetrack", "set", proj.ShortName, "--enabled")
	assert.Contains(t, out, "updated time tracking for project "+proj.ShortName)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "cmd workitem test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	out = runCmd(t, "issue", "workitem", "add", issue.IDReadable,
		"--minutes", "90",
		"--text", "design work",
		"--format", "json",
	)
	assert.Contains(t, out, "design work")

	var item struct {
		ID       string `json:"id"`
		Text     string `json:"text"`
		Duration struct {
			Minutes int `json:"minutes"`
		} `json:"duration"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &item))
	assert.NotEmpty(t, item.ID)
	assert.Equal(t, "design work", item.Text)
	assert.Equal(t, 90, item.Duration.Minutes)
}

func TestCmd_IssueDelete_Confirm(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "confirm delete test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)

	out := runCmdWithInput(t, "y\n", "issue", "delete", issue.IDReadable)
	assert.Contains(t, out, "deleted issue "+issue.IDReadable)

	_, err = c.GetIssue(issue.IDReadable)
	assert.Error(t, err, "issue should be deleted after confirming with y")
}

func TestCmd_IssueDelete_Cancel(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "cancel delete test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	out := runCmdWithInput(t, "n\n", "issue", "delete", issue.IDReadable)
	assert.Contains(t, out, "Canceled.")

	got, err := c.GetIssue(issue.IDReadable)
	require.NoError(t, err, "issue should still exist after canceling")
	assert.Equal(t, issue.IDReadable, got.IDReadable)
}
