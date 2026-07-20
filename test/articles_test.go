package api_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

func TestCmd_ArticleList(t *testing.T) {
	out := runCmd(t, "article", "list", "--format", "json")
	// Output should be valid JSON array (possibly empty).
	assert.True(t, len(out) > 0)
}

func TestCmd_ArticleView(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	p, err := c.GetProject(proj.ShortName)
	require.NoError(t, err)

	article, err := c.AddArticle(p.ID, "Cmd Get Test Article", "", "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteArticle(article.IDReadable) })

	out := runCmd(t, "article", "view", article.IDReadable, "--format", "json")
	assert.Contains(t, out, article.IDReadable)
}

func TestCmd_ArticleCRUD(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	out := runCmd(t, "article", "create",
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

	out := runCmd(t, "article", "comment", "create", article.IDReadable,
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

func newTempArticle(t *testing.T, c *api.Client) *api.Article {
	t.Helper()
	proj := createTempProject(t, c)
	p, err := c.GetProject(proj.ShortName)
	require.NoError(t, err)
	article, err := c.AddArticle(p.ID, "e2e-article-"+t.Name(), "", "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteArticle(article.IDReadable) })
	return article
}

// newTempArticleInProject creates an article in an already-fetched project.
func newTempArticleInProject(t *testing.T, c *api.Client, projectID string) *api.Article {
	t.Helper()
	article, err := c.AddArticle(projectID, "e2e-article-"+t.Name(), "", "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteArticle(article.IDReadable) })
	return article
}

func TestCmd_ArticleCommentView(t *testing.T) {
	c := newTestClient(t)
	article := newTempArticle(t, c)

	comment, err := c.AddArticleComment(article.IDReadable, "e2e test comment")
	require.NoError(t, err)

	out := runCmd(t, "article", "comment", "view", article.IDReadable, comment.ID, "--format", "json")
	assert.Contains(t, out, comment.ID)
	assert.Contains(t, out, "e2e test comment")
}

func TestCmd_ArticleCommentPin(t *testing.T) {
	c := newTestClient(t)
	article := newTempArticle(t, c)

	comment, err := c.AddArticleComment(article.IDReadable, "pin me")
	require.NoError(t, err)

	out := runCmd(t, "article", "comment", "pin", article.IDReadable, comment.ID)
	assert.Contains(t, out, "pinned")
}

func TestCmd_ArticleCommentVisibility(t *testing.T) {
	c := newTestClient(t)
	article := newTempArticle(t, c)

	comment, err := c.AddArticleComment(article.IDReadable, "visibility me")
	require.NoError(t, err)

	out := runCmd(t, "article", "comment", "visibility", article.IDReadable, comment.ID)
	assert.Contains(t, out, "updated visibility for comment "+comment.ID)
}

func TestCmd_ArticleCommentReactionDelete(t *testing.T) {
	c := newTestClient(t)
	article := newTempArticle(t, c)

	comment, err := c.AddArticleComment(article.IDReadable, "reaction test")
	require.NoError(t, err)

	addOut := runCmd(t, "article", "comment", "reaction", "add", article.IDReadable, comment.ID, "--reaction", "👍")
	assert.Contains(t, addOut, "added reaction")

	reactions, err := c.ListArticleCommentReactions(article.IDReadable, comment.ID)
	require.NoError(t, err)
	require.NotEmpty(t, reactions, "expected at least one reaction after adding")
	reactionID := reactions[0].ID
	require.NotEmpty(t, reactionID)

	delOut := runCmd(t, "article", "comment", "reaction", "delete", article.IDReadable, comment.ID, reactionID, "--force")
	assert.Contains(t, delOut, "removed reaction from comment "+comment.ID)

	reactions, err = c.ListArticleCommentReactions(article.IDReadable, comment.ID)
	require.NoError(t, err)
	assert.Empty(t, reactions, "reaction should be gone after deletion")
}

func TestCmd_ArticleSetParent(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	p, err := c.GetProject(proj.ShortName)
	require.NoError(t, err)
	parent := newTempArticleInProject(t, c, p.ID)
	child := newTempArticleInProject(t, c, p.ID)

	// set-parent passes <parentId> arg directly to SetParentArticle which expects internal ID
	out := runCmd(t, "article", "set-parent", child.IDReadable, parent.ID)
	assert.Contains(t, out, "set parent of "+child.IDReadable)
}

func TestCmd_ArticleChildList(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	p, err := c.GetProject(proj.ShortName)
	require.NoError(t, err)
	parent := newTempArticleInProject(t, c, p.ID)
	child := newTempArticleInProject(t, c, p.ID)
	require.NoError(t, c.SetParentArticle(child.IDReadable, parent.ID))

	out := runCmd(t, "article", "child", "list", parent.IDReadable, "--format", "json")
	assert.Contains(t, out, child.IDReadable)
}

func TestCmd_ArticleParentGet(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	p, err := c.GetProject(proj.ShortName)
	require.NoError(t, err)
	parent := newTempArticleInProject(t, c, p.ID)
	child := newTempArticleInProject(t, c, p.ID)
	require.NoError(t, c.SetParentArticle(child.IDReadable, parent.ID))

	out := runCmd(t, "article", "parent", "view", child.IDReadable, "--format", "json")
	assert.Contains(t, out, parent.IDReadable)
}

func TestCmd_ArticleAttachmentAdd(t *testing.T) {
	c := newTestClient(t)
	article := newTempArticle(t, c)

	f, err := os.CreateTemp(t.TempDir(), "article-attach-*.txt")
	require.NoError(t, err)
	_, err = f.WriteString("article attachment content")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	runCmd(t, "article", "attachment", "add", article.IDReadable, "--file", f.Name())

	out := runCmd(t, "article", "attachment", "list", article.IDReadable, "--format", "json")
	assert.Contains(t, out, filepath.Base(f.Name()))
}

func TestCmd_ArticleVisibility(t *testing.T) {
	c := newTestClient(t)
	article := newTempArticle(t, c)

	out := runCmd(t, "article", "visibility", article.IDReadable)
	assert.Contains(t, out, "updated visibility for article "+article.IDReadable)
}
