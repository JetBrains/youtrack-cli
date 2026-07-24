package api

import (
	"fmt"
	"net/url"
)

func (c *Client) ListDashboards() ([]Dashboard, error) {
	var result []Dashboard
	err := c.Get(pathDashboards+"?fields=id,name&$top=500", &result)
	return result, err
}

func (c *Client) AddDashboard(name string) (*Dashboard, error) {
	var result Dashboard
	err := c.Post(pathDashboards+"?fields=id,name", map[string]any{"name": name}, &result)
	return &result, err
}

func (c *Client) DeleteDashboard(id string) error {
	return c.Delete(fmt.Sprintf(pathDashboard, url.PathEscape(id)))
}

func (c *Client) UpdateDashboard(id, name string) (*Dashboard, error) {
	var result Dashboard
	p := fmt.Sprintf(pathDashboard+"?fields=id,name", url.PathEscape(id))
	err := c.Post(p, map[string]any{"name": name}, &result)
	return &result, err
}
