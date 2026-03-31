package api

import (
	"fmt"
	"net/url"
)

func (c *Client) ListDashboards() ([]Dashboard, error) {
	var result []Dashboard
	return result, c.Get(pathDashboards+"?fields=id,name&$top=-1", &result)
}

func (c *Client) AddDashboard(name string) (*Dashboard, error) {
	var result Dashboard
	return &result, c.Post(pathDashboards+"?fields=id,name", map[string]any{"name": name}, &result)
}

func (c *Client) DeleteDashboard(id string) error {
	return c.Delete(fmt.Sprintf(pathDashboard, url.PathEscape(id)))
}

func (c *Client) UpdateDashboard(id, name string) (*Dashboard, error) {
	var result Dashboard
	path := fmt.Sprintf(pathDashboard+"?fields=id,name", url.PathEscape(id))
	return &result, c.Post(path, map[string]any{"name": name}, &result)
}
