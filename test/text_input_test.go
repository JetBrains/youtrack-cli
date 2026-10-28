package api_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

// Multi-line markdown with characters a shell would mangle, so stdin must arrive byte for byte.
const stdinText = "## Steps\n\n1. run `make` with $HOME set\n2. see \"error\" & quit\n"

func TestCmd_IssueDescriptionFromStdin(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	created := runJSON[api.Issue](t, "issue", "create", "--project", proj.ShortName, "--summary", "stdin description",
		"--description-file", "-", "--json")
	require.NotEmpty(t, created.IDReadable)
	t.Cleanup(func() { _ = c.DeleteIssue(created.IDReadable) })
	assert.Empty(t, created.Description, "nothing was piped, so the description is empty")

	runCmdWithInput(t, stdinText, "issue", "update", created.IDReadable, "--description-file", "-", "--json")
	got, err := c.GetIssue(created.IDReadable)
	require.NoError(t, err)
	assert.Equal(t, stdinText, got.Description)

	file := filepath.Join(t.TempDir(), "desc.md")
	require.NoError(t, os.WriteFile(file, []byte("from a file"), 0o600))
	runCmd(t, "issue", "update", created.IDReadable, "--description-file", file, "--json")
	got, err = c.GetIssue(created.IDReadable)
	require.NoError(t, err)
	assert.Equal(t, "from a file", got.Description)

	out, err := runCmdFail(t, "issue", "update", created.IDReadable, "--description", "x", "--description-file", file)
	require.Error(t, err)
	assert.Contains(t, out, "description")
}

func TestCmd_IssueCreateDescriptionFromStdin(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	out := runCmdWithInput(t, stdinText, "issue", "create", "--project", proj.ShortName, "--summary", "stdin create",
		"--description-file", "-", "--json")
	var second api.Issue
	require.NoError(t, json.Unmarshal([]byte(out), &second))
	t.Cleanup(func() { _ = c.DeleteIssue(second.IDReadable) })
	got, err := c.GetIssue(second.IDReadable)
	require.NoError(t, err)
	assert.Equal(t, stdinText, got.Description)
}

func TestCmd_IssueDraftDescriptionFromStdin(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	out := runCmdWithInput(t, stdinText, "issue", "draft", "create", "--project", proj.ShortName, "--summary", "stdin draft",
		"--description-file", "-", "--json")
	var draft api.IssueDraft
	require.NoError(t, json.Unmarshal([]byte(out), &draft))
	t.Cleanup(func() { _ = c.DeleteDraft(draft.ID) })
	got, err := c.GetDraft(draft.ID)
	require.NoError(t, err)
	assert.Equal(t, stdinText, got.Description)

	runCmdWithInput(t, "updated from stdin", "issue", "draft", "update", draft.ID, "--description-file", "-", "--json")
	got, err = c.GetDraft(draft.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated from stdin", got.Description)

	runCmdWithInput(t, "", "issue", "draft", "update", draft.ID, "--description-file", "-", "--json")
	got, err = c.GetDraft(draft.ID)
	require.NoError(t, err)
	assert.Empty(t, got.Description, "empty stdin clears the description")
}

func TestCmd_IssueCommentTextFromFileOrStdin(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	issue, err := c.AddIssue(api.AddIssueRequest{Summary: "comment text input", Project: &api.Project{ID: proj.ID}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	out := runCmdWithInput(t, stdinText, "issue", "comment", "create", issue.IDReadable, "--text-file", "-", "--json")
	var comment api.Comment
	require.NoError(t, json.Unmarshal([]byte(out), &comment))
	got, err := c.GetComment(issue.IDReadable, comment.ID)
	require.NoError(t, err)
	assert.Equal(t, stdinText, got.Text)

	file := filepath.Join(t.TempDir(), "comment.md")
	require.NoError(t, os.WriteFile(file, []byte("edited from a file"), 0o600))
	runCmd(t, "issue", "comment", "update", issue.IDReadable, comment.ID, "--text-file", file, "--json")
	got, err = c.GetComment(issue.IDReadable, comment.ID)
	require.NoError(t, err)
	assert.Equal(t, "edited from a file", got.Text)

	for _, args := range [][]string{
		{"issue", "comment", "create", issue.IDReadable},
		{"issue", "comment", "create", issue.IDReadable, "--text", "x", "--text-file", file},
		{"issue", "comment", "update", issue.IDReadable, comment.ID},
		{"issue", "comment", "create", issue.IDReadable, "--text-file", filepath.Join(t.TempDir(), "missing.md")},
	} {
		out, err := runCmdFail(t, args...)
		assert.Error(t, err, "%v should fail: %s", args, out)
	}
	comments, err := c.ListComments(issue.IDReadable)
	require.NoError(t, err)
	assert.Len(t, comments, 1, "rejected calls must not post comments")
}
