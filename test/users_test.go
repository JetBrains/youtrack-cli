package api_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

func TestUser_Current(t *testing.T) {
	c := newTestClient(t)
	user, err := c.GetCurrentUser()
	require.NoError(t, err)
	assert.NotEmpty(t, user.Login)
}

func TestUser_ListAndGet(t *testing.T) {
	c := newTestClient(t)

	users, err := c.ListUsers("")
	require.NoError(t, err)
	require.NotEmpty(t, users)

	got, err := c.GetUser(users[0].Login)
	require.NoError(t, err)
	assert.Equal(t, users[0].Login, got.Login)
}

func TestUser_CRUD(t *testing.T) {
	c := newTestClient(t)

	login := fmt.Sprintf("testuser%d", time.Now().UnixMilli()%100000)
	user, err := c.AddUser(api.AddUserRequest{
		Login:    login,
		FullName: "Test User",
		Email:    login + "@example.com",
		Password: "Test1234!",
	})
	require.NoError(t, err)
	require.Equal(t, login, user.Login)
	t.Cleanup(func() { _ = c.DeleteUser(login) })

	got, err := c.GetUser(login)
	require.NoError(t, err)
	assert.Equal(t, login, got.Login)
}

func TestUser_Ban(t *testing.T) {
	c := newTestClient(t)

	login := fmt.Sprintf("banuser%d", time.Now().UnixMilli()%100000)
	_, err := c.AddUser(api.AddUserRequest{
		Login:    login,
		FullName: "Ban Test User",
		Email:    login + "@example.com",
		Password: "Test1234!",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteUser(login) })

	err = c.BanUser(login, true)
	require.NoError(t, err)

	err = c.BanUser(login, false)
	require.NoError(t, err)
}

func TestUser_Timezone(t *testing.T) {
	c := newTestClient(t)
	me, err := c.GetCurrentUser()
	require.NoError(t, err)

	err = c.SetUserTimezone(me.Login, "Europe/Berlin")
	require.NoError(t, err)
}

func TestUser_Locale(t *testing.T) {
	c := newTestClient(t)
	me, err := c.GetCurrentUser()
	require.NoError(t, err)

	err = c.SetUserLocale(me.Login, "en_US")
	if err != nil {
		t.Skipf("SetUserLocale returned error (known YouTrack instance limitation): %v", err)
	}
}

func TestCmd_UserCurrent(t *testing.T) {
	out := runCmd(t, "user", "current", "--format", "json")
	assert.Contains(t, out, "login")
}

func TestCmd_UserList(t *testing.T) {
	out := runCmd(t, "user", "list", "--format", "json")
	assert.Contains(t, out, "login")
}
