package api

import (
	"fmt"
	"net/url"
)

func (c *Client) ListBoards() ([]Board, error) {
	var result []Board
	return result, c.Get("/api/agiles?fields=id,name&$top=-1", &result)
}

func (c *Client) GetBoard(id string) (*Board, error) {
	var result Board
	path := fmt.Sprintf("/api/agiles/%s?fields=id,name,projects(id,shortName),sprints(id,name)", url.PathEscape(id))
	return &result, c.Get(path, &result)
}

func (c *Client) AddBoard(name string) (*Board, error) {
	var result Board
	return &result, c.Post("/api/agiles?fields=id,name", map[string]any{"name": name}, &result)
}

func (c *Client) DeleteBoard(id string) error {
	return c.Delete(fmt.Sprintf("/api/agiles/%s", url.PathEscape(id)))
}

func (c *Client) AddSprint(boardID, name string, start, finish int64, goal string) (*Sprint, error) {
	var result Sprint
	body := map[string]any{"name": name}
	if start > 0 {
		body["start"] = start
	}
	if finish > 0 {
		body["finish"] = finish
	}
	if goal != "" {
		body["goal"] = goal
	}
	path := fmt.Sprintf("/api/agiles/%s/sprints?fields=id,name", url.PathEscape(boardID))
	return &result, c.Post(path, body, &result)
}
