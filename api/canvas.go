package api

import (
	"fmt"
	"net/url"
)

type AddCanvasRequest struct {
	Name  string `json:"name"`
	Owner *User  `json:"owner,omitempty"`
}

func (c *Client) AddCanvas(req AddCanvasRequest) (*Canvas, error) {
	var result Canvas
	return &result, c.Post("/api/planningCanvases?fields=id,name", req, &result)
}

func (c *Client) GetCanvas(id string) (*Canvas, error) {
	var result Canvas
	return &result, c.Get(fmt.Sprintf("/api/planningCanvases/%s?fields=id,name", url.PathEscape(id)), &result)
}

func (c *Client) DeleteCanvas(id string) error {
	return c.Delete(fmt.Sprintf("/api/planningCanvases/%s", url.PathEscape(id)))
}

func (c *Client) ReloadCanvas(id string) error {
	return c.Post(fmt.Sprintf("/api/planningCanvases/%s", url.PathEscape(id)), map[string]any{"reload": true}, nil)
}

func (c *Client) AddCanvasNode(canvasID string, x, y int, label, description string) error {
	return c.Post(fmt.Sprintf("/api/planningCanvases/%s/cards", url.PathEscape(canvasID)), map[string]any{
		"x": x, "y": y, "label": label, "description": description,
	}, nil)
}

func (c *Client) GetCanvasLinks(canvasID string) ([]CanvasLink, error) {
	var result []CanvasLink
	return result, c.Get(fmt.Sprintf("/api/planningCanvases/%s/links?fields=id,sourceCard(id,label),targetCard(id,label)&$top=-1", url.PathEscape(canvasID)), &result)
}

func (c *Client) LinkCanvasNodes(canvasID, sourceCardID, targetCardID string) error {
	return c.Post(fmt.Sprintf("/api/planningCanvases/%s/links", url.PathEscape(canvasID)), map[string]any{
		"$type":      "PlanningCanvasLink",
		"sourceCard": map[string]any{"id": sourceCardID},
		"targetCard": map[string]any{"id": targetCardID},
	}, nil)
}
