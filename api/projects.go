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
	return result, c.Get("/api/admin/projects?fields="+projectFields+"&$top=-1", &result)
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

func (c *Client) ArchiveProject(shortName string) error {
	p, err := c.GetProject(shortName)
	if err != nil {
		return err
	}
	return c.Post(fmt.Sprintf("/api/admin/projects/%s", url.PathEscape(p.ID)), map[string]any{"archived": true}, nil)
}

func (c *Client) GetProjectTemplates() ([]ProjectTemplate, error) {
	var result []ProjectTemplate
	return result, c.Get("/api/admin/projectTemplates?fields=id,name,shortName&$top=-1", &result)
}

func (c *Client) GetProjectCustomFields(shortName string) ([]ProjectCustomField, error) {
	p, err := c.GetProject(shortName)
	if err != nil {
		return nil, err
	}
	var result []ProjectCustomField
	path := fmt.Sprintf("/api/admin/projects/%s/fields?fields=id,field(id,name),bundle(id,name)&$top=-1", url.PathEscape(p.ID))
	return result, c.Get(path, &result)
}

func (c *Client) AddProjectCustomField(shortName, fieldID string) error {
	p, err := c.GetProject(shortName)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/api/admin/projects/%s/fields", url.PathEscape(p.ID))
	return c.Post(path, map[string]any{
		"$type": "SimpleProjectCustomField",
		"field": map[string]any{"id": fieldID},
	}, nil)
}

func (c *Client) AddUserToProjectTeam(shortName, userID string) error {
	p, err := c.GetProject(shortName)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/api/admin/projects/%s/team", url.PathEscape(p.ID))
	return c.Post(path, map[string]any{"users": []map[string]any{{"id": userID}}}, nil)
}

func (c *Client) AddGroupToProjectTeam(shortName, groupID string) error {
	p, err := c.GetProject(shortName)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/api/admin/projects/%s/team", url.PathEscape(p.ID))
	return c.Post(path, map[string]any{"groups": []map[string]any{{"id": groupID}}}, nil)
}

func (c *Client) AttachWorkflowToProject(shortName, workflowID string) error {
	p, err := c.GetProject(shortName)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/api/admin/projects/%s/workflows", url.PathEscape(p.ID))
	return c.Post(path, map[string]any{"workflow": map[string]any{"id": workflowID}}, nil)
}

func (c *Client) DeleteProject(shortName string) error {
	p, err := c.GetProject(shortName)
	if err != nil {
		return err
	}
	return c.Delete(fmt.Sprintf("/api/admin/projects/%s", url.PathEscape(p.ID)))
}

func (c *Client) ConvertProjectToTemplate(shortName string) error {
	p, err := c.GetProject(shortName)
	if err != nil {
		return err
	}
	return c.Post(fmt.Sprintf("/api/admin/projects/%s", url.PathEscape(p.ID)), map[string]any{"template": true}, nil)
}

func (c *Client) SetProjectTimeTracking(shortName string, enabled bool) error {
	p, err := c.GetProject(shortName)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/api/admin/projects/%s/timeTrackingSettings", url.PathEscape(p.ID))
	return c.Post(path, map[string]any{"enabled": enabled}, nil)
}
