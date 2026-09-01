package api_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

func TestCmd_GroupList(t *testing.T) {
	c := newTestClient(t)
	groupName := fmt.Sprintf("testgroup-cmd-%d", time.Now().UnixMilli()%100000)
	group, err := c.AddGroup(groupName)
	require.NoError(t, err)
	successor := groupDeleteSuccessor(t, c, group.ID)
	t.Cleanup(func() { _ = c.DeleteGroup(group.ID, successor.ID) })

	out := runCmd(t, "group", "list", "--format", "json")
	assert.Contains(t, out, group.Name)
}

func TestCmd_GroupView(t *testing.T) {
	c := newTestClient(t)
	groupName := fmt.Sprintf("testgroup-get-%d", time.Now().UnixMilli()%100000)
	group, err := c.AddGroup(groupName)
	require.NoError(t, err)
	successor := groupDeleteSuccessor(t, c, group.ID)
	t.Cleanup(func() { _ = c.DeleteGroup(group.ID, successor.ID) })

	out := runCmd(t, "group", "view", group.Name, "--format", "json")
	assert.Contains(t, out, group.Name)
}

func TestCmd_GroupDelete(t *testing.T) {
	c := newTestClient(t)
	groupName := fmt.Sprintf("testgroup-del-%d", time.Now().UnixMilli()%100000)
	group, err := c.AddGroup(groupName)
	require.NoError(t, err)
	successor := groupDeleteSuccessor(t, c, group.ID)

	out := runCmd(t, "group", "delete", group.ID, "--successor", successor.ID, "--force")
	assert.Contains(t, out, "deleted group "+group.ID)

	_, err = c.GetGroup(group.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCmd_GroupDeleteRequiresSuccessor(t *testing.T) {
	// Security regression: rejecting the command before client creation prevents
	// an omitted successor from being replaced with a broad default group.
	out := runCmdExpectError(t, "group", "delete", "sensitive-group", "--force")
	assert.Contains(t, out, "required flag(s)")
	assert.Contains(t, out, "successor")
}

func TestCmd_GroupCreate(t *testing.T) {
	c := newTestClient(t)
	name := fmt.Sprintf("e2e-grp-add-%d", time.Now().UnixMilli()%100000)

	out := runCmd(t, "group", "create", "--name", name, "--format", "json")
	assert.Contains(t, out, name)

	groups, err := c.ListGroups("")
	require.NoError(t, err)
	for _, g := range groups {
		if g.Name == name {
			g := g
			successor := groupDeleteSuccessor(t, c, g.ID)
			t.Cleanup(func() { _ = c.DeleteGroup(g.ID, successor.ID) })
		}
	}
}

func TestCmd_GroupMemberList(t *testing.T) {
	c := newTestClient(t)
	name := fmt.Sprintf("e2e-grp-mbr-%d", time.Now().UnixMilli()%100000)
	group, err := c.AddGroup(name)
	require.NoError(t, err)
	successor := groupDeleteSuccessor(t, c, group.ID)
	t.Cleanup(func() { _ = c.DeleteGroup(group.ID, successor.ID) })

	// Some YouTrack versions expose members as a Hub-only endpoint; skip gracefully if unavailable.
	members, err := c.ListGroupMembers(group.ID)
	if err != nil {
		t.Skipf("group member list not supported on this instance: %v", err)
	}
	_ = members

	_ = runJSON[[]api.User](t, "group", "member", "list", group.ID, "--json")
}

func TestCmd_GroupMemberAddAndRemove(t *testing.T) {
	c := newTestClient(t)
	groupName := fmt.Sprintf("e2e-grp-members-%d", time.Now().UnixMilli()%100000)
	group, err := c.AddGroup(groupName)
	require.NoError(t, err)
	successorGroup := groupDeleteSuccessor(t, c, group.ID)
	t.Cleanup(func() { _ = c.DeleteGroup(group.ID, successorGroup.ID) })

	successorUser, err := c.GetCurrentUser()
	require.NoError(t, err)
	login := fmt.Sprintf("groupmember%d", time.Now().UnixMilli()%100000)
	user, err := c.AddUser(&api.AddUserRequest{
		Login: login, FullName: "Group Member", Email: login + "@example.com", Password: "Test1234!",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteUser(login, successorUser.Login) })

	addOut := runCmd(t, "group", "member", "add", group.ID, user.ID)
	assert.Contains(t, addOut, "added user "+user.ID)
	members := runJSON[[]api.User](t, "group", "member", "list", group.ID, "--json")
	require.True(t, containsUser(members, user.ID))

	removeOut := runCmd(t, "group", "member", "remove", group.ID, user.ID, "--force")
	assert.Contains(t, removeOut, "removed user "+user.ID)
	members = runJSON[[]api.User](t, "group", "member", "list", group.ID, "--json")
	require.False(t, containsUser(members, user.ID))
}

func containsUser(users []api.User, id string) bool {
	for _, user := range users {
		if user.ID == id {
			return true
		}
	}
	return false
}

func TestCmd_GroupRoleScopes(t *testing.T) {
	c := newTestClient(t)

	group, err := c.AddGroup(fmt.Sprintf("e2e-grp-scope-%d", time.Now().UnixMilli()%100000))
	require.NoError(t, err)
	successor := groupDeleteSuccessor(t, c, group.ID)
	t.Cleanup(func() { _ = c.DeleteGroup(group.ID, successor.ID) })

	project := createTempProject(t, c)

	roles, err := c.ListRoles()
	require.NoError(t, err)
	roleID := make(map[string]string, len(roles))
	for _, r := range roles {
		roleID[r.Name] = r.ID
	}
	require.Contains(t, roleID, "System Admin")
	require.Contains(t, roleID, "Project Admin")

	// 'group role add' and 'role grant --group' are the same code path; cover both.
	runCmd(t, "role", "grant", "--group", group.ID, "--role", roleID["System Admin"])
	out := runCmd(t, "group", "role", "add", group.ID, "--role", roleID["Project Admin"], "--project", project.ID)
	assert.Contains(t, out, "role assigned to group")

	assignments := runJSON[[]api.AssignedRole](t, "group", "role", "list", group.ID, "--json")
	for _, a := range assignments {
		id := a.ID
		t.Cleanup(func() { _ = c.RevokeRole(id) })
	}
	require.Len(t, assignments, 2, "expected exactly the two granted roles: %+v", assignments)

	scopeByRole := make(map[string]string, len(assignments))
	for _, a := range assignments {
		require.NotNil(t, a.Role, "assignment without a role: %+v", a)
		require.NotNil(t, a.Holder)
		require.NotNil(t, a.Scope, "assignment without a scope: %+v", a)
		assert.Equal(t, group.ID, a.Holder.ID, "list must filter server-side to the requested holder")
		scopeByRole[a.Role.Name] = a.Scope.Type
		if a.Role.Name == "Project Admin" {
			require.NotNil(t, a.Scope.Project)
			assert.Equal(t, project.ID, a.Scope.Project.ID)
		}
	}
	assert.Equal(t, map[string]string{
		"System Admin":  "GlobalScope",
		"Project Admin": "ProjectScope",
	}, scopeByRole)
}

func groupDeleteSuccessor(t *testing.T, c interface {
	ListGroups(query string) ([]api.Group, error)
}, excludeID string) api.Group {
	t.Helper()

	groups, err := c.ListGroups("")
	require.NoError(t, err)
	for _, group := range groups {
		if group.ID != excludeID {
			return group
		}
	}
	t.Fatal("no successor group available")
	return api.Group{}
}
