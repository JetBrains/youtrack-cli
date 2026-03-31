package api

import (
	"fmt"
	"net/url"
)

func (c *Client) ListWorkflows() ([]Workflow, error) {
	var result []Workflow
	return result, c.Get(pathAdminWorkflows+"?fields=id,name,title,rules(id,name,title)&$top=-1", &result)
}

func (c *Client) AddWorkflow(name, title string) (*Workflow, error) {
	var result Workflow
	return &result, c.Post(pathAdminWorkflows+"?fields=id,name,title", map[string]any{
		"name":  name,
		"title": title,
	}, &result)
}

func (c *Client) DeleteWorkflow(id string) error {
	return c.Delete(fmt.Sprintf(pathAdminWorkflow, url.PathEscape(id)))
}

func (c *Client) RestoreDefaultWorkflows() error {
	return c.Post(pathAdminWorkflowsRestoreStatus, map[string]any{"isDefault": true}, nil)
}

func (c *Client) AddWorkflowRule(workflowID, ruleType, name string) error {
	path := fmt.Sprintf(pathAdminWorkflowRules, url.PathEscape(workflowID))
	return c.Post(path, map[string]any{
		"$type": ruleType,
		"name":  name,
	}, nil)
}
