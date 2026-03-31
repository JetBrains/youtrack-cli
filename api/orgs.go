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
	return result, c.Get(pathAdminOrganizations+"?fields=id,name&$top=-1", &result)
}

func (c *Client) AddOrganization(req AddOrganizationRequest) (*Organization, error) {
	var result Organization
	return &result, c.Post(pathAdminOrganizations+"?fields=id,name", req, &result)
}

func (c *Client) DeleteOrganization(id string) error {
	return c.Delete(fmt.Sprintf(pathAdminOrganization, url.PathEscape(id)))
}
