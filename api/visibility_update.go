package api

import (
	"fmt"
	"net/url"
)

func (c *Client) setVisibility(path string, v VisibilityChange) (*Visibility, error) {
	if err := c.Post(path, map[string]any{"visibility": v}, nil); err != nil {
		return nil, err
	}
	var result struct {
		Visibility *Visibility `json:"visibility"`
	}
	if err := c.Get(path+"?fields="+visibilityFields, &result); err != nil {
		return nil, fmt.Errorf("re-read visibility: %w", err)
	}
	return result.Visibility, nil
}

// SetIssueVisibility writes the issue visibility and returns it as re-read from the server,
// together with the reporter's ID, whom YouTrack may add to the permitted users.
func (c *Client) SetIssueVisibility(issueID string, v VisibilityChange) (got *Visibility, reporterID string, err error) {
	path := fmt.Sprintf(pathIssue, url.PathEscape(issueID))
	if err := c.Post(path, map[string]any{"visibility": v}, nil); err != nil {
		return nil, "", err
	}
	var result struct {
		Visibility *Visibility `json:"visibility"`
		Reporter   *User       `json:"reporter"`
	}
	if err := c.Get(path+"?fields=reporter(id),"+visibilityFields, &result); err != nil {
		return nil, "", fmt.Errorf("re-read visibility: %w", err)
	}
	if result.Reporter != nil {
		reporterID = result.Reporter.ID
	}
	return result.Visibility, reporterID, nil
}

// SetCommentVisibility writes the comment visibility and returns it as re-read from the server.
func (c *Client) SetCommentVisibility(issueID, commentID string, v VisibilityChange) (*Visibility, error) {
	return c.setVisibility(fmt.Sprintf(pathIssueComment, url.PathEscape(issueID), url.PathEscape(commentID)), v)
}

// SetAttachmentVisibility writes the attachment visibility and returns it as re-read from the server.
func (c *Client) SetAttachmentVisibility(issueID, attachmentID string, v VisibilityChange) (*Visibility, error) {
	return c.setVisibility(fmt.Sprintf(pathIssueAttachment, url.PathEscape(issueID), url.PathEscape(attachmentID)), v)
}
