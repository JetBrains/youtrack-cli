package api

import (
	"fmt"
	"net/url"
)

func (c *Client) ListApps() ([]App, error) {
	var result []App
	return result, c.Get("/api/admin/apps?fields=id,name,title,version&$top=-1", &result)
}

func (c *Client) DeleteApp(id string) error {
	return c.Delete(fmt.Sprintf("/api/admin/apps/%s", url.PathEscape(id)))
}

func (c *Client) DeactivateApp(id string) error {
	return c.Post(fmt.Sprintf("/api/admin/apps/%s", url.PathEscape(id)), map[string]any{"globalConfig": map[string]any{"enabled": false}}, nil)
}
