package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListRoles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/roles")
		json.NewEncoder(w).Encode([]api.Role{
			{ID: "r-1", Name: "Developer", Description: "Can edit issues"},
		})
	}))
	defer srv.Close()

	roles, err := api.NewClient(srv.URL, "tok").ListRoles()

	require.NoError(t, err)
	assert.Len(t, roles, 1)
	assert.Equal(t, "Developer", roles[0].Name)
}

func TestAddRole(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/api/roles")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Manager", body["name"])
		json.NewEncoder(w).Encode(api.Role{ID: "r-new", Name: "Manager"})
	}))
	defer srv.Close()

	role, err := api.NewClient(srv.URL, "tok").AddRole("Manager", "Can manage", nil)

	require.NoError(t, err)
	assert.Equal(t, "Manager", role.Name)
}

func TestDeleteRole(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "r-1")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").DeleteRole("r-1")

	require.NoError(t, err)
}

func TestGetPermissions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/permissions")
		json.NewEncoder(w).Encode([]api.Permission{
			{ID: "perm-1", Name: "Read Issue"},
			{ID: "perm-2", Name: "Update Issue"},
		})
	}))
	defer srv.Close()

	perms, err := api.NewClient(srv.URL, "tok").GetPermissions()

	require.NoError(t, err)
	assert.Len(t, perms, 2)
	assert.Equal(t, "Read Issue", perms[0].Name)
}

func TestGrantRoleToUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/api/assignedRoles")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		holder, ok := body["holder"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "u-1", holder["id"])
		assert.Equal(t, "User", holder["$type"])
		role, ok := body["role"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "r-1", role["id"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").GrantRoleToUser("u-1", "r-1", "")

	require.NoError(t, err)
}

func TestGrantRoleToGroup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/api/assignedRoles")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		holder, _ := body["holder"].(map[string]any)
		assert.Equal(t, "g-1", holder["id"])
		assert.Equal(t, "UserGroup", holder["$type"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").GrantRoleToGroup("g-1", "r-1", "")

	require.NoError(t, err)
}

func TestUpdateRole(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "r-1")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		perms, _ := body["permissions"].([]any)
		assert.Len(t, perms, 2)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").UpdateRole("r-1", []string{"perm-1", "perm-2"})

	require.NoError(t, err)
}
