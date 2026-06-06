package api

import (
	"fmt"
	"net/url"
)

func (c *Client) GetRemovedDocuments() ([]RemovedDocument, error) {
	var result []RemovedDocument
	err := c.Get(pathRemovedDocuments+"?fields=id,idReadable,summary&$top=-1", &result)
	return result, err
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
