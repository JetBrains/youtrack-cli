package api

import (
	"fmt"
	"net/url"
)

func (c *Client) GetRemovedDocuments() ([]RemovedDocument, error) {
	var result []RemovedDocument
	return result, c.Get(pathRemovedDocuments+"?fields=id,idReadable,summary&$top=-1", &result)
}

func (c *Client) RestoreDocument(id string) error {
	return c.Post(pathRemovedDocuments, map[string]any{
		"documents": []map[string]any{{"id": id}},
		"remove":    false,
	}, nil)
}

func (c *Client) DeleteReport(id string) error {
	return c.Delete(fmt.Sprintf(pathReport, url.PathEscape(id)))
}

func (c *Client) ListGanttCharts() ([]GanttChart, error) {
	var result []GanttChart
	return result, c.Get(pathGanttCharts+"?fields=id,name&$top=-1", &result)
}

func (c *Client) DeleteGanttChart(id string) error {
	return c.Delete(fmt.Sprintf(pathGanttChart, url.PathEscape(id)))
}
