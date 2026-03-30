package api

import (
	"fmt"
	"net/url"
)

func (c *Client) GetRemovedDocuments() ([]RemovedDocument, error) {
	var result []RemovedDocument
	return result, c.Get("/api/removedDocuments?fields=id,idReadable,summary&$top=-1", &result)
}

func (c *Client) RestoreDocument(id string) error {
	return c.Post("/api/removedDocuments", map[string]any{
		"documents": []map[string]any{{"id": id}},
		"remove":    false,
	}, nil)
}

func (c *Client) DeleteReport(id string) error {
	return c.Delete(fmt.Sprintf("/api/reports/%s", url.PathEscape(id)))
}

func (c *Client) ListGanttCharts() ([]GanttChart, error) {
	var result []GanttChart
	return result, c.Get("/api/ganttCharts?fields=id,name&$top=-1", &result)
}

func (c *Client) DeleteGanttChart(id string) error {
	return c.Delete(fmt.Sprintf("/api/ganttCharts/%s", url.PathEscape(id)))
}
