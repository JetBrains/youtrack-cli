package api

import (
	"fmt"
	"net/url"
)

type AddTagRequest struct {
	Name           string `json:"name"`
	UntagOnResolve bool   `json:"untagOnResolve,omitempty"`
}

func (c *Client) ListTags() ([]Tag, error) {
	var result []Tag
	err := c.Get(pathTags+"?fields=id,name,owner(id,login)&$top=-1", &result)
	return result, err
}

func (c *Client) AddTag(req AddTagRequest) (*Tag, error) {
	var result Tag
	err := c.Post(pathTags+"?fields=id,name,owner(id,login)", req, &result)
	return &result, err
}

func (c *Client) DeleteTag(id string) error {
	return c.Delete(fmt.Sprintf(pathTag, url.PathEscape(id)))
}
