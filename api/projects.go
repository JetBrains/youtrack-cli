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
	err := c.Get(pathAdminProjects+"?fields="+projectFields+"&$top=-1", &result)
	return result, err
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
	err := c.Post(pathAdminProjects+"?fields="+projectFields, req, &result)
	return &result, err
}

func (c *Client) ArchiveProject(shortName string) error {
	p, err := c.GetProject(shortName)
	if err != nil {
		return err
	}
	return c.Post(fmt.Sprintf(pathAdminProject, url.PathEscape(p.ID)), map[string]any{"archived": true}, nil)
}

func (c *Client) GetProjectTemplates() ([]ProjectTemplate, error) {
	var result []ProjectTemplate
	err := c.Get(pathAdminProjectTemplates+"?fields=id,name,shortName&$top=-1", &result)
	return result, err
}

func (c *Client) GetProjectCustomFields(shortName string) ([]ProjectCustomField, error) {
	p, err := c.GetProject(shortName)
	if err != nil {
		return nil, err
	}
	var result []ProjectCustomField
	path := fmt.Sprintf(pathAdminProjectFields+"?fields=id,field(id,name),bundle(id,name)&$top=-1", url.PathEscape(p.ID))
	err = c.Get(path, &result)
	return result, err
}

func (c *Client) AddProjectCustomField(shortName, fieldID string) error {
	p, err := c.GetProject(shortName)
	if err != nil {
		return err
	}
	path := fmt.Sprintf(pathAdminProjectFields, url.PathEscape(p.ID))
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
	path := fmt.Sprintf(pathAdminProjectTeamOwnUsers, url.PathEscape(p.ID))
	return c.Post(path, map[string]any{"id": userID}, nil)
}

func (c *Client) AddGroupToProjectTeam(shortName, groupID string) error {
	p, err := c.GetProject(shortName)
	if err != nil {
		return err
	}
	path := fmt.Sprintf(pathAdminProjectTeamGroups, url.PathEscape(p.ID))
	return c.Post(path, map[string]any{"id": groupID}, nil)
}

func (c *Client) AttachWorkflowToProject(shortName, workflowID string) error {
	p, err := c.GetProject(shortName)
	if err != nil {
		return err
	}
	path := fmt.Sprintf(pathAdminProjectWorkflows, url.PathEscape(p.ID))
	return c.Post(path, map[string]any{"id": workflowID}, nil)
}

func (c *Client) DeleteProject(shortName string) error {
	p, err := c.GetProject(shortName)
	if err != nil {
		return err
	}
	return c.Delete(fmt.Sprintf(pathAdminProject, url.PathEscape(p.ID)))
}

func (c *Client) ConvertProjectToTemplate(shortName string) error {
	p, err := c.GetProject(shortName)
	if err != nil {
		return err
	}
	return c.Post(fmt.Sprintf(pathAdminProject, url.PathEscape(p.ID)), map[string]any{"template": true}, nil)
}

func (c *Client) SetProjectTimeTracking(shortName string, enabled bool) error {
	p, err := c.GetProject(shortName)
	if err != nil {
		return err
	}
	path := fmt.Sprintf(pathAdminProjectTimeTracking, url.PathEscape(p.ID))
	return c.Post(path, map[string]any{"enabled": enabled}, nil)
}
