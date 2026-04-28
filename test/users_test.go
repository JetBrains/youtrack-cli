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
		"user", "add",
		"--login", login,
		"--fullName", "Cmd Test User",
		"--email", login+"@example.com",
		"--password", "Test1234!",
		"--format", "json",
	)
	assert.Contains(t, out, login)

	out = runCmd(t, "user", "get", login, "--format", "json")
	assert.Contains(t, out, login)

	out = runCmd(t, "user", "list", "--query", login, "--format", "json")
	assert.Contains(t, out, login)

	out = runCmd(t, "user", "ban", login)
	assert.Contains(t, out, "banned user "+login)
	assertUserBanState(t, c, login, true)

	out = runCmd(t, "user", "unban", login)
	assert.Contains(t, out, "unbanned user "+login)
	assertUserBanState(t, c, login, false)

	out = runCmd(t, "user", "delete", login, "--successor", successor.Login, "--force")
	assert.Contains(t, out, "deleted user "+login)

	users, err := c.ListUsers(login)
	require.NoError(t, err)
	for _, user := range users {
		assert.NotEqual(t, login, user.Login)
	}
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
