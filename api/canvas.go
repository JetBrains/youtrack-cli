package api

import (
	"fmt"
	"net/url"
)

type AddCanvasRequest struct {
	Name  string `json:"name"`
	Owner *User  `json:"owner,omitempty"`
}

func (c *Client) ListCanvases() ([]Canvas, error) {
	var result []Canvas
	err := c.Get(pathPlanningCanvases+"?fields=id,name,owner(id,login,fullName),pinned,$type&$top=500", &result)
	return result, err
}

func (c *Client) ListCanvasCards(canvasID string) ([]CanvasNode, error) {
	var result []CanvasNode
	err := c.Get(fmt.Sprintf(pathPlanningCanvasCards+"?fields=id,label,x,y,description,width,height,$type&$top=500", url.PathEscape(canvasID)), &result)
	return result, err
}

func (c *Client) GetCanvasCard(canvasID, cardID string) (*CanvasNode, error) {
	var result CanvasNode
	err := c.Get(fmt.Sprintf(pathPlanningCanvasCard+"?fields=id,label,x,y,description,width,height,$type", url.PathEscape(canvasID), url.PathEscape(cardID)), &result)
	return &result, err
}

func (c *Client) AddCanvas(req AddCanvasRequest) (*Canvas, error) {
	var result Canvas
	err := c.Post(pathPlanningCanvases+"?fields=id,name,owner(id,login,fullName),pinned,$type", req, &result)
	return &result, err
}

func (c *Client) GetCanvas(id string) (*Canvas, error) {
	var result Canvas
	err := c.Get(fmt.Sprintf(pathPlanningCanvas+"?fields=id,name,owner(id,login,fullName),pinned,$type", url.PathEscape(id)), &result)
	return &result, err
}

func (c *Client) DeleteCanvas(id string) error {
	return c.Delete(fmt.Sprintf(pathPlanningCanvas, url.PathEscape(id)))
}

func (c *Client) ReloadCanvas(id string) error {
	return c.Post(fmt.Sprintf(pathPlanningCanvas, url.PathEscape(id)), map[string]any{"reload": true}, nil)
}

func (c *Client) AddCanvasNode(canvasID string, x, y int, label, description string) (*CanvasNode, error) {
	body := map[string]any{
		"x":        x,
		"y":        y,
		"label":    label,
		"nodeType": map[string]any{"id": "CardNode"},
	}
	if description != "" {
		body["description"] = description
	}
	var result CanvasNode
	err := c.Post(fmt.Sprintf(pathPlanningCanvasCards+"?fields=id,label,x,y,description,width,height,$type", url.PathEscape(canvasID)), body, &result)
	return &result, err
}

func (c *Client) GetCanvasLinks(canvasID string) ([]CanvasLink, error) {
	var result []CanvasLink
	err := c.Get(fmt.Sprintf(pathPlanningCanvasLinks+"?fields=id,sourceCard(id,label),targetCard(id,label)&$top=500", url.PathEscape(canvasID)), &result)
	return result, err
}

// LinkCanvasNodes creates a link between two canvas nodes.
// linkPrototypeID is required when kind is CanvasLinkKindPrototyped; ignored otherwise.
func (c *Client) LinkCanvasNodes(canvasID, sourceCardID, targetCardID string, kind CanvasLinkKind, linkPrototypeID string) error {
	if kind == "" {
		kind = CanvasLinkKindOneWay
	}
	body := map[string]any{
		"type":       map[string]any{"id": kind},
		"sourceCard": map[string]any{"id": sourceCardID},
		"targetCard": map[string]any{"id": targetCardID},
	}
	if linkPrototypeID != "" {
		body["linkPrototype"] = map[string]any{"id": linkPrototypeID}
	}
	return c.Post(fmt.Sprintf(pathPlanningCanvasLinks+"?fields=id,sourceCard(id,label),targetCard(id,label)", url.PathEscape(canvasID)), body, nil)
}
