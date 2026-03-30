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

func TestListBoards(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/agiles")
		json.NewEncoder(w).Encode([]api.Board{
			{ID: "b-1", Name: "Scrum Board"},
			{ID: "b-2", Name: "Kanban Board"},
		})
	}))
	defer srv.Close()

	boards, err := api.NewClient(srv.URL, "tok").ListBoards()

	require.NoError(t, err)
	assert.Len(t, boards, 2)
	assert.Equal(t, "Scrum Board", boards[0].Name)
}

func TestGetBoard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "b-1")
		json.NewEncoder(w).Encode(api.Board{
			ID: "b-1", Name: "Scrum Board",
			Sprints: []api.Sprint{{ID: "sp-1", Name: "Sprint 1"}},
		})
	}))
	defer srv.Close()

	board, err := api.NewClient(srv.URL, "tok").GetBoard("b-1")

	require.NoError(t, err)
	assert.Equal(t, "Scrum Board", board.Name)
	assert.Len(t, board.Sprints, 1)
}

func TestAddBoard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/api/agiles")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "My Board", body["name"])
		json.NewEncoder(w).Encode(api.Board{ID: "b-new", Name: "My Board"})
	}))
	defer srv.Close()

	board, err := api.NewClient(srv.URL, "tok").AddBoard("My Board")

	require.NoError(t, err)
	assert.Equal(t, "My Board", board.Name)
}

func TestDeleteBoard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "b-1")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").DeleteBoard("b-1")

	require.NoError(t, err)
}

func TestAddSprint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "b-1")
		assert.Contains(t, r.URL.Path, "sprints")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Sprint 1", body["name"])
		json.NewEncoder(w).Encode(api.Sprint{ID: "sp-new", Name: "Sprint 1"})
	}))
	defer srv.Close()

	sprint, err := api.NewClient(srv.URL, "tok").AddSprint("b-1", "Sprint 1", 0, 0, "")

	require.NoError(t, err)
	assert.Equal(t, "Sprint 1", sprint.Name)
}
