package api

import (
	"fmt"
	"net/url"
)

func (c *Client) ListGroups(query string) ([]Group, error) {
	var result []Group
	path := fmt.Sprintf("/api/groups?fields=id,name&query=%s&$top=-1", url.QueryEscape(query))
	return result, c.Get(path, &result)
}

func (c *Client) GetGroup(name string) (*Group, error) {
	groups, err := c.ListGroups(name)
	if err != nil {
		return nil, err
	}
	for i := range groups {
		if groups[i].Name == name {
			return &groups[i], nil
		}
	}
	return nil, fmt.Errorf("group '%s' not found", name)
}

func (c *Client) AddGroup(name string) (*Group, error) {
	var result Group
	return &result, c.Post("/api/groups?fields=id,name", map[string]any{"name": name}, &result)
}

func (c *Client) DeleteGroup(id string) error {
	return c.Delete(fmt.Sprintf("/api/groups/%s", url.PathEscape(id)))
}
