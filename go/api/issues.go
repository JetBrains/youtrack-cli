package api

import (
	"fmt"
	"net/url"
	"regexp"
)

const issueFields = "id,idReadable,summary,description,project(id,shortName,name),reporter(id,login,fullName),resolved,created"
const commentFields = "id,text,author(id,login,fullName),created"

// AddIssueRequest is the body for creating an issue.
type AddIssueRequest struct {
	Summary     string   `json:"summary"`
	Description string   `json:"description,omitempty"`
	Project     *Project `json:"project"`
	Reporter    *User    `json:"reporter,omitempty"`
}

// UpdateIssueRequest is the body for updating an issue.
type UpdateIssueRequest struct {
	Summary     string `json:"summary,omitempty"`
	Description string `json:"description,omitempty"`
}

// AddCommentRequest is the body for creating a comment.
type AddCommentRequest struct {
	Text   string `json:"text"`
	Author *User  `json:"author,omitempty"`
}

// ApplyCommandRequest is the body for the /api/commands endpoint.
type ApplyCommandRequest struct {
	Query  string  `json:"query"`
	Issues []Issue `json:"issues"`
}

var userValueRe = regexp.MustCompile(`(?i)\b(assignee|reporter)\s+(\S*[.\-@]\S+)`)

// wrapUserValues braces logins containing dots/hyphens for YouTrack's command parser.
// Example: "assignee alice.qa" → "assignee {alice.qa}"
// Logins that are already braced are left unchanged.
func wrapUserValues(query string) string {
	return userValueRe.ReplaceAllStringFunc(query, func(match string) string {
		sub := userValueRe.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		keyword, login := sub[1], sub[2]
		if len(login) > 0 && login[0] == '{' {
			return match
		}
		return keyword + " {" + login + "}"
	})
}

func (c *Client) GetIssue(id string) (*Issue, error) {
	var result Issue
	path := fmt.Sprintf("/api/issues/%s?fields=%s", url.PathEscape(id), issueFields)
	return &result, c.Get(path, &result)
}

func (c *Client) ListIssues(query string, limit int) ([]Issue, error) {
	var result []Issue
	path := fmt.Sprintf("/api/issues?query=%s&$top=%d&fields=%s",
		url.QueryEscape(query), limit, issueFields)
	return result, c.Get(path, &result)
}

func (c *Client) AddIssue(req AddIssueRequest) (*Issue, error) {
	var result Issue
	return &result, c.Post("/api/issues?fields="+issueFields, req, &result)
}

func (c *Client) UpdateIssue(id string, req UpdateIssueRequest) (*Issue, error) {
	var result Issue
	path := fmt.Sprintf("/api/issues/%s?fields=%s", url.PathEscape(id), issueFields)
	return &result, c.Post(path, req, &result)
}

func (c *Client) DeleteIssue(id string) error {
	return c.Delete(fmt.Sprintf("/api/issues/%s", url.PathEscape(id)))
}

func (c *Client) ListComments(issueID string) ([]Comment, error) {
	var result []Comment
	path := fmt.Sprintf("/api/issues/%s/comments?fields=%s", url.PathEscape(issueID), commentFields)
	return result, c.Get(path, &result)
}

func (c *Client) AddComment(issueID string, req AddCommentRequest) (*Comment, error) {
	var result Comment
	path := fmt.Sprintf("/api/issues/%s/comments?fields=%s", url.PathEscape(issueID), commentFields)
	return &result, c.Post(path, req, &result)
}

func (c *Client) DeleteComment(issueID, commentID string) error {
	return c.Delete(fmt.Sprintf("/api/issues/%s/comments/%s",
		url.PathEscape(issueID), url.PathEscape(commentID)))
}

// ApplyCommand resolves each issue ID to its internal ID then POSTs to /api/commands.
func (c *Client) ApplyCommand(query string, issueIDs []string) error {
	issues := make([]Issue, len(issueIDs))
	for i, id := range issueIDs {
		issue, err := c.GetIssue(id)
		if err != nil {
			return fmt.Errorf("resolve issue %s: %w", id, err)
		}
		issues[i] = Issue{ID: issue.ID}
	}
	return c.Post("/api/commands", ApplyCommandRequest{
		Query:  wrapUserValues(query),
		Issues: issues,
	}, nil)
}
