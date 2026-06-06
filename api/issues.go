package api

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
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

// WrapUserValues braces logins containing dots/hyphens for YouTrack's command parser.
// Example: "assignee alice.qa" → "assignee {alice.qa}"
// Logins that are already braced are left unchanged.
func WrapUserValues(query string) string {
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
	path := fmt.Sprintf(pathIssue+"?fields=%s", url.PathEscape(id), issueFields)
	err := c.Get(path, &result)
	return &result, err
}

func (c *Client) ListIssues(query string, limit int) ([]Issue, error) {
	var result []Issue
	path := pathIssues + fmt.Sprintf("?query=%s&$top=%d&fields=%s",
		url.QueryEscape(query), limit, issueFields)
	err := c.Get(path, &result)
	return result, err
}

func (c *Client) AddIssue(req AddIssueRequest) (*Issue, error) {
	var result Issue
	err := c.Post(pathIssues+"?fields="+issueFields, req, &result)
	return &result, err
}

func (c *Client) UpdateIssue(id string, req UpdateIssueRequest) (*Issue, error) {
	var result Issue
	path := fmt.Sprintf(pathIssue+"?fields=%s", url.PathEscape(id), issueFields)
	err := c.Post(path, req, &result)
	return &result, err
}

func (c *Client) DeleteIssue(id string) error {
	return c.Delete(fmt.Sprintf(pathIssue, url.PathEscape(id)))
}

func (c *Client) ListComments(issueID string) ([]Comment, error) {
	var result []Comment
	path := fmt.Sprintf(pathIssueComments+"?fields=%s", url.PathEscape(issueID), commentFields)
	err := c.Get(path, &result)
	return result, err
}

func (c *Client) AddComment(issueID string, req AddCommentRequest) (*Comment, error) {
	var result Comment
	path := fmt.Sprintf(pathIssueComments+"?fields=%s", url.PathEscape(issueID), commentFields)
	err := c.Post(path, req, &result)
	return &result, err
}

func (c *Client) DeleteComment(issueID, commentID string) error {
	return c.Delete(fmt.Sprintf(pathIssueComment,
		url.PathEscape(issueID), url.PathEscape(commentID)))
}

func (c *Client) GetIssueCount(query string) (int, error) {
	var result struct {
		Count int `json:"count"`
	}
	err := c.Post(pathIssuesGetterCount+"?fields=count", map[string]any{"query": query}, &result)
	return result.Count, err
}

func (c *Client) GetIssueLinkTypes() ([]LinkType, error) {
	var result []LinkType
	err := c.Get(pathIssueLinkTypes+"?fields=id,name,sourceToTarget,targetToSource&$top=-1", &result)
	return result, err
}

func (c *Client) AddIssueLinkType(name, sourceToTarget, targetToSource string) (*LinkType, error) {
	var result LinkType
	err := c.Post(pathIssueLinkTypes+"?fields=id,name,sourceToTarget,targetToSource", map[string]any{
		"name":           name,
		"sourceToTarget": sourceToTarget,
		"targetToSource": targetToSource,
		"directed":       true,
		"aggregation":    false,
	}, &result)
	return &result, err
}

func (c *Client) GetLinkedIssues(issueID string) ([]IssueLink, error) {
	var result []IssueLink
	path := fmt.Sprintf(pathIssueLinks+"?fields=id,direction,linkType(id,name,sourceToTarget,targetToSource),issues(id,idReadable,summary)&$top=-1", url.PathEscape(issueID))
	err := c.Get(path, &result)
	return result, err
}

func (c *Client) LinkIssues(issueID, targetID, linkTypeID string) error {
	path := fmt.Sprintf(pathIssueLinks, url.PathEscape(issueID))
	return c.Post(path, map[string]any{
		"linkType": map[string]any{"id": linkTypeID},
		"issues":   []map[string]any{{"id": targetID}},
	}, nil)
}

func (c *Client) RemoveIssueLink(issueID, linkID string) error {
	return c.Delete(fmt.Sprintf(pathIssueLink, url.PathEscape(issueID), url.PathEscape(linkID)))
}

func (c *Client) TagIssue(issueID, tagID string) error {
	path := fmt.Sprintf(pathIssueTags, url.PathEscape(issueID))
	return c.Post(path, map[string]any{"$type": "Tag", "id": tagID}, nil)
}

func (c *Client) GetIssueAttachments(issueID string) ([]IssueAttachment, error) {
	var result []IssueAttachment
	path := fmt.Sprintf(pathIssueAttachments+"?fields=id,name,url,mimeType,created&$top=-1", url.PathEscape(issueID))
	err := c.Get(path, &result)
	return result, err
}

func (c *Client) AddIssueAttachment(issueID, filePath string) (*IssueAttachment, error) {
	var result IssueAttachment
	path := fmt.Sprintf(pathIssueAttachments+"?fields=id,name,url,mimeType,created", url.PathEscape(issueID))
	err := c.Multipart(path, filePath, filepath.Base(filePath), "application/octet-stream", &result)
	return &result, err
}

func (c *Client) DeleteIssueAttachment(issueID, attachmentID string) error {
	return c.Delete(fmt.Sprintf(pathIssueAttachment,
		url.PathEscape(issueID), url.PathEscape(attachmentID)))
}

func (c *Client) AddWorkItem(issueID string, minutes int, text, typeID string) (*WorkItem, error) {
	var result WorkItem
	body := map[string]any{
		"duration": map[string]any{"minutes": minutes},
		"date":     time.Now().UnixMilli(),
	}
	if text != "" {
		body["text"] = text
	}
	if typeID != "" {
		body["type"] = map[string]any{"id": typeID}
	}
	path := fmt.Sprintf(pathIssueWorkItems+"?fields=id,text,date,duration(minutes),author(id,login)", url.PathEscape(issueID))
	err := c.Post(path, body, &result)
	return &result, err
}

func (c *Client) GetCustomFieldValue(issueID, fieldName string) (*IssueField, error) {
	var fields []IssueField
	path := fmt.Sprintf(pathIssueFields+"?fields=id,name,value(name,presentation,text,login,minutes)&$top=-1", url.PathEscape(issueID))
	if err := c.Get(path, &fields); err != nil {
		return nil, err
	}
	for i := range fields {
		if fields[i].Name == fieldName {
			return &fields[i], nil
		}
	}
	return nil, fmt.Errorf("field '%s' not found on issue %s", fieldName, issueID)
}

func (c *Client) SetCustomField(issueID, fieldName, fieldValue, fieldType string) error {
	var fields []IssueField
	path := fmt.Sprintf(pathIssueFields+"?fields=id,name&$top=-1", url.PathEscape(issueID))
	if err := c.Get(path, &fields); err != nil {
		return err
	}
	var fieldID string
	for _, f := range fields {
		if f.Name == fieldName {
			fieldID = f.ID
			break
		}
	}
	if fieldID == "" {
		return fmt.Errorf("field '%s' not found on issue %s", fieldName, issueID)
	}
	var body map[string]any
	switch fieldType {
	case "period":
		body = map[string]any{"value": map[string]any{"presentation": fieldValue}}
	case "text":
		body = map[string]any{"value": map[string]any{"text": fieldValue}}
	case "user":
		var users []User
		if err := c.Get(pathUsers+fmt.Sprintf("?query=%s&fields=id,login&$top=10", url.QueryEscape(fieldValue)), &users); err != nil {
			return err
		}
		var userID string
		for _, u := range users {
			if u.Login == fieldValue {
				userID = u.ID
				break
			}
		}
		if userID == "" {
			return fmt.Errorf("user '%s' not found", fieldValue)
		}
		body = map[string]any{"value": map[string]any{"id": userID}}
	case "int":
		n, err := strconv.Atoi(fieldValue)
		if err != nil {
			return fmt.Errorf("invalid integer value '%s': %w", fieldValue, err)
		}
		body = map[string]any{"value": n}
	case "string":
		body = map[string]any{"value": fieldValue}
	default:
		body = map[string]any{"value": map[string]any{"name": fieldValue}}
	}
	return c.Post(fmt.Sprintf(pathIssueField, url.PathEscape(issueID), url.PathEscape(fieldID)), body, nil)
}

func (c *Client) UpdateComment(issueID, commentID, text string) (*Comment, error) {
	var result Comment
	path := fmt.Sprintf(pathIssueComment+"?fields=%s", url.PathEscape(issueID), url.PathEscape(commentID), commentFields)
	err := c.Post(path, map[string]any{"text": text}, &result)
	return &result, err
}

func (c *Client) UpdateCommentVisibility(issueID, commentID string, groupIDs, userIDs []string) error {
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
	return c.Post(fmt.Sprintf(pathIssueComment, url.PathEscape(issueID), url.PathEscape(commentID)), body, nil)
}

func (c *Client) PinComment(issueID, commentID string, pinned bool) error {
	return c.Post(fmt.Sprintf(pathIssueComment, url.PathEscape(issueID), url.PathEscape(commentID)),
		map[string]any{"pinned": pinned}, nil)
}

func (c *Client) UpdateIssueVisibility(issueID string, groupIDs, userIDs []string) error {
	groups := make([]map[string]any, len(groupIDs))
	for i, id := range groupIDs {
		groups[i] = map[string]any{"id": id}
	}
	users := make([]map[string]any, len(userIDs))
	for i, id := range userIDs {
		users[i] = map[string]any{"id": id}
	}
	return c.Post(fmt.Sprintf(pathIssue, url.PathEscape(issueID)), map[string]any{
		"visibility": map[string]any{
			"$type":           "LimitedVisibility",
			"permittedGroups": groups,
			"permittedUsers":  users,
		},
	}, nil)
}

func (c *Client) GetCommandAssist(query string, issueIDs []string) (map[string]any, error) {
	issues := make([]map[string]any, len(issueIDs))
	for i, id := range issueIDs {
		issue, err := c.GetIssue(id)
		if err != nil {
			return nil, fmt.Errorf("resolve issue %s: %w", id, err)
		}
		issues[i] = map[string]any{"id": issue.ID}
	}
	var result map[string]any
	err := c.Post(pathCommandsAssist+"?fields=commands(error,description),query",
		map[string]any{"query": query, "issues": issues}, &result)
	return result, err
}

func (c *Client) AddReaction(issueID, commentID, reaction string) error {
	path := fmt.Sprintf(pathIssueCommentReactions, url.PathEscape(issueID), url.PathEscape(commentID))
	return c.Post(path, map[string]any{"reaction": reaction}, nil)
}

func (c *Client) DeleteReaction(issueID, commentID, reactionID string) error {
	return c.Delete(fmt.Sprintf(pathIssueCommentReaction,
		url.PathEscape(issueID), url.PathEscape(commentID), url.PathEscape(reactionID)))
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
	return c.Post(pathCommands, ApplyCommandRequest{
		Query:  WrapUserValues(query),
		Issues: issues,
	}, nil)
}

func (c *Client) CreateIssueDraft(projectShortName, summary string) (*IssueDraft, error) {
	project, err := c.GetProject(projectShortName)
	if err != nil {
		return nil, err
	}
	var result IssueDraft
	path := fmt.Sprintf(pathIssueDrafts+"?fields=id,summary", "me")
	err = c.Post(path, map[string]any{
		"project": map[string]any{"id": project.ID},
		"summary": summary,
	}, &result)
	return &result, err
}

func (c *Client) WatchIssue(id string) error {
	issue, err := c.GetIssue(id)
	if err != nil {
		return err
	}
	path := fmt.Sprintf(pathIssueWatchers, url.PathEscape(issue.ID))
	return c.Post(path, map[string]any{"hasWatchedIssue": true}, nil)
}

func (c *Client) UnwatchIssue(id string) error {
	issue, err := c.GetIssue(id)
	if err != nil {
		return err
	}
	path := fmt.Sprintf(pathIssueWatchers, url.PathEscape(issue.ID))
	return c.Post(path, map[string]any{"hasWatchedIssue": false}, nil)
}

func (c *Client) VoteIssue(id string) error {
	issue, err := c.GetIssue(id)
	if err != nil {
		return err
	}
	path := fmt.Sprintf(pathIssueVoters, url.PathEscape(issue.ID))
	return c.Post(path, map[string]any{"hasVote": true}, nil)
}

func (c *Client) UnvoteIssue(id string) error {
	issue, err := c.GetIssue(id)
	if err != nil {
		return err
	}
	path := fmt.Sprintf(pathIssueVoters, url.PathEscape(issue.ID))
	return c.Post(path, map[string]any{"hasVote": false}, nil)
}

func (c *Client) ListRecentComments(issueID string, top int) ([]Comment, error) {
	var result []Comment
	path := fmt.Sprintf(pathIssueComments+"?fields=%s&$top=%d", url.PathEscape(issueID), commentFields, top)
	err := c.Get(path, &result)
	return result, err
}
