package api_test

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmd_UserCurrent(t *testing.T) {
	out := runCmd(t, "user", "current", "--format", "json")
	assert.Contains(t, out, "login")
}

func TestCmd_UserCurrentTrailingSlashBaseURL(t *testing.T) {
	requireE2EEnv(t)

	baseURL := strings.TrimRight(os.Getenv("YOUTRACK_BASE_URL"), "/")
	out := runCmd(t,
		"--base-url", baseURL+"/",
		"--token", os.Getenv("YOUTRACK_TOKEN"),
		"user", "current",
		"--format", "json",
	)

	assert.Contains(t, out, "login")
}

func TestCmd_UserList(t *testing.T) {
	out := runCmd(t, "user", "list", "--format", "json")
	assert.Contains(t, out, "login")
}

func TestCmd_UserCRUD(t *testing.T) {
	c := newTestClient(t)
	successor, err := c.GetCurrentUser()
	require.NoError(t, err)

	login := fmt.Sprintf("cmduser%d", time.Now().UnixMilli()%100000)
	t.Cleanup(func() { _ = c.DeleteUser(login, successor.Login) })

	out := runCmd(t,
		"user", "create",
		"--login", login,
		"--fullName", "Cmd Test User",
		"--email", login+"@example.com",
		"--password", "Test1234!",
		"--format", "json",
	)
	assert.Contains(t, out, login)

	out = runCmd(t, "user", "view", login, "--format", "json")
	assert.Contains(t, out, login)

	out = runCmd(t, "user", "list", "--query", login, "--format", "json")
	assert.Contains(t, out, login)

	// Ban/unban/delete require Hub registration (ringId must be set).
	// Hub sync should complete on any properly configured YouTrack instance.
	// If it doesn't, that's a bug — fail loudly so it gets investigated.
	newUser, err := c.GetUser(login)
	require.NoError(t, err)
	hubReady, err := c.WaitForHubSync(newUser.ID)
	require.NoError(t, err)
	require.True(t, hubReady,
		"Hub sync failed: user %s (id=%s) has ringId=null after creation — Hub is not registering new users on this instance",
		login, newUser.ID,
	)

	out = runCmd(t, "user", "ban", login)
	assert.Contains(t, out, "banned user "+login)
	assertUserBanState(t, c, login, true)

	out = runCmd(t, "user", "unban", login)
	assert.Contains(t, out, "unbanned user "+login)
	assertUserBanState(t, c, login, false)

	out = runCmd(t, "user", "delete", login, "--successor", successor.Login, "--force")
	assert.Contains(t, out, "deleted user "+login)

	users, err := c.ListUsers(login, 100)
	require.NoError(t, err)
	for _, user := range users {
		assert.NotEqual(t, login, user.Login)
	}
}

func TestCmd_UserLocaleSet(t *testing.T) {
	c := newTestClient(t)
	me, err := c.GetCurrentUser()
	require.NoError(t, err)
	// Should not error — HTTP 500 means the body key is wrong
	runCmd(t, "user", "locale", "set", me.Login, "en_US")
}

func TestCmd_UserDeleteRequiresSuccessor(t *testing.T) {
	// F-09 regression: --successor must be required; CLI must reject before any HTTP call
	out := runCmdExpectError(t, "user", "delete", "nonexistent-user")
	_ = out
}

func assertUserBanState(t *testing.T, c interface {
	Get(path string, result any) error
}, login string, want bool) {
	t.Helper()

	var result struct {
		Banned bool `json:"banned"`
	}
	err := c.Get("/api/users/"+url.PathEscape(login)+"?fields=banned", &result)
	require.NoError(t, err)
	assert.Equal(t, want, result.Banned)
}

func TestCmd_UserTimezone(t *testing.T) {
	c := newTestClient(t)
	me, err := c.GetCurrentUser()
	require.NoError(t, err)

	orig := runCmd(t, "user", "timezone", "view", me.Login)
	orig = strings.TrimSpace(orig)
	t.Cleanup(func() {
		if orig != "" {
			_ = c.SetUserTimezone(me.Login, orig)
		}
	})

	// set returns no output; verify get returns a non-empty string after the change
	runCmd(t, "user", "timezone", "set", me.Login, "Europe/Berlin")
	out := runCmd(t, "user", "timezone", "view", me.Login)
	assert.NotEmpty(t, strings.TrimSpace(out))
}

func TestCmd_UserDrafts(t *testing.T) {
	c := newTestClient(t)
	me, err := c.GetCurrentUser()
	require.NoError(t, err)

	proj := createTempProject(t, c)
	draft, err := c.CreateDraft(proj.ID, "user drafts fixture", "")
	require.NoError(t, err)

	// Listing by login, "me", or no argument at all must resolve to the same drafts.
	out := runCmd(t, "user", "drafts", "issues", me.Login, "--format", "json")
	assert.Contains(t, out, draft.ID)
	out = runCmd(t, "user", "drafts", "issues", "me", "--format", "json")
	assert.Contains(t, out, draft.ID)
	out = runCmd(t, "user", "drafts", "issues", "--format", "json")
	assert.Contains(t, out, draft.ID)

	// Article drafts are private too: no argument defaults to the current user (a JSON array),
	// and another user's drafts are inaccessible.
	out = runCmd(t, "user", "drafts", "articles", "--format", "json")
	assert.Contains(t, out, "[")
	runCmdExpectError(t, "user", "drafts", "articles", "no-such-user-e2e")

	// An unknown login must error out, not silently target the current user.
	runCmdExpectError(t, "user", "drafts", "list", "no-such-user-e2e")
	runCmdExpectError(t, "user", "drafts", "delete", "no-such-user-e2e")
	drafts, err := c.GetUserIssueDrafts("me")
	require.NoError(t, err)
	assert.NotEmpty(t, drafts, "delete for an unknown user must not wipe the current user's drafts")

	// Delete by login, then assert the fixture is gone (not that the list is empty, since the account may have other drafts).
	runCmd(t, "user", "drafts", "delete", me.Login)
	drafts, err = c.GetUserIssueDrafts("me")
	require.NoError(t, err)
	for _, d := range drafts {
		assert.NotEqual(t, draft.ID, d.ID, "delete by login must remove the fixture draft")
	}
}

func TestCmd_UserProfileView(t *testing.T) {
	c := newTestClient(t)
	me, err := c.GetCurrentUser()
	require.NoError(t, err)

	out := runCmd(t, "user", "profile", "view", me.Login, "--format", "json")
	// profile returns GeneralUserProfile object, not user login
	assert.Contains(t, out, "GeneralUserProfile")
}

func TestCmd_UserTagList(t *testing.T) {
	c := newTestClient(t)
	me, err := c.GetCurrentUser()
	require.NoError(t, err)

	out := runCmd(t, "user", "tag", "list", me.ID, "--format", "json")
	assert.Contains(t, out, "[")
}
