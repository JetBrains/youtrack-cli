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

func TestListProjects(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "$top=-1")
		json.NewEncoder(w).Encode([]api.Project{
			{ID: "0-1", ShortName: "FOO", Name: "Foo Project"},
			{ID: "0-2", ShortName: "BAR", Name: "Bar Project"},
		})
	}))
	defer srv.Close()

	projects, err := api.NewClient(srv.URL, "tok").ListProjects()

	require.NoError(t, err)
	assert.Len(t, projects, 2)
	assert.Equal(t, "FOO", projects[0].ShortName)
}

func TestGetProject_found(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]api.Project{{ID: "0-1", ShortName: "FOO", Name: "Foo Project"}})
	}))
	defer srv.Close()

	project, err := api.NewClient(srv.URL, "tok").GetProject("FOO")

	require.NoError(t, err)
	assert.Equal(t, "FOO", project.ShortName)
	assert.Equal(t, "0-1", project.ID)
}

func TestGetProject_notFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]api.Project{})
	}))
	defer srv.Close()

	_, err := api.NewClient(srv.URL, "tok").GetProject("MISSING")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "MISSING")
}

func TestDeleteProject(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Method == "GET" {
			json.NewEncoder(w).Encode([]api.Project{{ID: "0-1", ShortName: "FOO", Name: "Foo Project"}})
		} else {
			assert.Equal(t, "DELETE", r.Method)
			assert.Contains(t, r.URL.Path, "0-1")
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").DeleteProject("FOO")

	require.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

func TestArchiveProject(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Method == "GET" {
			json.NewEncoder(w).Encode([]api.Project{{ID: "0-1", ShortName: "FOO"}})
			return
		}
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "0-1")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, true, body["archived"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").ArchiveProject("FOO")

	require.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

func TestGetProjectTemplates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/admin/projectTemplates")
		json.NewEncoder(w).Encode([]api.ProjectTemplate{
			{ID: "tmpl-1", Name: "Scrum", ShortName: "SCRUM"},
		})
	}))
	defer srv.Close()

	templates, err := api.NewClient(srv.URL, "tok").GetProjectTemplates()

	require.NoError(t, err)
	assert.Len(t, templates, 1)
	assert.Equal(t, "Scrum", templates[0].Name)
}

func TestAddProjectCustomField(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Method == "GET" {
			json.NewEncoder(w).Encode([]api.Project{{ID: "0-1", ShortName: "FOO"}})
			return
		}
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "fields")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		field, ok := body["field"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "cf-1", field["id"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").AddProjectCustomField("FOO", "cf-1")

	require.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

func TestAddUserToProjectTeam(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Method == "GET" {
			json.NewEncoder(w).Encode([]api.Project{{ID: "0-1", ShortName: "FOO"}})
			return
		}
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "team")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		users, ok := body["users"].([]any)
		require.True(t, ok)
		assert.Len(t, users, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").AddUserToProjectTeam("FOO", "u-1")

	require.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

func TestAttachWorkflowToProject(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Method == "GET" {
			json.NewEncoder(w).Encode([]api.Project{{ID: "0-1", ShortName: "FOO"}})
			return
		}
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "workflows")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		wf, ok := body["workflow"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "wf-1", wf["id"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").AttachWorkflowToProject("FOO", "wf-1")

	require.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

func TestAddProject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		var body api.AddProjectRequest
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "FOO", body.ShortName)
		json.NewEncoder(w).Encode(api.Project{ID: "0-1", ShortName: "FOO", Name: "Foo Project"})
	}))
	defer srv.Close()

	proj, err := api.NewClient(srv.URL, "tok").AddProject(api.AddProjectRequest{
		ShortName: "FOO",
		Name:      "Foo Project",
		Leader:    &api.User{ID: "1-1"},
	})

	require.NoError(t, err)
	assert.Equal(t, "FOO", proj.ShortName)
}

func TestGetProjectCustomFields(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			json.NewEncoder(w).Encode([]api.Project{{ID: "p-1", ShortName: "FOO"}})
			return
		}
		assert.Contains(t, r.URL.Path, "p-1")
		assert.Contains(t, r.URL.Path, "fields")
		json.NewEncoder(w).Encode([]api.ProjectCustomField{
			{ID: "pcf-1", Field: &api.CustomField{Name: "Priority"}},
		})
	}))
	defer srv.Close()

	fields, err := api.NewClient(srv.URL, "tok").GetProjectCustomFields("FOO")

	require.NoError(t, err)
	assert.Len(t, fields, 1)
}

func TestAddGroupToProjectTeam(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			json.NewEncoder(w).Encode([]api.Project{{ID: "p-1", ShortName: "FOO"}})
			return
		}
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "team")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		groups, _ := body["groups"].([]any)
		assert.Len(t, groups, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").AddGroupToProjectTeam("FOO", "g-1")

	require.NoError(t, err)
}

func TestConvertProjectToTemplate(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			json.NewEncoder(w).Encode([]api.Project{{ID: "p-1", ShortName: "FOO"}})
			return
		}
		assert.Equal(t, "POST", r.Method)
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, true, body["template"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").ConvertProjectToTemplate("FOO")

	require.NoError(t, err)
}

func TestSetProjectTimeTracking(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			json.NewEncoder(w).Encode([]api.Project{{ID: "p-1", ShortName: "FOO"}})
			return
		}
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "timeTrackingSettings")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, true, body["enabled"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").SetProjectTimeTracking("FOO", true)

	require.NoError(t, err)
}
