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

func TestListWorkflows(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/admin/workflows")
		json.NewEncoder(w).Encode([]api.Workflow{
			{ID: "wf-1", Name: "Default Workflow", Title: "Default"},
		})
	}))
	defer srv.Close()

	workflows, err := api.NewClient(srv.URL, "tok").ListWorkflows()

	require.NoError(t, err)
	assert.Len(t, workflows, 1)
	assert.Equal(t, "Default Workflow", workflows[0].Name)
}

func TestAddWorkflow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/api/admin/workflows")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "myWorkflow", body["name"])
		assert.Equal(t, "My Workflow", body["title"])
		json.NewEncoder(w).Encode(api.Workflow{ID: "wf-new", Name: "myWorkflow", Title: "My Workflow"})
	}))
	defer srv.Close()

	wf, err := api.NewClient(srv.URL, "tok").AddWorkflow("myWorkflow", "My Workflow")

	require.NoError(t, err)
	assert.Equal(t, "myWorkflow", wf.Name)
}

func TestDeleteWorkflow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "wf-1")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").DeleteWorkflow("wf-1")

	require.NoError(t, err)
}

func TestRestoreDefaultWorkflows(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "restoreStatus")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, true, body["isDefault"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").RestoreDefaultWorkflows()

	require.NoError(t, err)
}

func TestAddWorkflowRule(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "wf-1")
		assert.Contains(t, r.URL.Path, "rules")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "StateMachineRule", body["$type"])
		assert.Equal(t, "My Rule", body["name"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").AddWorkflowRule("wf-1", "StateMachineRule", "My Rule")

	require.NoError(t, err)
}
