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
