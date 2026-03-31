package api_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmd_GroupList(t *testing.T) {
	c := newTestClient(t)
	groupName := fmt.Sprintf("testgroup-cmd-%d", time.Now().UnixMilli()%100000)
	group, err := c.AddGroup(groupName)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteGroup(group.ID) })

	out := runCmd(t, "group", "list", "--format", "json")
	assert.Contains(t, out, group.Name)
}

func TestCmd_GroupGet(t *testing.T) {
	c := newTestClient(t)
	groupName := fmt.Sprintf("testgroup-get-%d", time.Now().UnixMilli()%100000)
	group, err := c.AddGroup(groupName)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteGroup(group.ID) })

	out := runCmd(t, "group", "get", group.Name, "--format", "json")
	assert.Contains(t, out, group.Name)
}

func TestGroup_CRUD(t *testing.T) {
	c := newTestClient(t)

	// Use a timestamp to ensure uniqueness (group names must be unique).
	groupName := fmt.Sprintf("testgroup-%d", time.Now().UnixMilli()%100000)

	// Create
	group, err := c.AddGroup(groupName)
	require.NoError(t, err)
	require.NotEmpty(t, group.ID)
	// Note: DeleteGroup requires a ?successor= query param that the current API
	// method doesn't support, so cleanup is best-effort only.
	t.Cleanup(func() { _ = c.DeleteGroup(group.ID) })

	assert.Equal(t, groupName, group.Name)

	// List
	groups, err := c.ListGroups("")
	require.NoError(t, err)
	found := false
	for _, g := range groups {
		if g.ID == group.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "newly created group should appear in list")

	// Get by name
	got, err := c.GetGroup(group.Name)
	require.NoError(t, err)
	assert.Equal(t, group.ID, got.ID)
}
