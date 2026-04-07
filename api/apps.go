package api

import (
	"fmt"
	"net/url"
)

func (c *Client) ListApps() ([]App, error) {
	var result []App
	err := c.Get(pathAdminApps+"?fields=id,name,title,version&$top=-1", &result)
	return result, err
}

func (c *Client) DeleteApp(id string) error {
	return c.Delete(fmt.Sprintf(pathAdminApp, url.PathEscape(id)))
}

func (c *Client) DeactivateApp(id string) error {
	return c.Post(fmt.Sprintf(pathAdminApp, url.PathEscape(id)), map[string]any{"globalConfig": map[string]any{"enabled": false}}, nil)
}
