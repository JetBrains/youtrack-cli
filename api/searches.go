package api

import (
	"fmt"
	"net/url"
)

type AddSavedSearchRequest struct {
	Name  string `json:"name"`
	Query string `json:"query"`
}

func (c *Client) AddSavedSearch(req AddSavedSearchRequest) (*SavedSearch, error) {
	var result SavedSearch
	err := c.Post(pathSavedQueries+"?fields=id,name,query", req, &result)
	return &result, err
}

func (c *Client) ShareSavedSearch(id string) error {
	return c.Post(fmt.Sprintf(pathSavedQuery, url.PathEscape(id)), map[string]any{
		"readSharingSettings": map[string]any{"$type": "AllUsersVisibility"},
	}, nil)
}
