package api_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

func TestCmd_RoleList(t *testing.T) {
	out := runCmd(t, "role", "list", "--format", "json")
	assert.NotEmpty(t, out)
}

func TestCmd_RolePermissions(t *testing.T) {
	out := runCmd(t, "role", "permissions", "--format", "json")
	assert.NotEmpty(t, out)
}

func TestCmd_RoleCreateDelete(t *testing.T) {
	c := newTestClient(t)
	perms, err := c.GetPermissions()
	require.NoError(t, err)
	require.NotEmpty(t, perms)

	name := fmt.Sprintf("cmd-role-%d", time.Now().UnixMilli()%100000)
	out := runCmd(t, "role", "create", "--name", name, "--permissions", perms[0].ID, "--format", "json")
	assert.Contains(t, out, name)

	var role struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &role))
	require.NotEmpty(t, role.ID)
	t.Cleanup(func() { _ = c.DeleteRole(role.ID) })

	out = runCmd(t, "role", "delete", role.ID)
	assert.Contains(t, out, "deleted role "+role.ID)
}

func TestCmd_RoleAssignmentsList(t *testing.T) {
	out := runCmd(t, "role", "assignments", "--format", "json")
	assert.NotEmpty(t, out)
	var rows []api.AssignedRole
	require.NoError(t, json.Unmarshal([]byte(out), &rows))
	assert.NotEmpty(t, rows)

	foundSystemAdmin := false
	for _, a := range rows {
		if a.Role != nil && a.Role.Name == "System Admin" {
			foundSystemAdmin = true
			break
		}
	}
	assert.True(t, foundSystemAdmin, "expected a System Admin assignment in %s", out)
}

func TestCmd_RoleView(t *testing.T) {
	roles, err := newTestClient(t).ListRoles()
	require.NoError(t, err)
	require.NotEmpty(t, roles, "need at least one role on the instance")

	out := runCmd(t, "role", "view", roles[0].ID, "--format", "json")
	assert.Contains(t, out, roles[0].ID)
}

func TestCmd_RolePermissionList(t *testing.T) {
	c := newTestClient(t)
	name := fmt.Sprintf("e2e-rperm-list-%d", time.Now().UnixMilli()%100000)
	role, err := c.AddRole(name, "", nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteRole(role.ID) })

	out := runCmd(t, "role", "permission", "list", role.ID, "--format", "json")
	assert.Contains(t, out, "[")
}

func TestCmd_RolePermissionAddRemove(t *testing.T) {
	c := newTestClient(t)
	name := fmt.Sprintf("e2e-rperm-crud-%d", time.Now().UnixMilli()%100000)
	role, err := c.AddRole(name, "", nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteRole(role.ID) })

	perms, err := c.GetPermissions()
	require.NoError(t, err)
	require.NotEmpty(t, perms, "need at least one permission on the instance")
	permID := perms[0].ID

	runCmd(t, "role", "permission", "add", role.ID, permID)

	out := runCmd(t, "role", "permission", "list", role.ID, "--format", "json")
	assert.Contains(t, out, permID)

	runCmd(t, "role", "permission", "remove", role.ID, permID)

	out2 := runCmd(t, "role", "permission", "list", role.ID, "--format", "json")
	assert.NotContains(t, out2, permID)
}

func TestCmd_RoleGrantAssignmentsRevoke(t *testing.T) {
	c := newTestClient(t)

	successor, err := c.GetCurrentUser()
	require.NoError(t, err)
	login := fmt.Sprintf("cmdrole%d", time.Now().UnixMilli()%100000)
	user, err := c.AddUser(&api.AddUserRequest{
		Login:    login,
		FullName: "Cmd Role User",
		Email:    login + "@example.com",
		Password: "Test1234!",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteUser(login, successor.Login) })

	roles, err := c.ListRoles()
	require.NoError(t, err)
	require.NotEmpty(t, roles)
	roleID := roles[0].ID

	runCmd(t, "role", "grant", "--user", user.ID, "--role", roleID)

	out := runCmd(t, "role", "assignments", "--user", user.ID, "--format", "json")
	var assignments []api.AssignedRole
	require.NoError(t, json.Unmarshal([]byte(out), &assignments))

	var assignmentID string
	for _, a := range assignments {
		if a.Holder != nil && a.Holder.ID == user.ID && a.Role != nil && a.Role.ID == roleID {
			assignmentID = a.ID
			break
		}
	}
	require.NotEmpty(t, assignmentID, "granted role not found in assignments: %s", out)

	out = runCmd(t, "role", "revoke", assignmentID)
	assert.Contains(t, out, "revoked role assignment "+assignmentID)

	out = runCmd(t, "role", "assignments", "--user", user.ID, "--format", "json")
	require.NoError(t, json.Unmarshal([]byte(out), &assignments))
	for _, a := range assignments {
		assert.NotEqual(t, assignmentID, a.ID, "assignment was not revoked")
	}
}
