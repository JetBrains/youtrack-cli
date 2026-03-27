package api

import (
	"fmt"
	"net/url"
)

const projectFields = "id,shortName,name,leader(id,login,fullName)"

// AddProjectRequest is the body for creating a project.
type AddProjectRequest struct {
	ShortName string `json:"shortName"`
	Name      string `json:"name"`
	Leader    *User  `json:"leader"`
}

func (c *Client) ListProjects() ([]Project, error) {
	var result []Project
	return result, c.Get("/api/admin/projects?fields="+projectFields, &result)
}

// GetProject finds a project by its short name (case-sensitive).
func (c *Client) GetProject(shortName string) (*Project, error) {
	projects, err := c.ListProjects()
	if err != nil {
		return nil, err
	}
	for i := range projects {
		if projects[i].ShortName == shortName {
			return &projects[i], nil
		}
	}
	return nil, fmt.Errorf("project '%s' not found", shortName)
}

func (c *Client) AddProject(req AddProjectRequest) (*Project, error) {
	var result Project
	return &result, c.Post("/api/admin/projects?fields="+projectFields, req, &result)
}

func (c *Client) DeleteProject(shortName string) error {
	p, err := c.GetProject(shortName)
	if err != nil {
		return err
	}
	return c.Delete(fmt.Sprintf("/api/admin/projects/%s", url.PathEscape(p.ID)))
}
