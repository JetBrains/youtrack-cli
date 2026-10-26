package api

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

const tagFields = "id,name,owner(id,login,fullName),color,untagOnResolve"

type AddTagRequest struct {
	Name           string `json:"name"`
	UntagOnResolve bool   `json:"untagOnResolve,omitempty"`
}

// ListTags returns up to limit tags visible to the current user; a non-empty query keeps tags whose name words start with it.
func (c *Client) ListTags(query string, limit int) ([]Tag, error) {
	var result []Tag
	path := fmt.Sprintf(pathTags+"?fields=%s&$top=%d", tagFields, limit)
	if query != "" {
		path += "&query=" + url.QueryEscape(query)
	}
	err := c.Get(path, &result)
	return result, err
}

func (c *Client) AddTag(req AddTagRequest) (*Tag, error) {
	var result Tag
	err := c.Post(pathTags+"?fields="+tagFields, req, &result)
	return &result, err
}

func (c *Client) DeleteTag(id string) error {
	return c.Delete(fmt.Sprintf(pathTag, url.PathEscape(id)))
}

func (c *Client) ListTagIssues(tagID string, limit int) ([]Issue, error) {
	var result []Issue
	path := fmt.Sprintf(pathTagIssues+"?fields=%s&$top=%d", url.PathEscape(tagID), issueFields, limit)
	err := c.Get(path, &result)
	return result, err
}

var tagIDPattern = regexp.MustCompile(`^\d+-\d+$`)

// IsTagID reports whether ref has the shape of a tag database ID such as 6-12.
func IsTagID(ref string) bool {
	return tagIDPattern.MatchString(ref)
}

// ResolveTagID returns ref unchanged when it looks like a tag ID; otherwise it finds the single visible tag named exactly ref.
func (c *Client) ResolveTagID(ref string) (string, error) {
	if ref == "" {
		return "", fmt.Errorf("tag name or ID must not be empty")
	}
	if IsTagID(ref) {
		return ref, nil
	}
	const pageSize = 500
	var candidates []Tag
	for skip := 0; ; skip += pageSize {
		var page []Tag
		path := fmt.Sprintf(pathTags+"?fields=%s&query=%s&$top=%d&$skip=%d", tagFields, url.QueryEscape(ref), pageSize, skip)
		if err := c.Get(path, &page); err != nil {
			return "", err
		}
		candidates = append(candidates, page...)
		if len(page) < pageSize {
			break
		}
	}
	return pickTagByName(candidates, ref, "visible to you")
}

// ResolveIssueTagID returns ref unchanged when it looks like a tag ID; otherwise it finds the single tag on the issue named exactly ref.
func (c *Client) ResolveIssueTagID(issueID, ref string) (string, error) {
	if ref == "" {
		return "", fmt.Errorf("tag name or ID must not be empty")
	}
	if IsTagID(ref) {
		return ref, nil
	}
	tags, err := c.ListIssueTags(issueID)
	if err != nil {
		return "", err
	}
	return pickTagByName(tags, ref, "on issue "+issueID)
}

func pickTagByName(tags []Tag, name, scope string) (string, error) {
	var matches []Tag
	for _, t := range tags {
		if t.Name == name {
			matches = append(matches, t)
		}
	}
	switch len(matches) {
	case 0:
		var similar []string
		for _, t := range tags {
			if strings.EqualFold(t.Name, name) {
				similar = append(similar, fmt.Sprintf("%q (%s)", t.Name, t.ID))
			}
		}
		if len(similar) > 0 {
			return "", fmt.Errorf("no tag named exactly %q %s; names differing only in case: %s", name, scope, strings.Join(similar, ", "))
		}
		return "", fmt.Errorf("no tag named %q %s; pass a tag ID or create the tag with 'youtrack tag create'", name, scope)
	case 1:
		return matches[0].ID, nil
	}
	described := make([]string, 0, len(matches))
	for _, t := range matches {
		owner := "unknown owner"
		if t.Owner != nil && t.Owner.Login != "" {
			owner = "owner " + t.Owner.Login
		}
		described = append(described, fmt.Sprintf("%s (%s)", t.ID, owner))
	}
	return "", fmt.Errorf("%d tags named %q %s: %s; pass a tag ID instead", len(matches), name, scope, strings.Join(described, ", "))
}
