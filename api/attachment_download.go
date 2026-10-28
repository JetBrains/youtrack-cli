package api

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
)

const attachmentDownloadFields = "id,name,url,mimeType,created"

func (c *Client) GetIssueAttachment(issueID, attachmentID string) (*IssueAttachment, error) {
	var result IssueAttachment
	path := fmt.Sprintf(pathIssueAttachment+"?fields="+attachmentDownloadFields, url.PathEscape(issueID), url.PathEscape(attachmentID))
	err := c.Get(path, &result)
	return &result, err
}

// DownloadFile writes the content behind a file URL from an attachment listing to w.
// The URL carries a signature that grants access without a token, so it stays out of every error.
func (c *Client) DownloadFile(fileURL string, w io.Writer) error {
	path, err := c.instancePath(fileURL)
	if err != nil {
		return err
	}
	return c.stream("GET", path, nil, "", w)
}

// instancePath turns a URL that YouTrack returned, absolute or relative to the server root, into a request path below BaseURL.
func (c *Client) instancePath(ref string) (string, error) {
	base, err := url.Parse(c.BaseURL + "/")
	if err != nil {
		return "", fmt.Errorf("parse base URL: %w", err)
	}
	u, err := base.Parse(ref)
	if err != nil {
		return "", errors.New("attachment has an invalid file URL")
	}
	// Compare normalized origins, including case-insensitive hosts and default ports.
	if !strings.EqualFold(u.Scheme, base.Scheme) || !strings.EqualFold(u.Hostname(), base.Hostname()) || defaultPort(u) != defaultPort(base) || u.User != nil {
		return "", errors.New("attachment file URL points outside the configured instance")
	}
	// Reject dot segments even when escaped: intermediaries may decode before
	// normalizing, so returning the original escaped path would cross the boundary.
	parsed, parseErr := url.Parse(ref)
	if parseErr != nil {
		return "", errors.New("attachment has an invalid file URL")
	}
	for _, segment := range strings.Split(parsed.Path, "/") {
		if segment == "." || segment == ".." || strings.Contains(segment, "\\") {
			return "", errors.New("attachment file URL points outside the configured instance")
		}
	}
	prefix := strings.TrimSuffix(base.EscapedPath(), "/")
	if prefix != "" && strings.HasPrefix(u.EscapedPath(), prefix+"/") {
		return strings.TrimPrefix(u.RequestURI(), prefix), nil
	}
	if strings.HasPrefix(u.Path, "/api/") {
		return u.RequestURI(), nil
	}
	return "", errors.New("attachment file URL points outside the configured instance")
}
