package api_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmd_ArticleList(t *testing.T) {
	out := runCmd(t, "article", "list", "--format", "json")
	// Output should be valid JSON array (possibly empty).
	assert.True(t, len(out) > 0)
}

func TestCmd_ArticleGet(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	p, err := c.GetProject(proj.ShortName)
	require.NoError(t, err)

	article, err := c.AddArticle(p.ID, "Cmd Get Test Article", "", "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteArticle(article.IDReadable) })

	out := runCmd(t, "article", "get", article.IDReadable, "--format", "json")
	assert.Contains(t, out, article.IDReadable)
}

func TestCmd_ArticleCRUD(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	out := runCmd(t, "article", "add",
		"--project", proj.ShortName,
		"--summary", "Cmd Article Add",
		"--content", "Initial content",
		"--format", "json",
	)
	assert.Contains(t, out, "Cmd Article Add")

	var article struct {
		IDReadable string `json:"idReadable"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &article))
	require.NotEmpty(t, article.IDReadable)
	t.Cleanup(func() { _ = c.DeleteArticle(article.IDReadable) })

	out = runCmd(t, "article", "update", article.IDReadable,
		"--summary", "Cmd Article Updated",
		"--content", "Updated content",
		"--format", "json",
	)
	assert.Contains(t, out, "Cmd Article Updated")

	got, err := c.GetArticle(article.IDReadable)
	require.NoError(t, err)
	assert.Equal(t, "Cmd Article Updated", got.Summary)
	assert.Equal(t, "Updated content", got.Content)

	out = runCmd(t, "article", "delete", article.IDReadable, "--force")
	assert.Contains(t, out, "deleted article "+article.IDReadable)

	_, err = c.GetArticle(article.IDReadable)
	require.Error(t, err)
}

func TestCmd_ArticleCommentCRUD(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	project, err := c.GetProject(proj.ShortName)
	require.NoError(t, err)

	article, err := c.AddArticle(project.ID, "Cmd Article Comment", "", "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteArticle(article.IDReadable) })

	out := runCmd(t, "article", "comment", "add", article.IDReadable,
		"--text", "article comment from cli",
		"--format", "json",
	)
	assert.Contains(t, out, "article comment from cli")

	var comment struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &comment))
	require.NotEmpty(t, comment.ID)

	out = runCmd(t, "article", "comment", "list", article.IDReadable, "--format", "json")
	assert.Contains(t, out, comment.ID)

	out = runCmd(t, "article", "comment", "update", article.IDReadable, comment.ID,
		"--text", "article comment updated",
		"--format", "json",
	)
	assert.Contains(t, out, "article comment updated")

	comments, err := c.ListArticleComments(article.IDReadable)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "article comment updated", comments[0].Text)

	out = runCmd(t, "article", "comment", "delete", article.IDReadable, comment.ID, "--force")
	assert.Contains(t, out, "deleted comment")

	comments, err = c.ListArticleComments(article.IDReadable)
	require.NoError(t, err)
	assert.Empty(t, comments)
}
