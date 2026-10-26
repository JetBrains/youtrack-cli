package api

import (
	"fmt"
	"net/url"
)

func (c *Client) ListWorkflows() ([]Workflow, error) {
	const pageSize = 500
	var result []Workflow
	for skip := 0; ; skip += pageSize {
		var page []Workflow
		path := fmt.Sprintf(pathAdminWorkflows+"?fields=id,name,title,rules(id,name,title)&$top=%d&$skip=%d", pageSize, skip)
		if err := c.Get(path, &page); err != nil {
			return nil, err
		}
		result = append(result, page...)
		if len(page) < pageSize {
			return result, nil
		}
	}
}

func (c *Client) AddWorkflow(name, title string) (*Workflow, error) {
	var result Workflow
	err := c.Post(pathAdminWorkflows+"?fields=id,name,title", map[string]any{
		"name":  name,
		"title": title,
	}, &result)
	return &result, err
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
