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
	return &result, c.Post(pathSavedQueries+"?fields=id,name,query", req, &result)
}

func (c *Client) ShareSavedSearch(id string) error {
	return c.Post(fmt.Sprintf(pathSavedQuery, url.PathEscape(id)), map[string]any{"isShareable": true}, nil)
}
