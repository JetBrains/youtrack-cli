package api

import (
	"fmt"
	"net/url"
)

func (c *Client) ListWorkflows() ([]Workflow, error) {
	var result []Workflow
	return result, c.Get("/api/admin/workflows?fields=id,name,title,rules(id,name,title)&$top=-1", &result)
}

func (c *Client) AddWorkflow(name, title string) (*Workflow, error) {
	var result Workflow
	return &result, c.Post("/api/admin/workflows?fields=id,name,title", map[string]any{
		"name":  name,
		"title": title,
	}, &result)
}

func (c *Client) DeleteWorkflow(id string) error {
	return c.Delete(fmt.Sprintf("/api/admin/workflows/%s", url.PathEscape(id)))
}

func (c *Client) RestoreDefaultWorkflows() error {
	return c.Post("/api/admin/workflows/restoreStatus", map[string]any{"isDefault": true}, nil)
}

func (c *Client) AddWorkflowRule(workflowID, ruleType, name string) error {
	path := fmt.Sprintf("/api/admin/workflows/%s/rules", url.PathEscape(workflowID))
	return c.Post(path, map[string]any{
		"$type": ruleType,
		"name":  name,
	}, nil)
}
