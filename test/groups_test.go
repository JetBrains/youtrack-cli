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

func TestCmd_GroupGet(t *testing.T) {
	c := newTestClient(t)
	groupName := fmt.Sprintf("testgroup-get-%d", time.Now().UnixMilli()%100000)
	group, err := c.AddGroup(groupName)
	require.NoError(t, err)
	successor := groupDeleteSuccessor(t, c, group.ID)
	t.Cleanup(func() { _ = c.DeleteGroup(group.ID, successor.ID) })

	out := runCmd(t, "group", "get", group.Name, "--format", "json")
	assert.Contains(t, out, group.Name)
}

func TestCmd_GroupDelete(t *testing.T) {
	c := newTestClient(t)
	groupName := fmt.Sprintf("testgroup-del-%d", time.Now().UnixMilli()%100000)
	group, err := c.AddGroup(groupName)
	require.NoError(t, err)
	successor := groupDeleteSuccessor(t, c, group.ID)

	out := runCmd(t, "group", "delete", group.ID, "--successor", successor.ID)
	assert.Contains(t, out, "deleted group "+group.ID)

	_, err = c.GetGroup(group.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
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
