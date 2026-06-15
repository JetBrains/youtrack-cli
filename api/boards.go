package api

import (
	"fmt"
	"net/url"
)

func (c *Client) ListBoards() ([]Board, error) {
	var result []Board
	err := c.Get(pathBoards+"?fields=id,name,owner(id,login,fullName),status,currentSprint(id,name),projects(id,shortName),sprints(id,name,archived,isDefault)&$top=-1", &result)
	return result, err
}

func (c *Client) GetBoard(id string) (*Board, error) {
	var result Board
	path := fmt.Sprintf(pathBoard+"?fields=id,name,owner(id,login,fullName),status,currentSprint(id,name),projects(id,shortName),sprints(id,name,archived,isDefault)", url.PathEscape(id))
	err := c.Get(path, &result)
	return &result, err
}

func (c *Client) AddBoard(name, projectShortName string) (*Board, error) {
	p, err := c.GetProject(projectShortName)
	if err != nil {
		return nil, err
	}
	var result Board
	err = c.Post(pathBoards+"?fields=id,name", map[string]any{
		"name":     name,
		"projects": []map[string]any{{"id": p.ID}},
	}, &result)
	return &result, err
}

func (c *Client) DeleteBoard(id string) error {
	return c.Delete(fmt.Sprintf(pathBoard, url.PathEscape(id)))
}

func (c *Client) ListSprints(boardID string) ([]Sprint, error) {
	var result []Sprint
	path := fmt.Sprintf(pathBoardSprints+"?fields=id,name,goal,start,finish,archived,isDefault&$top=-1", url.PathEscape(boardID))
	err := c.Get(path, &result)
	return result, err
}

func (c *Client) GetSprint(boardID, sprintID string) (*Sprint, error) {
	var result Sprint
	path := fmt.Sprintf(pathBoardSprint+"?fields=id,name,goal,start,finish,archived,isDefault", url.PathEscape(boardID), url.PathEscape(sprintID))
	err := c.Get(path, &result)
	return &result, err
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
	path := fmt.Sprintf(pathBoardSprints+"?fields=id,name,goal,start,finish,archived,isDefault", url.PathEscape(boardID))
	err := c.Post(path, body, &result)
	return &result, err
}
