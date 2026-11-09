package api

import (
	"fmt"
	"net/url"
)

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
