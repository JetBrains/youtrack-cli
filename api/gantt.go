package api

import (
	"fmt"
	"net/url"
)

func (c *Client) AddGanttChart(name string, projectIDs []string, startDate int64) (*GanttChart, error) {
	projects := make([]map[string]any, len(projectIDs))
	for i, id := range projectIDs {
		projects[i] = map[string]any{"id": id}
	}
	var result GanttChart
	err := c.Post(pathGanttCharts+"?fields=id,name", map[string]any{
		"name":               name,
		"projects":           projects,
		"startDate":          startDate,
		"estimationTimeUnit": map[string]any{"id": "DAY"},
	}, &result)
	return &result, err
}

func (c *Client) ListGanttCharts() ([]GanttChart, error) {
	var result []GanttChart
	err := c.Get(pathGanttCharts+"?fields=id,name&$top=-1", &result)
	return result, err
}

func (c *Client) DeleteGanttChart(id string) error {
	return c.Delete(fmt.Sprintf(pathGanttChart, url.PathEscape(id)))
}

func (c *Client) GetGanttChart(id string) (*GanttChart, error) {
	var result GanttChart
	err := c.Get(fmt.Sprintf(pathGanttChart+"?fields=id,name,startDate", url.PathEscape(id)), &result)
	return &result, err
}

func (c *Client) UpdateGanttChart(id, name string, startDate int64) (*GanttChart, error) {
	body := map[string]any{}
	if name != "" {
		body["name"] = name
	}
	if startDate != 0 {
		body["startDate"] = startDate
	}
	var result GanttChart
	err := c.Post(fmt.Sprintf(pathGanttChart+"?fields=id,name,startDate", url.PathEscape(id)), body, &result)
	return &result, err
}

func (c *Client) ListGanttResources(ganttID string) ([]GanttResource, error) {
	var result []GanttResource
	err := c.Get(fmt.Sprintf(pathGanttResources+"?fields=id,user(id,login,fullName),startDate&$top=-1", url.PathEscape(ganttID)), &result)
	return result, err
}

func (c *Client) AddGanttResource(ganttID, userID string, startDate int64) (*GanttResource, error) {
	body := map[string]any{"user": map[string]any{"id": userID}}
	if startDate != 0 {
		body["startDate"] = startDate
	}
	var result GanttResource
	err := c.Post(fmt.Sprintf(pathGanttResources+"?fields=id,user(id,login,fullName),startDate", url.PathEscape(ganttID)), body, &result)
	return &result, err
}

func (c *Client) SetGanttResourceStartDate(ganttID, resourceID string, startDate int64) (*GanttResource, error) {
	var result GanttResource
	err := c.Post(fmt.Sprintf(pathGanttResource+"?fields=id,user(id,login,fullName),startDate", url.PathEscape(ganttID), url.PathEscape(resourceID)),
		map[string]any{"startDate": startDate}, &result)
	return &result, err
}

func (c *Client) UpdateGanttSettings(ganttID string, settings map[string]any) (map[string]any, error) {
	var result map[string]any
	err := c.Post(fmt.Sprintf(pathGanttSettings, url.PathEscape(ganttID)), settings, &result)
	return result, err
}
