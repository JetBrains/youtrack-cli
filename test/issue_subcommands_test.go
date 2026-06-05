package api_test

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

// issueStateGetter is a minimal interface for raw API field verification.
type issueStateGetter interface {
	Get(path string, result any) error
}

// assertIssueAssignee checks the issue's assignee login via a direct API call.
func assertIssueAssignee(t *testing.T, c issueStateGetter, issueID, wantLogin string) {
	t.Helper()
	var result struct {
		Assignee *struct {
			Login string `json:"login"`
		} `json:"assignee"`
	}
	err := c.Get(fmt.Sprintf("/api/issues/%s?fields=assignee(login)", url.PathEscape(issueID)), &result)
	require.NoError(t, err)
	if wantLogin == "" {
		assert.Nil(t, result.Assignee)
	} else {
		require.NotNil(t, result.Assignee)
		assert.Equal(t, wantLogin, result.Assignee.Login)
	}
}

// assertIssueWatching checks whether the current user is watching the issue.
func assertIssueWatching(t *testing.T, c issueStateGetter, issueID string, want bool) {
	t.Helper()
	var result struct {
		Watchers struct {
			HasWatchedIssue bool `json:"hasWatchedIssue"`
		} `json:"watchers"`
	}
	err := c.Get(fmt.Sprintf("/api/issues/%s?fields=watchers(hasWatchedIssue)", url.PathEscape(issueID)), &result)
	require.NoError(t, err)
	assert.Equal(t, want, result.Watchers.HasWatchedIssue)
}

// assertIssueVoted checks whether the current user has voted for the issue.
func assertIssueVoted(t *testing.T, c issueStateGetter, issueID string, want bool) {
	t.Helper()
	var result struct {
		Voters struct {
			HasVote bool `json:"hasVote"`
		} `json:"voters"`
	}
	err := c.Get(fmt.Sprintf("/api/issues/%s?fields=voters(hasVote)", url.PathEscape(issueID)), &result)
	require.NoError(t, err)
	assert.Equal(t, want, result.Voters.HasVote)
}

func TestCmd_IssueAssignUnassign(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	me, err := c.GetCurrentUser()
	require.NoError(t, err)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "assign test issue",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	t.Run("assign", func(t *testing.T) {
		out := runCmd(t, "issue", "assign", issue.IDReadable, me.Login)
		assert.Contains(t, out, "assigned "+issue.IDReadable+" to "+me.Login)
		assertIssueAssignee(t, c, issue.IDReadable, me.Login)
	})

	t.Run("unassign", func(t *testing.T) {
		out := runCmd(t, "issue", "unassign", issue.IDReadable)
		assert.Contains(t, out, "unassigned "+issue.IDReadable)
		assertIssueAssignee(t, c, issue.IDReadable, "")
	})
}

func TestCmd_IssueWatchUnwatch(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "watch test issue",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	t.Run("watch", func(t *testing.T) {
		out := runCmd(t, "issue", "watch", issue.IDReadable)
		assert.Contains(t, out, "watching "+issue.IDReadable)
		assertIssueWatching(t, c, issue.IDReadable, true)
	})

	t.Run("unwatch", func(t *testing.T) {
		out := runCmd(t, "issue", "unwatch", issue.IDReadable)
		assert.Contains(t, out, "unwatched "+issue.IDReadable)
		assertIssueWatching(t, c, issue.IDReadable, false)
	})
}

func TestCmd_IssueVoteUnvote(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	// Create the issue as a different user so we can vote on it.
	// (YouTrack does not allow voting on your own issue by default.)
	// We use a fresh project so the reporter is the current user; voting
	// is attempted on the issue itself. Some YouTrack configs allow this;
	// if the vote API returns an error the test will surface it clearly.
	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "vote test issue",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	t.Run("vote", func(t *testing.T) {
		out := runCmd(t, "issue", "vote", issue.IDReadable)
		assert.Contains(t, out, "voted for "+issue.IDReadable)
		assertIssueVoted(t, c, issue.IDReadable, true)
	})

	t.Run("unvote", func(t *testing.T) {
		out := runCmd(t, "issue", "unvote", issue.IDReadable)
		assert.Contains(t, out, "removed vote from "+issue.IDReadable)
		assertIssueVoted(t, c, issue.IDReadable, false)
	})
}

func TestCmd_IssueDraftCreate(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	out := runCmd(t, "issue", "draft", "create",
		"--project", proj.ShortName,
		"--summary", "draft summary",
		"--format", "json",
	)
	assert.Contains(t, out, "draft summary")
	// Drafts have opaque IDs that start with "2-" — verify the response has an id field.
	assert.Contains(t, out, `"id"`)
}

func TestCmd_IssueSuggest(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "suggest test issue",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	// Ask for suggestions for a partial "State " query — YouTrack always returns
	// a commands array (possibly empty) and the echoed query.
	out := runCmd(t, "issue", "suggest", issue.IDReadable, "State ", "--format", "json")
	assert.Contains(t, out, "commands")
}

func TestCmd_IssueAddDescriptionFile(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	content := "description from file: unique content 42"
	f := filepath.Join(t.TempDir(), "desc.md")
	require.NoError(t, os.WriteFile(f, []byte(content), 0o600))

	out := runCmd(t, "issue", "add",
		"--project", proj.ShortName,
		"--summary", "desc-file test",
		"--description-file", f,
		"--format", "json",
	)
	assert.Contains(t, out, "desc-file test")

	// Verify description was stored.
	var created struct {
		IDReadable  string `json:"idReadable"`
		Description string `json:"description"`
	}
	// Fetch the issue via the API client to read the description field.
	issues, err := c.ListIssues(fmt.Sprintf("project: %s summary: {desc-file test}", proj.ShortName), 1)
	require.NoError(t, err)
	require.NotEmpty(t, issues)
	t.Cleanup(func() { _ = c.DeleteIssue(issues[0].IDReadable) })
	_ = created

	var result struct {
		Description string `json:"description"`
	}
	err = c.Get(fmt.Sprintf("/api/issues/%s?fields=description", url.PathEscape(issues[0].IDReadable)), &result)
	require.NoError(t, err)
	assert.Equal(t, content, result.Description)
}

func TestCmd_IssueAddParent(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	parent, err := c.AddIssue(api.AddIssueRequest{
		Summary: "parent issue",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(parent.IDReadable) })

	out := runCmd(t, "issue", "add",
		"--project", proj.ShortName,
		"--summary", "child issue",
		"--parent", parent.IDReadable,
		"--format", "json",
	)
	assert.Contains(t, out, "child issue")

	// Extract child ID from output.
	issues, err := c.ListIssues(fmt.Sprintf("project: %s summary: {child issue}", proj.ShortName), 1)
	require.NoError(t, err)
	require.NotEmpty(t, issues)
	childID := issues[0].IDReadable
	t.Cleanup(func() { _ = c.DeleteIssue(childID) })

	// Verify parent link via raw API: links contain the parent issue.
	var links []struct {
		LinkType struct {
			TargetToSource string `json:"targetToSource"`
		} `json:"linkType"`
		Issues []struct {
			IDReadable string `json:"idReadable"`
		} `json:"issues"`
	}
	err = c.Get(fmt.Sprintf("/api/issues/%s/links?fields=linkType(targetToSource),issues(idReadable)&$top=-1",
		url.PathEscape(childID)), &links)
	require.NoError(t, err)

	var parentFound bool
	for _, link := range links {
		for _, linked := range link.Issues {
			if linked.IDReadable == parent.IDReadable {
				parentFound = true
			}
		}
	}
	assert.True(t, parentFound, "expected child %s to be linked to parent %s", childID, parent.IDReadable)
}

func TestCmd_IssueGetComments(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "comments flag test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	_, err = c.AddComment(issue.IDReadable, api.AddCommentRequest{Text: "hello from test"})
	require.NoError(t, err)

	out := runCmd(t, "issue", "get", issue.IDReadable, "--comments", "5", "--format", "json")
	// Output includes the issue JSON followed by the comments JSON array.
	assert.True(t, strings.Contains(out, "hello from test"), "expected comment text in output")
}
