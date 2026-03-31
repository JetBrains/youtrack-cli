package api

import (
	"fmt"
	"net/url"
)

type AddTagRequest struct {
	Name           string `json:"name"`
	Owner          *User  `json:"owner,omitempty"`
	IsShareable    bool   `json:"isShareable,omitempty"`
	UntagOnResolve bool   `json:"untagOnResolve,omitempty"`
	Pinned         bool   `json:"pinned,omitempty"`
}

func (c *Client) ListTags() ([]Tag, error) {
	var result []Tag
	return result, c.Get(pathTags+"?fields=id,name,owner(id,login)&$top=-1", &result)
}

func (c *Client) AddTag(req AddTagRequest) (*Tag, error) {
	var result Tag
	return &result, c.Post(pathTags+"?fields=id,name,owner(id,login)", req, &result)
}

func (c *Client) DeleteTag(id string) error {
	return c.Delete(fmt.Sprintf(pathTag, url.PathEscape(id)))
}
