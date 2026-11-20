package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"unicode/utf16"

	"github.com/spf13/cobra"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/internal"
	"github.com/JetBrains/youtrack-cli/internal/output"
)

// fragmentPreview is how many characters of a lost thread's fragment a warning quotes.
const fragmentPreview = 50

// contentThreads returns the article's inline threads anchored to its content, skipping threads
// whose comments were all deleted: the web UI hides them and they cannot be removed.
func contentThreads(text *api.ArticleText) []internal.TextAnchor {
	var anchors []internal.TextAnchor
	for _, tm := range text.TextMetadatas {
		if len(tm.Comments) == 0 || (tm.TextFieldName != "" && tm.TextFieldName != "content") {
			continue
		}
		anchors = append(anchors, internal.TextAnchor{ID: tm.ID, Start: tm.StartOffset, End: tm.EndOffset})
	}
	return anchors
}

// reanchorThreads moves the inline threads of articleID from the old content to the new one.
// YouTrack keeps thread offsets unchanged when the content changes, so without this they point at the wrong text.
// A thread that cannot be found or moved stays where it is, with a warning.
// The returned error lists the threads whose move failed for a reason other than permissions;
// the content is already saved then, so the caller reports it after printing the article.
func reanchorThreads(cmd *cobra.Command, client *api.Client, articleID string, before *api.ArticleText, content string) error {
	moved, lost := internal.Reanchor(before.Content, content, contentThreads(before))
	for _, a := range lost {
		fragment, ok := fragmentAt(before.Content, a)
		if !ok {
			output.PrintWarn(cmd, fmt.Sprintf("inline thread %s: offsets %d-%d are outside the old content, left as is", a.ID, a.Start, a.End))
			continue
		}
		output.PrintWarn(cmd, fmt.Sprintf("inline thread %s: fragment %q not found in the new content, left at its old position", a.ID, fragment))
	}
	var failed []error
	for _, a := range moved {
		err := client.MoveArticleTextMetadata(articleID, a.ID, a.Start, a.End)
		if err == nil {
			continue
		}
		var apiErr *api.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusForbidden {
			output.PrintWarn(cmd, fmt.Sprintf("inline thread %s: no permission to move it, left at its old position", a.ID))
			continue
		}
		output.PrintWarn(cmd, fmt.Sprintf("inline thread %s: could not move it to offsets %d-%d: %v", a.ID, a.Start, a.End, err))
		failed = append(failed, fmt.Errorf("thread %s: %w", a.ID, err))
	}
	if len(failed) > 0 {
		return fmt.Errorf("article content updated, but %d inline threads were not moved: %w", len(failed), errors.Join(failed...))
	}
	return nil
}

// fragmentAt returns the start of the text that anchor a points at; ok is false when its offsets are out of range.
func fragmentAt(text string, a internal.TextAnchor) (fragment string, ok bool) {
	units := utf16.Encode([]rune(text))
	if a.Start < 0 || a.End < a.Start || a.End > len(units) {
		return "", false
	}
	runes := utf16.Decode(units[a.Start:a.End])
	if len(runes) > fragmentPreview {
		return string(runes[:fragmentPreview]) + "…", true
	}
	return string(runes), true
}
