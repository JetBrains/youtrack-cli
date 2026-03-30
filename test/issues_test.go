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

func TestUpdateIssue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "FOO-1")
		var body api.UpdateIssueRequest
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Updated summary", body.Summary)
		json.NewEncoder(w).Encode(api.Issue{IDReadable: "FOO-1", Summary: "Updated summary"})
	}))
	defer srv.Close()

	issue, err := api.NewClient(srv.URL, "tok").UpdateIssue("FOO-1", api.UpdateIssueRequest{Summary: "Updated summary"})

	require.NoError(t, err)
	assert.Equal(t, "Updated summary", issue.Summary)
}

func TestListComments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "FOO-1")
		assert.Contains(t, r.URL.Path, "comments")
		json.NewEncoder(w).Encode([]api.Comment{
			{ID: "c-1", Text: "First comment"},
			{ID: "c-2", Text: "Second comment"},
		})
	}))
	defer srv.Close()

	comments, err := api.NewClient(srv.URL, "tok").ListComments("FOO-1")

	require.NoError(t, err)
	assert.Len(t, comments, 2)
	assert.Equal(t, "First comment", comments[0].Text)
}

func TestAddComment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "FOO-1")
		assert.Contains(t, r.URL.Path, "comments")
		var body api.AddCommentRequest
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Hello world", body.Text)
		json.NewEncoder(w).Encode(api.Comment{ID: "c-new", Text: "Hello world"})
	}))
	defer srv.Close()

	comment, err := api.NewClient(srv.URL, "tok").AddComment("FOO-1", api.AddCommentRequest{Text: "Hello world"})

	require.NoError(t, err)
	assert.Equal(t, "Hello world", comment.Text)
}

func TestDeleteComment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "FOO-1")
		assert.Contains(t, r.URL.Path, "c-1")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").DeleteComment("FOO-1", "c-1")

	require.NoError(t, err)
}

func TestGetIssueCount(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/api/issuesGetter/count")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "project: FOO", body["query"])
		json.NewEncoder(w).Encode(map[string]any{"count": 42})
	}))
	defer srv.Close()

	count, err := api.NewClient(srv.URL, "tok").GetIssueCount("project: FOO")

	require.NoError(t, err)
	assert.Equal(t, 42, count)
}

func TestGetIssueLinkTypes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/issueLinkTypes")
		json.NewEncoder(w).Encode([]api.LinkType{
			{ID: "lt-1", Name: "Depends on", SourceToTarget: "depends on", TargetToSource: "is required for"},
		})
	}))
	defer srv.Close()

	types, err := api.NewClient(srv.URL, "tok").GetIssueLinkTypes()

	require.NoError(t, err)
	assert.Len(t, types, 1)
	assert.Equal(t, "Depends on", types[0].Name)
}

func TestGetLinkedIssues(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "FOO-1")
		assert.Contains(t, r.URL.Path, "links")
		json.NewEncoder(w).Encode([]api.IssueLink{
			{ID: "link-1", Issues: []api.Issue{{IDReadable: "FOO-2"}}},
		})
	}))
	defer srv.Close()

	links, err := api.NewClient(srv.URL, "tok").GetLinkedIssues("FOO-1")

	require.NoError(t, err)
	assert.Len(t, links, 1)
	assert.Equal(t, "link-1", links[0].ID)
}

func TestGetIssueAttachments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "FOO-1")
		assert.Contains(t, r.URL.Path, "attachments")
		json.NewEncoder(w).Encode([]api.IssueAttachment{
			{ID: "att-1", Name: "screenshot.png", URL: "https://example.com/att-1"},
		})
	}))
	defer srv.Close()

	attachments, err := api.NewClient(srv.URL, "tok").GetIssueAttachments("FOO-1")

	require.NoError(t, err)
	assert.Len(t, attachments, 1)
	assert.Equal(t, "screenshot.png", attachments[0].Name)
}

func TestDeleteIssueAttachment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "FOO-1")
		assert.Contains(t, r.URL.Path, "att-1")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").DeleteIssueAttachment("FOO-1", "att-1")

	require.NoError(t, err)
}

func TestTagIssue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "FOO-1")
		assert.Contains(t, r.URL.Path, "tags")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "tag-1", body["id"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").TagIssue("FOO-1", "tag-1")

	require.NoError(t, err)
}

func TestAddIssueLinkType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/api/issueLinkTypes")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Depends on", body["name"])
		json.NewEncoder(w).Encode(api.LinkType{ID: "lt-new", Name: "Depends on"})
	}))
	defer srv.Close()

	lt, err := api.NewClient(srv.URL, "tok").AddIssueLinkType("Depends on", "depends on", "is required for")

	require.NoError(t, err)
	assert.Equal(t, "Depends on", lt.Name)
}

func TestLinkIssues(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "FOO-1")
		assert.Contains(t, r.URL.Path, "links")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		lt, _ := body["linkType"].(map[string]any)
		assert.Equal(t, "lt-1", lt["id"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").LinkIssues("FOO-1", "2-2", "lt-1")

	require.NoError(t, err)
}

func TestRemoveIssueLink(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "FOO-1")
		assert.Contains(t, r.URL.Path, "link-1")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").RemoveIssueLink("FOO-1", "link-1")

	require.NoError(t, err)
}

func TestAddWorkItem(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "FOO-1")
		assert.Contains(t, r.URL.Path, "workItems")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		dur, _ := body["duration"].(map[string]any)
		assert.Equal(t, float64(90), dur["minutes"])
		json.NewEncoder(w).Encode(api.WorkItem{ID: "wi-1"})
	}))
	defer srv.Close()

	wi, err := api.NewClient(srv.URL, "tok").AddWorkItem("FOO-1", 90, "design work", "")

	require.NoError(t, err)
	assert.Equal(t, "wi-1", wi.ID)
}

func TestUpdateComment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "FOO-1")
		assert.Contains(t, r.URL.Path, "c-1")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "updated text", body["text"])
		json.NewEncoder(w).Encode(api.Comment{ID: "c-1", Text: "updated text"})
	}))
	defer srv.Close()

	comment, err := api.NewClient(srv.URL, "tok").UpdateComment("FOO-1", "c-1", "updated text")

	require.NoError(t, err)
	assert.Equal(t, "updated text", comment.Text)
}

func TestUpdateCommentVisibility(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "FOO-1")
		assert.Contains(t, r.URL.Path, "c-1")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		vis, _ := body["visibility"].(map[string]any)
		assert.Equal(t, "LimitedVisibility", vis["$type"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").UpdateCommentVisibility("FOO-1", "c-1", []string{"g-1"}, []string{"u-1"})

	require.NoError(t, err)
}

func TestPinComment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "FOO-1")
		assert.Contains(t, r.URL.Path, "c-1")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, true, body["pinned"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").PinComment("FOO-1", "c-1", true)

	require.NoError(t, err)
}

func TestUpdateIssueVisibility(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "FOO-1")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		vis, _ := body["visibility"].(map[string]any)
		assert.Equal(t, "LimitedVisibility", vis["$type"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").UpdateIssueVisibility("FOO-1", []string{"g-1"}, []string{})

	require.NoError(t, err)
}

func TestGetCustomFieldValue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "FOO-1")
		assert.Contains(t, r.URL.Path, "fields")
		json.NewEncoder(w).Encode([]api.IssueField{
			{ID: "f-1", Name: "Priority", Value: "High"},
		})
	}))
	defer srv.Close()

	field, err := api.NewClient(srv.URL, "tok").GetCustomFieldValue("FOO-1", "Priority")

	require.NoError(t, err)
	assert.Equal(t, "Priority", field.Name)
}

func TestGetCommandAssist(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Method == "GET" {
			json.NewEncoder(w).Encode(api.Issue{ID: "2-1", IDReadable: "FOO-1"})
			return
		}
		assert.Contains(t, r.URL.Path, "/api/commands/assist")
		json.NewEncoder(w).Encode(map[string]any{"query": "State Fixed", "commands": []any{}})
	}))
	defer srv.Close()

	result, err := api.NewClient(srv.URL, "tok").GetCommandAssist("State", []string{"FOO-1"})

	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAddReaction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "FOO-1")
		assert.Contains(t, r.URL.Path, "c-1")
		assert.Contains(t, r.URL.Path, "reactions")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "👍", body["reaction"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").AddReaction("FOO-1", "c-1", "👍")

	require.NoError(t, err)
}

func TestDeleteReaction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "FOO-1")
		assert.Contains(t, r.URL.Path, "c-1")
		assert.Contains(t, r.URL.Path, "r-1")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").DeleteReaction("FOO-1", "c-1", "r-1")

	require.NoError(t, err)
}

func TestSetCustomField(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Method == "GET" {
			json.NewEncoder(w).Encode([]api.IssueField{{ID: "f-1", Name: "Priority"}})
			return
		}
		assert.Contains(t, r.URL.Path, "f-1")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		val, _ := body["value"].(map[string]any)
		assert.Equal(t, "High", val["name"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").SetCustomField("FOO-1", "Priority", "High", "")

	require.NoError(t, err)
	assert.Equal(t, 2, callCount)
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
