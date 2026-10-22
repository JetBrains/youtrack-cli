package api_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

func TestCmd_UserListsBeyondTen(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	me, err := c.GetCurrentUser()
	require.NoError(t, err)

	prefix := "scale-" + proj.ShortName
	group, err := c.AddGroup(prefix)
	require.NoError(t, err)
	successor := groupDeleteSuccessor(t, c, group.ID)
	t.Cleanup(func() { _ = c.DeleteGroup(group.ID, successor.ID) })

	var users []api.User
	for i := range 11 {
		login := fmt.Sprintf("%s-%02d", prefix, i)
		user, err := c.AddUser(&api.AddUserRequest{
			Login: login, FullName: login, Email: login + "@example.com", Password: "Test1234!",
		})
		require.NoError(t, err)
		t.Cleanup(func() { _ = c.DeleteUser(user.ID, me.ID) })
		require.NoError(t, c.AddGroupMember(group.ID, user.ID))
		_, err = c.AddUserToProjectTeam(proj.ShortName, user.ID)
		require.NoError(t, err)
		users = append(users, *user)
	}

	cases := []struct {
		name       string
		args       []string
		exactCount bool
	}{
		{"user list", []string{"user", "list", "--query", prefix, "--limit", "11", "--json"}, true},
		{"group members", []string{"group", "member", "list", group.ID, "--json"}, true},
		{"project team", []string{"project", "team", "list-users", proj.ShortName, "--json"}, false},
		{"raw pagination", []string{"raw", "/api/users", "--fields", "id,login", "--paginate", "--page-size", "5"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			listed := runJSON[[]api.User](t, tc.args...)
			if tc.exactCount {
				assert.Len(t, listed, len(users))
			}
			seen := make(map[string]bool)
			for _, user := range listed {
				assert.False(t, seen[user.ID], "duplicate user %s", user.ID)
				seen[user.ID] = true
			}
			for _, user := range users {
				assert.True(t, seen[user.ID], "missing user %s", user.Login)
			}
		})
	}

	t.Run("user view", func(t *testing.T) {
		want := users[len(users)-1]
		got := runJSON[api.User](t, "user", "view", want.Login, "--json")
		assert.Equal(t, want.ID, got.ID)
		assert.Equal(t, want.Login, got.Login)
	})
}
