package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListProjects(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Project{
			{ID: "0-1", ShortName: "FOO", Name: "Foo Project"},
			{ID: "0-2", ShortName: "BAR", Name: "Bar Project"},
		})
	}))
	defer srv.Close()

	projects, err := NewClient(srv.URL, "tok").ListProjects()

	require.NoError(t, err)
	assert.Len(t, projects, 2)
	assert.Equal(t, "FOO", projects[0].ShortName)
}

func TestGetProject_found(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Project{{ID: "0-1", ShortName: "FOO", Name: "Foo Project"}})
	}))
	defer srv.Close()

	project, err := NewClient(srv.URL, "tok").GetProject("FOO")

	require.NoError(t, err)
	assert.Equal(t, "FOO", project.ShortName)
	assert.Equal(t, "0-1", project.ID)
}

func TestGetProject_notFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Project{})
	}))
	defer srv.Close()

	_, err := NewClient(srv.URL, "tok").GetProject("MISSING")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "MISSING")
}

func TestDeleteProject(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Method == "GET" {
			json.NewEncoder(w).Encode([]Project{{ID: "0-1", ShortName: "FOO", Name: "Foo Project"}})
		} else {
			assert.Equal(t, "DELETE", r.Method)
			assert.Contains(t, r.URL.Path, "0-1")
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer srv.Close()

	err := NewClient(srv.URL, "tok").DeleteProject("FOO")

	require.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

func TestAddProject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		var body AddProjectRequest
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "FOO", body.ShortName)
		json.NewEncoder(w).Encode(Project{ID: "0-1", ShortName: "FOO", Name: "Foo Project"})
	}))
	defer srv.Close()

	proj, err := NewClient(srv.URL, "tok").AddProject(AddProjectRequest{
		ShortName: "FOO",
		Name:      "Foo Project",
		Leader:    &User{ID: "1-1"},
	})

	require.NoError(t, err)
	assert.Equal(t, "FOO", proj.ShortName)
}
