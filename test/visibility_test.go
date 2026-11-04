package api_test

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

// Names resolve to IDs, each call replaces both lists, and --public lifts the restriction.
func TestCmd_IssueVisibilityByName(t *testing.T) {
	c := newTestClient(t)
	f := newVisibilityFixture(t, c)
	path := "/api/issues/" + url.PathEscape(f.issue)

	out := runCmd(t, "issue", "visibility", f.issue, "--groups", f.group.Name, "--users", f.user.Login)
	assert.Contains(t, out, "updated visibility for "+f.issue)
	groups, users := permittedIDs(readVisibility(t, c, path))
	assert.Equal(t, []string{f.group.ID}, groups)
	assert.Equal(t, []string{f.user.ID}, users)

	runCmd(t, "issue", "visibility", f.issue, "--users", f.user.Login)
	groups, users = permittedIDs(readVisibility(t, c, path))
	assert.Empty(t, groups, "a list left out must be cleared, not kept")
	assert.Equal(t, []string{f.user.ID}, users)

	runCmd(t, "issue", "visibility", f.issue, "--groups", f.group.ID)
	groups, users = permittedIDs(readVisibility(t, c, path))
	assert.Equal(t, []string{f.group.ID}, groups, "IDs are still accepted")
	assert.Empty(t, users)

	out = runCmd(t, "issue", "visibility", f.issue, "--public")
	assert.Contains(t, out, "public")
	assert.Equal(t, "UnlimitedVisibility", readVisibility(t, c, path).Type)
}

// Without an audience the command must fail instead of sending empty lists, which YouTrack treats as public.
func TestCmd_IssueVisibilityRequiresAudience(t *testing.T) {
	c := newTestClient(t)
	f := newVisibilityFixture(t, c)
	path := "/api/issues/" + url.PathEscape(f.issue)
	runCmd(t, "issue", "visibility", f.issue, "--users", f.user.Login)

	for _, args := range [][]string{
		{"issue", "visibility", f.issue},
		{"issue", "visibility", f.issue, "--users", ""},
		{"issue", "visibility", f.issue, "--public", "--users", f.user.Login},
		{"issue", "visibility", f.issue, "--users", "no.such.user.visibility"},
	} {
		out, err := runCmdFail(t, args...)
		assert.Error(t, err, "%v should fail: %s", args, out)
	}
	_, users := permittedIDs(readVisibility(t, c, path))
	assert.Equal(t, []string{f.user.ID}, users, "a rejected call must not change visibility")
}

func TestCmd_IssueCommentCreateWithVisibility(t *testing.T) {
	c := newTestClient(t)
	f := newVisibilityFixture(t, c)

	comment := runJSON[api.Comment](t, "issue", "comment", "create", f.issue, "--text", "limited from the start",
		"--visibility", f.group.Name, "--visibility", "user:"+f.user.Login, "--json")
	require.NotEmpty(t, comment.ID)
	groups, users := permittedIDs(readVisibility(t, c, fmt.Sprintf("/api/issues/%s/comments/%s", url.PathEscape(f.issue), comment.ID)))
	assert.Equal(t, []string{f.group.ID}, groups)
	assert.Equal(t, []string{f.user.ID}, users)

	public := runJSON[api.Comment](t, "issue", "comment", "create", f.issue, "--text", "public", "--public", "--json")
	assert.Equal(t, "UnlimitedVisibility", readVisibility(t, c, fmt.Sprintf("/api/issues/%s/comments/%s", url.PathEscape(f.issue), public.ID)).Type)

	out, err := runCmdFail(t, "issue", "comment", "create", f.issue, "--text", "never posted", "--visibility", "no-such-group-or-user")
	require.Error(t, err, out)
	comments, err := c.ListComments(f.issue)
	require.NoError(t, err)
	for _, cm := range comments {
		assert.NotEqual(t, "never posted", cm.Text, "an unresolved audience must not post a public comment")
	}
}

func TestCmd_IssueCommentCreateVisibilityAmbiguousName(t *testing.T) {
	c := newTestClient(t)
	f := newVisibilityFixture(t, c)
	twin, err := c.AddGroup(f.user.Login)
	require.NoError(t, err)
	successor := groupDeleteSuccessor(t, c, twin.ID)
	t.Cleanup(func() { _ = c.DeleteGroup(twin.ID, successor.ID) })

	out, err := runCmdFail(t, "issue", "comment", "create", f.issue, "--text", "ambiguous", "--visibility", f.user.Login)
	require.Error(t, err)
	assert.Contains(t, out, "both a group and a user")
}

func TestCmd_IssueCommentVisibilityByName(t *testing.T) {
	c := newTestClient(t)
	f := newVisibilityFixture(t, c)
	comment, err := c.AddComment(f.issue, api.AddCommentRequest{Text: "visibility me"})
	require.NoError(t, err)
	path := fmt.Sprintf("/api/issues/%s/comments/%s", url.PathEscape(f.issue), comment.ID)

	out := runCmd(t, "issue", "comment", "visibility", f.issue, comment.ID, "--groups", f.group.Name)
	assert.Contains(t, out, "updated visibility for comment "+comment.ID)
	groups, users := permittedIDs(readVisibility(t, c, path))
	assert.Equal(t, []string{f.group.ID}, groups)
	assert.Empty(t, users)

	runCmd(t, "issue", "comment", "visibility", f.issue, comment.ID, "--public")
	assert.Equal(t, "UnlimitedVisibility", readVisibility(t, c, path).Type)
}

func TestCmd_IssueAttachmentVisibility(t *testing.T) {
	c := newTestClient(t)
	f := newVisibilityFixture(t, c)
	file := filepath.Join(t.TempDir(), "secret.txt")
	require.NoError(t, os.WriteFile(file, []byte("secret"), 0o600))
	attachment, err := c.AddIssueAttachment(f.issue, file)
	require.NoError(t, err)
	path := fmt.Sprintf("/api/issues/%s/attachments/%s", url.PathEscape(f.issue), attachment.ID)

	out := runCmd(t, "issue", "attachment", "visibility", f.issue, attachment.ID, "--users", f.user.Login)
	assert.Contains(t, out, "updated visibility for attachment "+attachment.ID)
	groups, users := permittedIDs(readVisibility(t, c, path))
	assert.Empty(t, groups)
	assert.Equal(t, []string{f.user.ID}, users)

	listed := runJSON[[]api.IssueAttachment](t, "issue", "attachment", "list", f.issue, "--json")
	require.Len(t, listed, 1)
	require.NotNil(t, listed[0].Visibility)
	assert.Equal(t, "LimitedVisibility", listed[0].Visibility.Type)

	runCmd(t, "issue", "attachment", "visibility", f.issue, attachment.ID, "--public")
	assert.Equal(t, "UnlimitedVisibility", readVisibility(t, c, path).Type)
}

// A Developer can post restricted comments without permission to read group records.
func TestCmd_VisibilityAsNonAdmin(t *testing.T) {
	c := newTestClient(t)
	f := newVisibilityFixture(t, c)
	project := strings.Split(f.issue, "-")[0]
	_, err := c.AddUserToProjectTeam(project, f.user.ID)
	require.NoError(t, err)
	twin, err := c.AddGroup(f.user.Login)
	require.NoError(t, err)
	successor := groupDeleteSuccessor(t, c, twin.ID)
	t.Cleanup(func() { _ = c.DeleteGroup(twin.ID, successor.ID) })
	token := visibilityUserToken(t, c, f.user)
	reader := api.NewClient(os.Getenv("YOUTRACK_BASE_URL"), token)
	var group api.Group
	err = reader.Get("/api/groups/"+twin.ID+"?fields=id", &group)
	var apiErr *api.APIError
	require.ErrorAs(t, err, &apiErr)
	require.Contains(t, []int{http.StatusForbidden, http.StatusNotFound}, apiErr.StatusCode, "fixture must lack group read permission")
	comment := runJSON[api.Comment](t, "--token", token, "issue", "comment", "create", f.issue,
		"--text", "private developer comment", "--asUser", f.user.Login, "--visibility", f.user.Login, "--json")
	_, users := permittedIDs(readVisibility(t, c, fmt.Sprintf("/api/issues/%s/comments/%s", f.issue, comment.ID)))
	assert.Equal(t, []string{f.user.ID}, users)
	// A group-only name must still surface the permission error, rather than guessing.
	output := runCmdExpectError(t, "--token", token, "issue", "comment", "create", f.issue,
		"--text", "must not be posted", "--visibility", "group:"+twin.Name)
	assert.NotEmpty(t, output)
	// Explicit IDs must remain usable without group reads.
	runCmd(t, "--token", token, "issue", "comment", "visibility", f.issue, comment.ID, "--groups", twin.ID)
}

func TestCmd_IssueCreateWithVisibility(t *testing.T) {
	c := newTestClient(t)
	f := newVisibilityFixture(t, c)
	project := strings.Split(f.issue, "-")[0]
	_, err := c.AddUserToProjectTeam(project, f.user.ID)
	require.NoError(t, err)
	token := visibilityUserToken(t, c, f.user)
	parent := runJSON[api.Issue](t, "--token", token, "issue", "view", f.issue, "--json")
	require.Equal(t, f.issue, parent.IDReadable)
	issue := runJSON[api.Issue](t, "issue", "create", "--project", project, "--summary", "restricted at creation",
		"--visibility", "group:"+f.group.ID, "--json")
	t.Cleanup(func() { _ = c.DeleteIssue(issue.ID) })
	groups, _ := permittedIDs(readVisibility(t, c, "/api/issues/"+issue.ID))
	assert.Equal(t, []string{f.group.ID}, groups)
	runCmdExpectError(t, "--token", token, "issue", "view", issue.IDReadable, "--json")
	comment := runJSON[api.Comment](t, "issue", "comment", "create", f.issue, "--text", "restricted at creation",
		"--visibility", "group:"+f.group.ID, "--json")
	runCmdExpectError(t, "--token", token, "issue", "comment", "view", f.issue, comment.ID, "--json")
	before, err := c.ListIssues("project: "+project, 100)
	require.NoError(t, err)
	runCmdExpectError(t, "issue", "create", "--project", project, "--summary", "never created", "--visibility", "user:e2e.missing."+project)
	after, err := c.ListIssues("project: "+project, 100)
	require.NoError(t, err)
	assert.Len(t, after, len(before), "invalid audiences must not create an issue")
}
