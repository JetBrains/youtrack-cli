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

	out := runCmd(t, "group", "member", "list", group.ID, "--format", "json")
	assert.Contains(t, out, "[")
}

func TestCmd_GroupRoleList(t *testing.T) {
	c := newTestClient(t)
	name := fmt.Sprintf("e2e-grp-role-%d", time.Now().UnixMilli()%100000)
	group, err := c.AddGroup(name)
	require.NoError(t, err)
	successor := groupDeleteSuccessor(t, c, group.ID)
	t.Cleanup(func() { _ = c.DeleteGroup(group.ID, successor.ID) })

	// Some YouTrack versions expose project roles as a Hub-only endpoint; skip gracefully if unavailable.
	roles, err := c.ListGroupProjectRoles(group.ID)
	if err != nil {
		t.Skipf("group role list not supported on this instance: %v", err)
	}
	_ = roles

	out := runCmd(t, "group", "role", "list", group.ID, "--format", "json")
	assert.Contains(t, out, "[")
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
