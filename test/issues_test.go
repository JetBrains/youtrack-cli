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

func TestGetIssue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "FOO-1")
		json.NewEncoder(w).Encode(api.Issue{ID: "2-1", IDReadable: "FOO-1", Summary: "Test"})
	}))
	defer srv.Close()

	issue, err := api.NewClient(srv.URL, "tok").GetIssue("FOO-1")

	require.NoError(t, err)
	assert.Equal(t, "FOO-1", issue.IDReadable)
	assert.Equal(t, "Test", issue.Summary)
}

func TestListIssues(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "query=")
		json.NewEncoder(w).Encode([]api.Issue{{IDReadable: "FOO-1"}, {IDReadable: "FOO-2"}})
	}))
	defer srv.Close()

	issues, err := api.NewClient(srv.URL, "tok").ListIssues("project: FOO", 10)

	require.NoError(t, err)
	assert.Len(t, issues, 2)
	assert.Equal(t, "FOO-1", issues[0].IDReadable)
}

func TestAddIssue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		var body api.AddIssueRequest
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "New issue", body.Summary)
		json.NewEncoder(w).Encode(api.Issue{IDReadable: "FOO-3", Summary: "New issue"})
	}))
	defer srv.Close()

	issue, err := api.NewClient(srv.URL, "tok").AddIssue(api.AddIssueRequest{
		Summary: "New issue",
		Project: &api.Project{ID: "0-0"},
	})

	require.NoError(t, err)
	assert.Equal(t, "FOO-3", issue.IDReadable)
}

func TestDeleteIssue(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "FOO-1")
		called = true
		w.WriteHeader(204)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").DeleteIssue("FOO-1")

	require.NoError(t, err)
	assert.True(t, called)
}

func TestApplyCommand(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Method == "GET" {
			json.NewEncoder(w).Encode(api.Issue{ID: "2-1", IDReadable: "FOO-1"})
			return
		}
		var body api.ApplyCommandRequest
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "State Fixed", body.Query)
		assert.Len(t, body.Issues, 1)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").ApplyCommand("State Fixed", []string{"FOO-1"})

	require.NoError(t, err)
	assert.Equal(t, 2, callCount) // one GET to resolve ID, one POST for command
}

func TestWrapUserValues(t *testing.T) {
	cases := []struct{ in, out string }{
		{"assignee alice", "assignee alice"},
		{"assignee alice.qa", "assignee {alice.qa}"},
		{"reporter bob-dev State Fixed", "reporter {bob-dev} State Fixed"},
		{"assignee {alice.qa}", "assignee {alice.qa}"},
	}
	for _, c := range cases {
		assert.Equal(t, c.out, api.WrapUserValues(c.in), "input: %q", c.in)
	}
}
