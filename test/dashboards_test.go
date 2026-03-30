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

func TestListDashboards(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/dashboards")
		json.NewEncoder(w).Encode([]api.Dashboard{
			{ID: "d-1", Name: "My Dashboard"},
		})
	}))
	defer srv.Close()

	dashboards, err := api.NewClient(srv.URL, "tok").ListDashboards()

	require.NoError(t, err)
	assert.Len(t, dashboards, 1)
	assert.Equal(t, "My Dashboard", dashboards[0].Name)
}

func TestAddDashboard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/api/dashboards")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "New Dashboard", body["name"])
		json.NewEncoder(w).Encode(api.Dashboard{ID: "d-new", Name: "New Dashboard"})
	}))
	defer srv.Close()

	dashboard, err := api.NewClient(srv.URL, "tok").AddDashboard("New Dashboard")

	require.NoError(t, err)
	assert.Equal(t, "New Dashboard", dashboard.Name)
}

func TestDeleteDashboard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "d-1")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").DeleteDashboard("d-1")

	require.NoError(t, err)
}

func TestUpdateDashboard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "d-1")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Renamed Dashboard", body["name"])
		json.NewEncoder(w).Encode(api.Dashboard{ID: "d-1", Name: "Renamed Dashboard"})
	}))
	defer srv.Close()

	dashboard, err := api.NewClient(srv.URL, "tok").UpdateDashboard("d-1", "Renamed Dashboard")

	require.NoError(t, err)
	assert.Equal(t, "Renamed Dashboard", dashboard.Name)
}
