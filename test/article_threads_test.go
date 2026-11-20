package api_test

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

// utf16Range returns the UTF-16 offsets of fragment in text, as YouTrack stores inline thread anchors.
func utf16Range(t *testing.T, text, fragment string) (start, end int) {
	t.Helper()
	i := strings.Index(text, fragment)
	require.GreaterOrEqual(t, i, 0, "%q not in %q", fragment, text)
	start = len(utf16.Encode([]rune(text[:i])))
	return start, start + len(utf16.Encode([]rune(fragment)))
}

func utf16Slice(text string, start, end int) string {
	units := utf16.Encode([]rune(text))
	if start < 0 || end > len(units) || start > end {
		return ""
	}
	return string(utf16.Decode(units[start:end]))
}

// seedThread creates an inline comment thread on fragment through the API, as the web UI does.
func seedThread(t *testing.T, c *api.Client, articleID, content, fragment string) api.ArticleTextMetadata {
	t.Helper()
	start, end := utf16Range(t, content, fragment)
	var tm api.ArticleTextMetadata
	require.NoError(t, c.Post(fmt.Sprintf("/api/articles/%s/textMetadatas?fields=id,startOffset,endOffset,comments(id)", articleID),
		map[string]any{
			"textFieldName": "content", "startOffset": start, "endOffset": end,
			"comments": []map[string]any{{"text": "thread on " + fragment}},
		}, &tm))
	require.NotEmpty(t, tm.ID)
	require.Len(t, tm.Comments, 1)
	return tm
}

func TestCmd_ArticleUpdateReanchorsInlineThreads(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	// Emoji and Cyrillic before the threads check that offsets are UTF-16 code units.
	const oldContent = "# Заголовок 😀\n\nalpha beta gamma\n\ndelta epsilon zeta\n\neta theta iota\n\nlast kappa line\n"
	const newContent = "# Заголовок 😀😀 длиннее\n\nalpha beta gamma\n\nnew words before delta epsilon, and zeta is gone\n\neta theta iota\n\nlast kappa line\n"
	article, err := c.AddArticle(proj.ID, "Inline threads", oldContent, "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteArticle(article.IDReadable) })

	shifted := seedThread(t, c, article.ID, oldContent, "beta")
	inside := seedThread(t, c, article.ID, oldContent, "delta epsilon")
	lost := seedThread(t, c, article.ID, oldContent, "epsilon zeta")
	after := seedThread(t, c, article.ID, oldContent, "theta")
	empty := seedThread(t, c, article.ID, oldContent, "kappa")
	require.NoError(t, c.Delete(fmt.Sprintf("/api/articles/%s/textMetadatas/%s/comments/%s", article.ID, empty.ID, empty.Comments[0].ID)))

	_, stderr := runCmdStderr(t, "article", "update", article.IDReadable, "--content", newContent, "--json")
	assert.Contains(t, stderr, "inline thread "+lost.ID)
	assert.Contains(t, stderr, "epsilon zeta")

	got, err := c.GetArticleText(article.IDReadable)
	require.NoError(t, err)
	require.Equal(t, newContent, got.Content)
	byID := map[string]api.ArticleTextMetadata{}
	for _, tm := range got.TextMetadatas {
		byID[tm.ID] = tm
	}
	for tm, fragment := range map[string]string{shifted.ID: "beta", inside.ID: "delta epsilon", after.ID: "theta"} {
		assert.Equal(t, fragment, utf16Slice(newContent, byID[tm].StartOffset, byID[tm].EndOffset), "thread %s", tm)
	}
	assert.Equal(t, lost.StartOffset, byID[lost.ID].StartOffset, "a thread whose text was rewritten stays put")
	assert.Equal(t, lost.EndOffset, byID[lost.ID].EndOffset)
	assert.Equal(t, empty.StartOffset, byID[empty.ID].StartOffset, "a thread without comments is not touched")
	assert.Equal(t, empty.EndOffset, byID[empty.ID].EndOffset)
}
