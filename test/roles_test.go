package api_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmd_RoleList(t *testing.T) {
	out := runCmd(t, "role", "list", "--format", "json")
	assert.NotEmpty(t, out)
}

func TestCmd_RolePermissions(t *testing.T) {
	out := runCmd(t, "role", "permissions", "--format", "json")
	assert.NotEmpty(t, out)
}

func TestCmd_RoleAddDelete(t *testing.T) {
	c := newTestClient(t)
	perms, err := c.GetPermissions()
	require.NoError(t, err)
	require.NotEmpty(t, perms)

	name := fmt.Sprintf("cmd-role-%d", time.Now().UnixMilli()%100000)
	out := runCmd(t, "role", "add", "--name", name, "--permissions", perms[0].ID, "--format", "json")
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
