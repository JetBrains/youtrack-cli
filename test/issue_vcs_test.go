package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

// createGenericVCSProcessor gives a project a Generic VCS integration.
//
// The harness has no real repository to talk to, and the Generic integration is
// the one kind that needs none: it accepts any commit hash and builds the change
// from the request itself. Every other integration would call out to a hosting
// service to resolve the hash, so this is what makes the attach path testable.
func createGenericVCSProcessor(t *testing.T, c *api.Client, projectID string) string {
	t.Helper()

	body := map[string]any{
		"$type": "GenericVcsChangesProcessor",
		// The Generic integration requires a commit URL template carrying $hash$.
		"url":     "https://example.invalid/org/repo/commit/$hash$",
		"enabled": true,
		"project": map[string]any{"id": projectID},
	}
	var result struct {
		ID string `json:"id"`
	}
	err := c.Post(
		"/api/admin/projects/"+url.PathEscape(projectID)+"/vcsChangesProcessors?fields=id,enabled",
		body, &result,
	)
	require.NoError(t, err, "creating Generic VCS changes processor")
	require.NotEmpty(t, result.ID)
	return result.ID
}

// createVCSTestIssue makes an issue in a project that carries a Generic VCS integration.
func createVCSTestIssue(t *testing.T, c *api.Client, summary string) (*api.Issue, *api.Project) {
	t.Helper()

	proj := createTempProject(t, c)
	createGenericVCSProcessor(t, c, proj.ID)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: summary,
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })
	return issue, proj
}

// The detach branch that keeps the record is not covered here. With a Generic VCS
// integration YouTrack sets shouldBeFullyRemoved, so a detached change is always
// deleted (AbstractVcsItemsService.handleRemovedChanges), and the command takes the
// DELETE branch for a manual attachment anyway. Reaching the other branch needs an
// integration-attached change, which the harness cannot produce without a real
// repository push.
func TestCmd_IssueVCSAttachDetach(t *testing.T) {
	c := newTestClient(t)
	issue, _ := createVCSTestIssue(t, c, "vcs lifecycle test issue")

	const hash = "1a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d"

	t.Run("list is empty before any attach", func(t *testing.T) {
		changes := runJSON[[]api.VCSChange](t, "issue", "vcs", "list", issue.IDReadable, "--json")
		assert.Empty(t, changes)
	})

	var changeID string

	t.Run("attach links the commit", func(t *testing.T) {
		change := runJSON[api.VCSChange](t,
			"issue", "vcs", "attach", issue.IDReadable, hash,
			"--message", "fix the thing", "--json")

		assert.Equal(t, hash, change.Version)
		require.NotEmpty(t, change.ID)
		require.NotNil(t, change.State)
		assert.Equal(t, api.VCSChangeStateManuallyAttached, *change.State)
		// A Generic VCS integration takes the commit message from the request.
		assert.Equal(t, "fix the thing", change.Text)
		changeID = change.ID
	})

	t.Run("list shows the attached commit", func(t *testing.T) {
		changes := runJSON[[]api.VCSChange](t, "issue", "vcs", "list", issue.IDReadable, "--json")
		require.Len(t, changes, 1)
		assert.Equal(t, changeID, changes[0].ID)
		assert.Equal(t, hash, changes[0].Version)
	})

	// Table output cannot be asserted here: the CLI auto-enables --json when
	// stdout is not a TTY, so --plain is what exercises a non-JSON formatter.
	t.Run("list renders plain output", func(t *testing.T) {
		out := runCmd(t, "issue", "vcs", "list", issue.IDReadable, "--plain")
		fields := strings.Split(strings.TrimSpace(out), "\t")
		require.Len(t, fields, 4, "plain output: %q", out)
		assert.Equal(t, changeID, fields[0])
		assert.Equal(t, hash, fields[1])
		assert.Equal(t, "admin", fields[2])
		assert.Equal(t, "fix the thing", fields[3])
	})

	t.Run("view reads the change back", func(t *testing.T) {
		change := runJSON[api.VCSChange](t, "issue", "vcs", "view", issue.IDReadable, changeID, "--json")
		assert.Equal(t, changeID, change.ID)
		assert.Equal(t, hash, change.Version)
	})

	t.Run("attaching the same commit twice fails", func(t *testing.T) {
		stderr := runCmdExpectError(t, "issue", "vcs", "attach", issue.IDReadable, hash)
		assert.Contains(t, stderr, "already attached")
	})

	t.Run("detach removes a manually attached commit", func(t *testing.T) {
		out := runCmd(t, "issue", "vcs", "detach", issue.IDReadable, changeID)
		assert.Contains(t, out, "removed VCS change "+changeID)

		changes := runJSON[[]api.VCSChange](t, "issue", "vcs", "list", issue.IDReadable, "--json")
		assert.Empty(t, changes)
	})

	t.Run("detach reports an unknown change", func(t *testing.T) {
		stderr := runCmdExpectError(t, "issue", "vcs", "detach", issue.IDReadable, changeID)
		assert.Contains(t, stderr, "no VCS change matching")
	})

	t.Run("delete is an alias of detach", func(t *testing.T) {
		change := runJSON[api.VCSChange](t, "issue", "vcs", "attach", issue.IDReadable, hash, "--json")
		require.NotEmpty(t, change.ID)

		out := runCmd(t, "issue", "vcs", "delete", issue.IDReadable, change.ID)
		assert.Contains(t, out, "removed VCS change "+change.ID)
		assert.Empty(t, runJSON[[]api.VCSChange](t, "issue", "vcs", "list", issue.IDReadable, "--json"))
	})
}

func TestCmd_IssueVCSRefResolution(t *testing.T) {
	c := newTestClient(t)
	issue, _ := createVCSTestIssue(t, c, "vcs ref resolution test issue")

	const hash = "abcdef0123456789abcdef0123456789abcdef01"

	change := runJSON[api.VCSChange](t, "issue", "vcs", "attach", issue.IDReadable, hash, "--json")
	changeID := change.ID
	require.NotEmpty(t, changeID)

	t.Run("hash shorter than 12 characters does not resolve", func(t *testing.T) {
		stderr := runCmdExpectError(t, "issue", "vcs", "detach", issue.IDReadable, hash[:8])
		assert.Contains(t, stderr, "no VCS change matching")
	})

	t.Run("unknown ref reports no match", func(t *testing.T) {
		stderr := runCmdExpectError(t,
			"issue", "vcs", "detach", issue.IDReadable, "0000000000000000000000000000000000000000")
		assert.Contains(t, stderr, "no VCS change matching")
	})

	t.Run("uppercase hash resolves", func(t *testing.T) {
		out := runCmd(t, "issue", "vcs", "detach", issue.IDReadable, strings.ToUpper(hash))
		assert.Contains(t, out, "removed VCS change "+changeID)
	})

	t.Run("abbreviated hash of 12 characters resolves", func(t *testing.T) {
		fresh := runJSON[api.VCSChange](t, "issue", "vcs", "attach", issue.IDReadable, hash, "--json")
		require.NotEmpty(t, fresh.ID)

		out := runCmd(t, "issue", "vcs", "detach", issue.IDReadable, hash[:12])
		assert.Contains(t, out, "removed VCS change "+fresh.ID)
	})
}

func TestCmd_IssueVCSNoIntegration(t *testing.T) {
	c := newTestClient(t)

	// No VCS integration on this project, so YouTrack cannot resolve any hash.
	proj := createTempProject(t, c)
	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "vcs no-integration test issue",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	t.Run("list is empty", func(t *testing.T) {
		changes := runJSON[[]api.VCSChange](t, "issue", "vcs", "list", issue.IDReadable, "--json")
		assert.Empty(t, changes)
	})

	t.Run("attach fails for a commit no integration knows", func(t *testing.T) {
		stderr := runCmdExpectError(t,
			"issue", "vcs", "attach", issue.IDReadable, "1234567890abcdef1234567890abcdef12345678")
		assert.NotEmpty(t, stderr)
	})
}
