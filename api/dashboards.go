package api

import (
	"fmt"
	"net/url"
)

func (c *Client) ListDashboards() ([]Dashboard, error) {
	var result []Dashboard
	return result, c.Get("/api/dashboards?fields=id,name&$top=-1", &result)
}

func (c *Client) AddDashboard(name string) (*Dashboard, error) {
	var result Dashboard
	return &result, c.Post("/api/dashboards?fields=id,name", map[string]any{"name": name}, &result)
}

func (c *Client) DeleteDashboard(id string) error {
	return c.Delete(fmt.Sprintf("/api/dashboards/%s", url.PathEscape(id)))
}

func (c *Client) UpdateDashboard(id, name string) (*Dashboard, error) {
	var result Dashboard
	path := fmt.Sprintf("/api/dashboards/%s?fields=id,name", url.PathEscape(id))
	return &result, c.Post(path, map[string]any{"name": name}, &result)
}
