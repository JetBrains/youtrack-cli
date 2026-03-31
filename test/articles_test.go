package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArticle_CRUD(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	// Get the project to obtain its opaque ID.
	p, err := c.GetProject(proj.ShortName)
	require.NoError(t, err)

	article, err := c.AddArticle(p.ID, "Integration Test Article", "Test content", "")
	require.NoError(t, err)
	require.NotEmpty(t, article.IDReadable)
	t.Cleanup(func() { _ = c.DeleteArticle(article.IDReadable) })

	got, err := c.GetArticle(article.IDReadable)
	require.NoError(t, err)
	assert.Equal(t, article.IDReadable, got.IDReadable)
	assert.Equal(t, "Integration Test Article", got.Summary)

	updated, err := c.UpdateArticle(article.IDReadable, "Updated Title", "")
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", updated.Summary)
}

func TestArticle_List(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	p, err := c.GetProject(proj.ShortName)
	require.NoError(t, err)

	article, err := c.AddArticle(p.ID, "List Test Article", "", "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteArticle(article.IDReadable) })

	articles, err := c.ListArticles("")
	require.NoError(t, err)
	found := false
	for _, a := range articles {
		if a.IDReadable == article.IDReadable {
			found = true
			break
		}
	}
	assert.True(t, found, "created article should appear in list")
}

func TestArticle_Comments(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	p, err := c.GetProject(proj.ShortName)
	require.NoError(t, err)

	article, err := c.AddArticle(p.ID, "Comment Test Article", "", "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteArticle(article.IDReadable) })

	comment, err := c.AddArticleComment(article.IDReadable, "Test comment", "")
	require.NoError(t, err)
	require.NotEmpty(t, comment.ID)
	assert.Equal(t, "Test comment", comment.Text)

	comments, err := c.ListArticleComments(article.IDReadable)
	require.NoError(t, err)
	assert.NotEmpty(t, comments)

	updated, err := c.UpdateArticleComment(article.IDReadable, comment.ID, "Updated comment")
	require.NoError(t, err)
	assert.Equal(t, "Updated comment", updated.Text)

	err = c.DeleteArticleComment(article.IDReadable, comment.ID)
	require.NoError(t, err)
}

func TestArticle_Attachments(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	p, err := c.GetProject(proj.ShortName)
	require.NoError(t, err)

	article, err := c.AddArticle(p.ID, "Attachment Test Article", "", "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteArticle(article.IDReadable) })

	attachments, err := c.ListArticleAttachments(article.IDReadable)
	require.NoError(t, err)
	assert.NotNil(t, attachments)
}

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
