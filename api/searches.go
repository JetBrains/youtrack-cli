package api

import (
	"fmt"
	"net/url"
	"strconv"
)

type AddSavedSearchRequest struct {
	Name  string `json:"name"`
	Query string `json:"query"`
}

const savedSearchFields = "id,name,query,owner(id,login,fullName),readSharingSettings(permittedGroups(id,name)),updateSharingSettings"

func (c *Client) ListSavedSearches() ([]SavedSearch, error) {
	const pageSize = 500
	var all []SavedSearch
	for skip := 0; ; skip += pageSize {
		var page []SavedSearch
		if err := c.Get(pathSavedQueries+"?fields="+savedSearchFields+"&$top="+strconv.Itoa(pageSize)+"&$skip="+strconv.Itoa(skip), &page); err != nil {
			return nil, err
		}
		all = append(all, page...)
		if len(page) < pageSize {
			break
		}
	}
	return all, nil
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
	allUsersID, err := c.allUsersGroupID()
	if err != nil {
		return err
	}
	return c.Post(fmt.Sprintf(pathSavedQuery, url.PathEscape(id)), map[string]any{
		"readSharingSettings": map[string]any{
			"$type":           "WatchFolderSharingSettings",
			"permittedGroups": []map[string]any{{"id": allUsersID}},
		},
	}, nil)
}

func (c *Client) allUsersGroupID() (string, error) {
	var groups []struct {
		ID            string `json:"id"`
		AllUsersGroup bool   `json:"allUsersGroup"`
	}
	if err := c.Get(pathGroups+"?fields=id,allUsersGroup&$top=500", &groups); err != nil {
		return "", fmt.Errorf("fetch groups: %w", err)
	}
	for _, g := range groups {
		if g.AllUsersGroup {
			return g.ID, nil
		}
	}
	return "", fmt.Errorf("all users group not found")
}
