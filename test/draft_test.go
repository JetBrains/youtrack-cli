package api_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

// runCmdFail runs the CLI and returns (combined output, error).
// Does not call t.Fatal on non-zero exit — intended for testing error paths.
func runCmdFail(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Env = append(os.Environ(), "HOME="+t.TempDir())
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}

func TestCmd_IssueDraftCreateAndSubmit(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	// Create draft — template prefills description and fields.
	out := runCmd(t, "issue", "draft", "create",
		"--project", proj.ShortName,
		"--summary", "draft e2e test",
		"--format", "json",
	)

	var draft api.IssueDraft
	require.NoError(t, json.Unmarshal([]byte(out), &draft), "create output: %s", out)
	require.NotEmpty(t, draft.ID, "draft ID must be set")
	assert.Equal(t, "draft e2e test", draft.Summary)

	t.Cleanup(func() {
		// Best-effort cleanup: if the draft was submitted it no longer exists.
		// Ignore the error.
	})

	// Update summary and description.
	out = runCmd(t, "issue", "draft", "update", draft.ID,
		"--summary", "draft e2e test updated",
		"--description", "added via E2E",
		"--format", "json",
	)
	var updated api.IssueDraft
	require.NoError(t, json.Unmarshal([]byte(out), &updated))
	assert.Equal(t, "draft e2e test updated", updated.Summary)
	assert.Equal(t, "added via E2E", updated.Description)

	// Submit draft → real issue.
	out = runCmd(t, "issue", "draft", "submit", draft.ID, "--format", "json")
	var issue api.Issue
	require.NoError(t, json.Unmarshal([]byte(out), &issue))
	require.NotEmpty(t, issue.IDReadable)

	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	// Verify the issue exists with correct summary.
	fetched, err := c.GetIssue(issue.IDReadable)
	require.NoError(t, err)
	assert.Equal(t, "draft e2e test updated", fetched.Summary)
	assert.Equal(t, "added via E2E", fetched.Description)
}

func TestCmd_IssueDraftDelete(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	out := runCmd(t, "issue", "draft", "create",
		"--project", proj.ShortName,
		"--summary", "draft delete e2e test",
		"--format", "json",
	)
	var draft api.IssueDraft
	require.NoError(t, json.Unmarshal([]byte(out), &draft), "create output: %s", out)
	require.NotEmpty(t, draft.ID, "draft ID must be set")

	runCmd(t, "issue", "draft", "delete", draft.ID, "--force")

	_, err := c.GetDraft(draft.ID)
	assert.Error(t, err, "draft should no longer exist after delete")
}

func TestCmd_IssueDraftCreate_RequiresProject(t *testing.T) {
	// --project is required; cobra validates before any API call.
	out, err := runCmdFail(t, "issue", "draft", "create", "--summary", "no project")
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(out), "project")
}

func TestCmd_IssueDraftUpdate_UnknownField(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	out := runCmd(t, "issue", "draft", "create",
		"--project", proj.ShortName,
		"--format", "json",
	)
	var draft api.IssueDraft
	require.NoError(t, json.Unmarshal([]byte(out), &draft))

	combined, err := runCmdFail(t, "issue", "draft", "update", draft.ID,
		"--custom-fields", `{"NonExistentField999": "Bug"}`,
	)
	require.Error(t, err)
	assert.Contains(t, combined, "not found")
}
