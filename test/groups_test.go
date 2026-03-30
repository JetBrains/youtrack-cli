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

func TestListGroups(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/groups")
		json.NewEncoder(w).Encode([]api.Group{
			{ID: "g-1", Name: "Developers"},
			{ID: "g-2", Name: "QA"},
		})
	}))
	defer srv.Close()

	groups, err := api.NewClient(srv.URL, "tok").ListGroups("")

	require.NoError(t, err)
	assert.Len(t, groups, 2)
	assert.Equal(t, "Developers", groups[0].Name)
}

func TestGetGroup_found(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]api.Group{{ID: "g-1", Name: "Developers"}})
	}))
	defer srv.Close()

	group, err := api.NewClient(srv.URL, "tok").GetGroup("Developers")

	require.NoError(t, err)
	assert.Equal(t, "g-1", group.ID)
}

func TestAddGroup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/api/groups")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "NewTeam", body["name"])
		json.NewEncoder(w).Encode(api.Group{ID: "g-new", Name: "NewTeam"})
	}))
	defer srv.Close()

	group, err := api.NewClient(srv.URL, "tok").AddGroup("NewTeam")

	require.NoError(t, err)
	assert.Equal(t, "NewTeam", group.Name)
}

func TestDeleteGroup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "g-1")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").DeleteGroup("g-1")

	require.NoError(t, err)
}
