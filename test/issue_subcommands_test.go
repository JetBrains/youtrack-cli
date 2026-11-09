package api_test

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

// issueStateGetter is a minimal interface for raw API field verification.
type issueStateGetter interface {
	Get(path string, result any) error
}

// assertIssueAssignee checks the issue's Assignee custom field.
// Uses json.RawMessage so mixed-type custom field values (object vs array) don't cause unmarshal errors.
func assertIssueAssignee(t *testing.T, c issueStateGetter, issueID, wantLogin string) {
	t.Helper()
	var result struct {
		CustomFields []struct {
			Name  string          `json:"name"`
			Value json.RawMessage `json:"value"`
		} `json:"customFields"`
	}
	err := c.Get(fmt.Sprintf("/api/issues/%s?fields=customFields(name,value(login))", url.PathEscape(issueID)), &result)
	require.NoError(t, err)
	for _, f := range result.CustomFields {
		if f.Name != "Assignee" {
			continue
		}
		if wantLogin == "" {
			isNull := len(f.Value) == 0 || string(f.Value) == "null"
			assert.True(t, isNull, "expected no assignee on issue %s, got: %s", issueID, string(f.Value))
		} else {
			require.True(t, len(f.Value) > 0 && string(f.Value) != "null",
				"expected assignee on issue %s", issueID)
			var user struct {
				Login string `json:"login"`
			}
			require.NoError(t, json.Unmarshal(f.Value, &user), "parsing Assignee value")
			assert.Equal(t, wantLogin, user.Login)
		}
		return
	}
	if wantLogin != "" {
		t.Errorf("Assignee custom field not found on issue %s", issueID)
	}
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

func assertIssueWatched(t *testing.T, c issueStateGetter, issueID string, want bool) {
	t.Helper()
	var result struct {
		Watchers struct {
			HasStar bool `json:"hasStar"`
		} `json:"watchers"`
	}
	err := c.Get(fmt.Sprintf("/api/issues/%s?fields=watchers(hasStar)", url.PathEscape(issueID)), &result)
	require.NoError(t, err)
	assert.Equal(t, want, result.Watchers.HasStar)
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
		assertIssueWatched(t, c, issue.IDReadable, true)
	})

	t.Run("unwatch", func(t *testing.T) {
		out := runCmd(t, "issue", "unwatch", issue.IDReadable)
		assert.Contains(t, out, "unwatched "+issue.IDReadable)
		// The reporter watches their own issue from creation, so this is the step that proves the request takes effect.
		assertIssueWatched(t, c, issue.IDReadable, false)
	})

	t.Run("watch again", func(t *testing.T) {
		runCmd(t, "issue", "watch", issue.IDReadable)
		assertIssueWatched(t, c, issue.IDReadable, true)
	})
}

func TestCmd_IssueVoteUnvote(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	// YouTrack blocks voting on issues you reported yourself, so we create a
	// temporary reporter user and create the issue under their name.
	me, err := c.GetCurrentUser()
	require.NoError(t, err)

	reporterLogin := fmt.Sprintf("vote-reporter-%d", time.Now().UnixMilli()%100000)
	reporter, err := c.AddUser(&api.AddUserRequest{
		Login:    reporterLogin,
		FullName: "Vote Test Reporter",
		Email:    reporterLogin + "@example.com",
		Password: "Test1234!",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteUser(reporter.Login, me.Login) })

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary:  "vote test issue",
		Project:  &api.Project{ID: proj.ID},
		Reporter: &api.User{ID: reporter.ID},
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
	out := runCmd(t, "issue", "command", "suggest", issue.IDReadable, "State ", "--format", "json")
	assert.Contains(t, out, "commands")
	preview := runJSON[map[string]any](t, "issue", "command", "suggest", issue.IDReadable, "assignee non.existent", "--json")
	assert.Equal(t, "assignee {non.existent}", preview["query"], "suggest must preview the same normalized query as apply")
}

func TestCmd_IssueCreateDescriptionFile(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	content := "description from file: unique content 42"
	f := filepath.Join(t.TempDir(), "desc.md")
	require.NoError(t, os.WriteFile(f, []byte(content), 0o600))

	out := runCmd(t, "issue", "create",
		"--project", proj.ShortName,
		"--summary", "desc-file test",
		"--description-file", f,
		"--format", "json",
	)
	assert.Contains(t, out, "desc-file test")

	// Parse the created issue ID directly from the JSON output (avoids search-index lag).
	var created api.Issue
	require.NoError(t, json.Unmarshal([]byte(out), &created))
	require.NotEmpty(t, created.IDReadable)
	t.Cleanup(func() { _ = c.DeleteIssue(created.IDReadable) })

	var result struct {
		Description string `json:"description"`
	}
	err := c.Get(fmt.Sprintf("/api/issues/%s?fields=description", url.PathEscape(created.IDReadable)), &result)
	require.NoError(t, err)
	assert.Equal(t, content, result.Description)
}

func TestCmd_IssueCreateParent(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	parent, err := c.AddIssue(api.AddIssueRequest{
		Summary: "parent issue",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(parent.IDReadable) })

	out := runCmd(t, "issue", "create",
		"--project", proj.ShortName,
		"--summary", "child issue",
		"--parent", parent.IDReadable,
		"--format", "json",
	)
	assert.Contains(t, out, "child issue")

	// Parse child ID directly from JSON output (avoids search-index lag).
	var child api.Issue
	require.NoError(t, json.Unmarshal([]byte(out), &child))
	require.NotEmpty(t, child.IDReadable)
	childID := child.IDReadable
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
	err = c.Get(fmt.Sprintf("/api/issues/%s/links?fields=linkType(targetToSource),issues(idReadable)&$top=500",
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

func TestCmd_IssueViewComments(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "comments flag test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	for _, text := range []string{"first comment", "second comment", "third comment"} {
		_, err = c.AddComment(issue.IDReadable, api.AddCommentRequest{Text: text})
		require.NoError(t, err)
	}

	result := runJSON[struct {
		IDReadable string        `json:"idReadable"`
		Comments   []api.Comment `json:"comments"`
	}](t, "issue", "view", issue.IDReadable, "--comments", "5", "--json")
	require.Equal(t, issue.IDReadable, result.IDReadable)
	require.Len(t, result.Comments, 3)
	require.Equal(t, "first comment", result.Comments[0].Text)

	// --comments N must return the N most recent comments, not the N oldest.
	result = runJSON[struct {
		IDReadable string        `json:"idReadable"`
		Comments   []api.Comment `json:"comments"`
	}](t, "issue", "view", issue.IDReadable, "--comments", "2", "--json")
	require.Len(t, result.Comments, 2)
	require.Equal(t, "second comment", result.Comments[0].Text)
	require.Equal(t, "third comment", result.Comments[1].Text)
}

func TestCmd_IssueViewPlain(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary:     "plain view test issue",
		Description: "full description text here",
		Project:     &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	out := runCmd(t, "issue", "view", issue.IDReadable, "--plain")
	assert.Contains(t, out, issue.IDReadable)
	assert.Contains(t, out, "plain view test issue")
	assert.Contains(t, out, "full description text here")
	assert.Contains(t, out, "Reporter:")
	assert.Contains(t, out, "Project:")
	assert.Contains(t, out, proj.ShortName)
}

func TestCmd_IssueViewNoDescription(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary:     "no description view test",
		Description: "description that should stay hidden",
		Project:     &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	result := runJSON[map[string]any](t, "issue", "view", issue.IDReadable, "--no-description", "--json")
	assert.Equal(t, issue.IDReadable, result["idReadable"])
	assert.Equal(t, "no description view test", result["summary"])
	assert.NotContains(t, result, "description")

	out := runCmd(t, "issue", "view", issue.IDReadable, "--no-description", "--plain")
	assert.Contains(t, out, "no description view test")
	assert.Contains(t, out, "Reporter:")
	assert.NotContains(t, out, "description that should stay hidden")
}

func TestCmd_IssueViewOnlyFields(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary:     "only fields view test",
		Description: "description outside the projection",
		Project:     &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })
	_, err = c.AddComment(issue.IDReadable, api.AddCommentRequest{Text: "only fields comment"})
	require.NoError(t, err)

	added := runJSON[map[string]any](t, "issue", "view", issue.IDReadable, "--fields", "idReadable,votes", "--json")
	assert.Contains(t, added, "description", "--fields alone must keep the default projection")
	assert.Contains(t, added, "votes")

	only := runJSON[map[string]any](t, "issue", "view", issue.IDReadable, "--only-fields", "--fields", "idReadable,summary,reporter", "--json")
	delete(only, "$type")
	assert.ElementsMatch(t, []string{"idReadable", "summary", "reporter"}, slices.Collect(maps.Keys(only)))
	reporter, ok := only["reporter"].(map[string]any)
	require.True(t, ok, "reporter must be an object: %v", only["reporter"])
	assert.NotEmpty(t, reporter["login"], "bare entity fields must still expand to their sub-fields")

	withComments := runJSON[struct {
		IDReadable  string        `json:"idReadable"`
		Description *string       `json:"description"`
		Comments    []api.Comment `json:"comments"`
	}](t, "issue", "view", issue.IDReadable, "--only-fields", "--fields", "idReadable", "--comments", "1", "--json")
	assert.Equal(t, issue.IDReadable, withComments.IDReadable)
	assert.Nil(t, withComments.Description)
	require.Len(t, withComments.Comments, 1)
	assert.Equal(t, "only fields comment", withComments.Comments[0].Text)

	out := runCmd(t, "issue", "view", issue.IDReadable, "--only-fields", "--fields", "idReadable,summary", "--plain")
	assert.Equal(t, issue.IDReadable+"\tonly fields view test", strings.TrimSpace(out))

	for _, format := range []string{"plain", "table"} {
		out = runCmd(t, "issue", "view", issue.IDReadable, "--only-fields", "--fields", "idReadable", "--comments", "1", "--format", format)
		assert.Contains(t, out, issue.IDReadable)
		assert.Contains(t, out, "only fields comment", "--fields projects the issue, not its comments (%s)", format)
	}

	errOut := runCmdExpectError(t, "issue", "view", issue.IDReadable, "--only-fields", "--json")
	assert.Contains(t, errOut, "--only-fields requires --fields")
}
