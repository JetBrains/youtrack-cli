package api

import (
	"fmt"
	"net/url"
)

type AddSavedSearchRequest struct {
	Name  string `json:"name"`
	Query string `json:"query"`
}

const savedSearchFields = "id,name,query,owner(id,login,fullName),readSharingSettings,updateSharingSettings"

func (c *Client) ListSavedSearches() ([]SavedSearch, error) {
	var result []SavedSearch
	err := c.Get(pathSavedQueries+"?fields="+savedSearchFields+"&$top=-1", &result)
	return result, err
}

func (c *Client) GetSavedSearch(id string) (*SavedSearch, error) {
	var result SavedSearch
	path := fmt.Sprintf(pathSavedQuery+"?fields="+savedSearchFields, url.PathEscape(id))
	err := c.Get(path, &result)
	return &result, err
}

func (c *Client) DeleteSavedSearch(id string) error {
	return c.Delete(fmt.Sprintf(pathSavedQuery, url.PathEscape(id)))
}

func (c *Client) AddSavedSearch(req AddSavedSearchRequest) (*SavedSearch, error) {
	var result SavedSearch
	err := c.Post(pathSavedQueries+"?fields="+savedSearchFields, req, &result)
	return &result, err
}

func (c *Client) ShareSavedSearch(id string) error {
	// Fetch the All Users group so we can use its ID.
	var groups []struct {
		ID            string `json:"id"`
		AllUsersGroup bool   `json:"allUsersGroup"`
	}
	if err := c.Get(pathGroups+"?fields=id,allUsersGroup&$top=-1", &groups); err != nil {
		return fmt.Errorf("fetch groups: %w", err)
	}
	var allUsersID string
	for _, g := range groups {
		if g.AllUsersGroup {
			allUsersID = g.ID
			break
		}
	}
	if allUsersID == "" {
		return fmt.Errorf("all users group not found")
	}
	return c.Post(fmt.Sprintf(pathSavedQuery, url.PathEscape(id)), map[string]any{
		"readSharingSettings": map[string]any{
			"$type":           "WatchFolderSharingSettings",
			"permittedGroups": []map[string]any{{"id": allUsersID}},
		},
	}, nil)
}
