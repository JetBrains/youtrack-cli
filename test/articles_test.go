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

func TestListArticles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/articles")
		json.NewEncoder(w).Encode([]api.Article{
			{ID: "a-1", IDReadable: "KB-1", Summary: "How to setup"},
			{ID: "a-2", IDReadable: "KB-2", Summary: "Troubleshooting"},
		})
	}))
	defer srv.Close()

	articles, err := api.NewClient(srv.URL, "tok").ListArticles("")

	require.NoError(t, err)
	assert.Len(t, articles, 2)
	assert.Equal(t, "KB-1", articles[0].IDReadable)
}

func TestGetArticle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "KB-1")
		json.NewEncoder(w).Encode(api.Article{ID: "a-1", IDReadable: "KB-1", Summary: "How to setup", Content: "Content here"})
	}))
	defer srv.Close()

	article, err := api.NewClient(srv.URL, "tok").GetArticle("KB-1")

	require.NoError(t, err)
	assert.Equal(t, "KB-1", article.IDReadable)
	assert.Equal(t, "Content here", article.Content)
}

func TestAddArticle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/api/articles")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "How to configure", body["summary"])
		proj, ok := body["project"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "proj-1", proj["id"])
		json.NewEncoder(w).Encode(api.Article{ID: "a-new", IDReadable: "KB-3", Summary: "How to configure"})
	}))
	defer srv.Close()

	article, err := api.NewClient(srv.URL, "tok").AddArticle("proj-1", "How to configure", "", "")

	require.NoError(t, err)
	assert.Equal(t, "KB-3", article.IDReadable)
}

func TestUpdateArticle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "KB-1")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Updated title", body["summary"])
		json.NewEncoder(w).Encode(api.Article{IDReadable: "KB-1", Summary: "Updated title"})
	}))
	defer srv.Close()

	article, err := api.NewClient(srv.URL, "tok").UpdateArticle("KB-1", "Updated title", "")

	require.NoError(t, err)
	assert.Equal(t, "Updated title", article.Summary)
}

func TestDeleteArticle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "KB-1")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").DeleteArticle("KB-1")

	require.NoError(t, err)
}

func TestListArticleComments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "KB-1")
		assert.Contains(t, r.URL.Path, "comments")
		json.NewEncoder(w).Encode([]api.Comment{
			{ID: "c-1", Text: "Good article"},
		})
	}))
	defer srv.Close()

	comments, err := api.NewClient(srv.URL, "tok").ListArticleComments("KB-1")

	require.NoError(t, err)
	assert.Len(t, comments, 1)
	assert.Equal(t, "Good article", comments[0].Text)
}

func TestAddArticleComment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "KB-1")
		assert.Contains(t, r.URL.Path, "comments")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Thanks!", body["text"])
		json.NewEncoder(w).Encode(api.Comment{ID: "c-new", Text: "Thanks!"})
	}))
	defer srv.Close()

	comment, err := api.NewClient(srv.URL, "tok").AddArticleComment("KB-1", "Thanks!", "")

	require.NoError(t, err)
	assert.Equal(t, "Thanks!", comment.Text)
}

func TestDeleteArticleComment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "KB-1")
		assert.Contains(t, r.URL.Path, "c-1")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").DeleteArticleComment("KB-1", "c-1")

	require.NoError(t, err)
}

func TestListArticleAttachments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "KB-1")
		assert.Contains(t, r.URL.Path, "attachments")
		json.NewEncoder(w).Encode([]api.ArticleAttachment{
			{ID: "att-1", Name: "diagram.png", URL: "https://example.com/att-1"},
		})
	}))
	defer srv.Close()

	attachments, err := api.NewClient(srv.URL, "tok").ListArticleAttachments("KB-1")

	require.NoError(t, err)
	assert.Len(t, attachments, 1)
	assert.Equal(t, "diagram.png", attachments[0].Name)
}

func TestUpdateArticleComment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "A-1")
		assert.Contains(t, r.URL.Path, "c-1")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "updated text", body["text"])
		json.NewEncoder(w).Encode(api.Comment{ID: "c-1", Text: "updated text"})
	}))
	defer srv.Close()

	comment, err := api.NewClient(srv.URL, "tok").UpdateArticleComment("A-1", "c-1", "updated text")

	require.NoError(t, err)
	assert.Equal(t, "updated text", comment.Text)
}

func TestSetParentArticle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "A-1")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		parent, _ := body["parentArticle"].(map[string]any)
		assert.Equal(t, "A-0", parent["id"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").SetParentArticle("A-1", "A-0")

	require.NoError(t, err)
}

func TestTagArticle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "A-1")
		assert.Contains(t, r.URL.Path, "tags")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "tag-1", body["id"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").TagArticle("A-1", "tag-1")

	require.NoError(t, err)
}

func TestUpdateArticleVisibility(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "A-1")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		vis, _ := body["visibility"].(map[string]any)
		assert.Equal(t, "LimitedVisibility", vis["$type"])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := api.NewClient(srv.URL, "tok").UpdateArticleVisibility("A-1", []string{"g-1"}, []string{})

	require.NoError(t, err)
}
