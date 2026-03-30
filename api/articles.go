package api

import (
	"fmt"
	"net/url"
	"path/filepath"
)

const articleFields = "id,idReadable,summary,content,reporter(id,login,fullName),project(id,shortName),created"
const articleAttachmentFields = "id,name,url,mimeType"
const articleCommentFields = "id,text,author(id,login,fullName),created"

func (c *Client) ListArticles(query string) ([]Article, error) {
	var result []Article
	path := fmt.Sprintf("/api/articles?fields=%s&query=%s&$top=-1", articleFields, url.QueryEscape(query))
	return result, c.Get(path, &result)
}

func (c *Client) GetArticle(id string) (*Article, error) {
	var result Article
	path := fmt.Sprintf("/api/articles/%s?fields=%s", url.PathEscape(id), articleFields)
	return &result, c.Get(path, &result)
}

func (c *Client) AddArticle(projectID, summary, content, reporterID string) (*Article, error) {
	var result Article
	body := map[string]any{
		"project": map[string]any{"id": projectID},
		"summary": summary,
		"content": content,
	}
	if reporterID != "" {
		body["reporter"] = map[string]any{"id": reporterID}
	}
	return &result, c.Post("/api/articles?fields="+articleFields, body, &result)
}

func (c *Client) UpdateArticle(id, summary, content string) (*Article, error) {
	var result Article
	body := map[string]any{}
	if summary != "" {
		body["summary"] = summary
	}
	if content != "" {
		body["content"] = content
	}
	path := fmt.Sprintf("/api/articles/%s?fields=%s", url.PathEscape(id), articleFields)
	return &result, c.Post(path, body, &result)
}

func (c *Client) DeleteArticle(id string) error {
	return c.Delete(fmt.Sprintf("/api/articles/%s", url.PathEscape(id)))
}

func (c *Client) ListArticleAttachments(articleID string) ([]ArticleAttachment, error) {
	var result []ArticleAttachment
	path := fmt.Sprintf("/api/articles/%s/attachments?fields=%s&$top=-1", url.PathEscape(articleID), articleAttachmentFields)
	return result, c.Get(path, &result)
}

func (c *Client) AddArticleAttachment(articleID, filePath string) (*ArticleAttachment, error) {
	var result ArticleAttachment
	path := fmt.Sprintf("/api/articles/%s/attachments?fields=%s", url.PathEscape(articleID), articleAttachmentFields)
	return &result, c.Multipart(path, filePath, filepath.Base(filePath), "application/octet-stream", &result)
}

func (c *Client) ListArticleComments(articleID string) ([]Comment, error) {
	var result []Comment
	path := fmt.Sprintf("/api/articles/%s/comments?fields=%s&$top=-1", url.PathEscape(articleID), articleCommentFields)
	return result, c.Get(path, &result)
}

func (c *Client) AddArticleComment(articleID, text, authorID string) (*Comment, error) {
	var result Comment
	body := map[string]any{"text": text}
	if authorID != "" {
		body["author"] = map[string]any{"id": authorID}
	}
	path := fmt.Sprintf("/api/articles/%s/comments?fields=%s", url.PathEscape(articleID), articleCommentFields)
	return &result, c.Post(path, body, &result)
}

func (c *Client) DeleteArticleComment(articleID, commentID string) error {
	return c.Delete(fmt.Sprintf("/api/articles/%s/comments/%s",
		url.PathEscape(articleID), url.PathEscape(commentID)))
}

func (c *Client) UpdateArticleComment(articleID, commentID, text string) (*Comment, error) {
	var result Comment
	path := fmt.Sprintf("/api/articles/%s/comments/%s?fields=%s", url.PathEscape(articleID), url.PathEscape(commentID), articleCommentFields)
	return &result, c.Post(path, map[string]any{"text": text}, &result)
}

func (c *Client) SetParentArticle(articleID, parentID string) error {
	path := fmt.Sprintf("/api/articles/%s", url.PathEscape(articleID))
	return c.Post(path, map[string]any{"parentArticle": map[string]any{"id": parentID}}, nil)
}

func (c *Client) TagArticle(articleID, tagID string) error {
	path := fmt.Sprintf("/api/articles/%s/tags", url.PathEscape(articleID))
	return c.Post(path, map[string]any{"$type": "Tag", "id": tagID}, nil)
}

func (c *Client) UpdateArticleVisibility(articleID string, groupIDs, userIDs []string) error {
	groups := make([]map[string]any, len(groupIDs))
	for i, id := range groupIDs {
		groups[i] = map[string]any{"id": id}
	}
	users := make([]map[string]any, len(userIDs))
	for i, id := range userIDs {
		users[i] = map[string]any{"id": id}
	}
	path := fmt.Sprintf("/api/articles/%s", url.PathEscape(articleID))
	return c.Post(path, map[string]any{
		"visibility": map[string]any{
			"$type":           "LimitedVisibility",
			"permittedGroups": groups,
			"permittedUsers":  users,
		},
	}, nil)
}
