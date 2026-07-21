package api

import (
	"fmt"
	"net/url"
	"path/filepath"
)

const articleFields = "id,idReadable,summary,content,reporter(id,login,fullName),project(id,shortName),created,updated,updatedBy(id,login,fullName),parentArticle(id,idReadable),tags(id,name),childArticles(id,idReadable,summary)"
const articleAttachmentFields = "id,name,url,mimeType"
const articleCommentFields = "id,text,author(id,login,fullName),created,updated"

func (c *Client) ListArticles(query string, limit int) ([]Article, error) {
	var result []Article
	path := pathArticles + fmt.Sprintf("?fields=%s&query=%s&$top=%d", articleFields, url.QueryEscape(query), limit)
	err := c.Get(path, &result)
	return result, err
}

func (c *Client) GetArticle(id string) (*Article, error) {
	var result Article
	path := fmt.Sprintf(pathArticle+"?fields=%s", url.PathEscape(id), articleFields)
	err := c.Get(path, &result)
	return &result, err
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
	err := c.Post(pathArticles+"?fields="+articleFields, body, &result)
	return &result, err
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
	path := fmt.Sprintf(pathArticle+"?fields=%s", url.PathEscape(id), articleFields)
	err := c.Post(path, body, &result)
	return &result, err
}

func (c *Client) DeleteArticle(id string) error {
	return c.Delete(fmt.Sprintf(pathArticle, url.PathEscape(id)))
}

func (c *Client) ListArticleAttachments(articleID string) ([]ArticleAttachment, error) {
	var result []ArticleAttachment
	path := fmt.Sprintf(pathArticleAttachments+"?fields=%s&$top=-1", url.PathEscape(articleID), articleAttachmentFields)
	err := c.Get(path, &result)
	return result, err
}

func (c *Client) AddArticleAttachment(articleID, filePath string) (*ArticleAttachment, error) {
	var results []ArticleAttachment
	path := fmt.Sprintf(pathArticleAttachments+"?fields=%s", url.PathEscape(articleID), articleAttachmentFields)
	if err := c.Multipart(path, filePath, filepath.Base(filePath), "application/octet-stream", &results); err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no attachment returned from API")
	}
	return &results[0], nil
}

func (c *Client) ListArticleComments(articleID string) ([]Comment, error) {
	var result []Comment
	path := fmt.Sprintf(pathArticleComments+"?fields=%s&$top=-1", url.PathEscape(articleID), articleCommentFields)
	err := c.Get(path, &result)
	return result, err
}

func (c *Client) AddArticleComment(articleID, text string) (*Comment, error) {
	var result Comment
	path := fmt.Sprintf(pathArticleComments+"?fields=%s", url.PathEscape(articleID), articleCommentFields)
	err := c.Post(path, map[string]any{"text": text}, &result)
	return &result, err
}

func (c *Client) DeleteArticleComment(articleID, commentID string) error {
	return c.Delete(fmt.Sprintf(pathArticleComment,
		url.PathEscape(articleID), url.PathEscape(commentID)))
}

func (c *Client) UpdateArticleComment(articleID, commentID, text string) (*Comment, error) {
	var result Comment
	path := fmt.Sprintf(pathArticleComment+"?fields=%s", url.PathEscape(articleID), url.PathEscape(commentID), articleCommentFields)
	err := c.Post(path, map[string]any{"text": text}, &result)
	return &result, err
}

func (c *Client) GetArticleComment(articleID, commentID string) (*Comment, error) {
	var result Comment
	path := fmt.Sprintf(pathArticleComment+"?fields=%s", url.PathEscape(articleID), url.PathEscape(commentID), articleCommentFields)
	err := c.Get(path, &result)
	return &result, err
}

func (c *Client) PinArticleComment(articleID, commentID string, pinned bool) error {
	return c.Post(fmt.Sprintf(pathArticleComment, url.PathEscape(articleID), url.PathEscape(commentID)),
		map[string]any{"pinned": pinned}, nil)
}

func (c *Client) UpdateArticleCommentVisibility(articleID, commentID string, groupIDs, userIDs []string) error {
	groups := make([]map[string]any, len(groupIDs))
	for i, id := range groupIDs {
		groups[i] = map[string]any{"id": id}
	}
	users := make([]map[string]any, len(userIDs))
	for i, id := range userIDs {
		users[i] = map[string]any{"id": id}
	}
	body := map[string]any{
		"visibility": map[string]any{
			"$type":           "LimitedVisibility",
			"permittedGroups": groups,
			"permittedUsers":  users,
		},
	}
	return c.Post(fmt.Sprintf(pathArticleComment, url.PathEscape(articleID), url.PathEscape(commentID)), body, nil)
}

func (c *Client) SetParentArticle(articleID, parentID string) error {
	path := fmt.Sprintf(pathArticle, url.PathEscape(articleID))
	return c.Post(path, map[string]any{"parentArticle": map[string]any{"id": parentID}}, nil)
}

func (c *Client) TagArticle(articleID, tagID string) error {
	path := fmt.Sprintf(pathArticleTags, url.PathEscape(articleID))
	return c.Post(path, map[string]any{"$type": "Tag", "id": tagID}, nil)
}

func (c *Client) ListArticleChildren(articleID string) ([]Article, error) {
	article, err := c.GetArticle(articleID)
	if err != nil {
		return nil, err
	}
	return article.ChildArticles, nil
}

func (c *Client) AddArticleChild(parentID, childID string) error {
	path := fmt.Sprintf(pathArticle, url.PathEscape(childID))
	return c.Post(path, map[string]any{"parentArticle": map[string]any{"id": parentID}}, nil)
}

func (c *Client) GetArticleParent(articleID string) (*Article, error) {
	article, err := c.GetArticle(articleID)
	if err != nil {
		return nil, err
	}
	return article.ParentArticle, nil
}

func (c *Client) ListArticleCommentReactions(articleID, commentID string) ([]Reaction, error) {
	var result []Reaction
	path := fmt.Sprintf(pathArticleCommentReactions+"?fields=id,reaction,author(id,login,fullName)&$top=-1",
		url.PathEscape(articleID), url.PathEscape(commentID))
	err := c.Get(path, &result)
	return result, err
}

func (c *Client) AddArticleCommentReaction(articleID, commentID, reaction string) error {
	path := fmt.Sprintf(pathArticleCommentReactions, url.PathEscape(articleID), url.PathEscape(commentID))
	return c.Post(path, map[string]any{"reaction": reaction}, nil)
}

func (c *Client) DeleteArticleCommentReaction(articleID, commentID, reactionID string) error {
	return c.Delete(fmt.Sprintf(pathArticleCommentReaction,
		url.PathEscape(articleID), url.PathEscape(commentID), url.PathEscape(reactionID)))
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
	path := fmt.Sprintf(pathArticle, url.PathEscape(articleID))
	return c.Post(path, map[string]any{
		"visibility": map[string]any{
			"$type":           "LimitedVisibility",
			"permittedGroups": groups,
			"permittedUsers":  users,
		},
	}, nil)
}
