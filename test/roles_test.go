package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRole_ListRoles(t *testing.T) {
	c := newTestClient(t)

	roles, err := c.ListRoles()
	require.NoError(t, err)
	// Every YouTrack instance has built-in roles.
	assert.NotEmpty(t, roles)
}

func TestRole_GetPermissions(t *testing.T) {
	c := newTestClient(t)

	perms, err := c.GetPermissions()
	require.NoError(t, err)
	assert.NotEmpty(t, perms)
}

func TestRole_CRUD(t *testing.T) {
	c := newTestClient(t)

	// Create with no permissions
	role, err := c.AddRole("TestRole"+t.Name(), "integration test role", nil)
	require.NoError(t, err)
	require.NotEmpty(t, role.ID)
	t.Cleanup(func() { _ = c.DeleteRole(role.ID) })

	assert.Contains(t, role.Name, "TestRole")

	// List and verify it appears
	roles, err := c.ListRoles()
	require.NoError(t, err)
	found := false
	for _, r := range roles {
		if r.ID == role.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "newly created role should appear in list")

	// Delete
	err = c.DeleteRole(role.ID)
	require.NoError(t, err)
}
