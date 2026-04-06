package api

import (
	"fmt"
	"net/url"
)

type AddOrganizationRequest struct {
	Name string `json:"name"`
}

func (c *Client) ListOrganizations() ([]Organization, error) {
	var result []Organization
	err := c.Get(pathAdminOrganizations+"?fields=id,name&$top=-1", &result)
	return result, err
}

func (c *Client) AddOrganization(req AddOrganizationRequest) (*Organization, error) {
	var result Organization
	err := c.Post(pathAdminOrganizations+"?fields=id,name", req, &result)
	return &result, err
}

func (c *Client) DeleteOrganization(id string) error {
	return c.Delete(fmt.Sprintf(pathAdminOrganization, url.PathEscape(id)))
}
