package api

import (
	"fmt"
	"net/url"
)

type issueFieldMetadata struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"$type"`
}

func (c *Client) resolveIssueField(issueID, field string) (*issueFieldMetadata, error) {
	fields, err := getIssueFields[issueFieldMetadata](c, issueID, "id,name,$type")
	if err != nil {
		return nil, err
	}
	for i := range fields {
		if fields[i].ID == field || fields[i].Name == field {
			return &fields[i], nil
		}
	}
	return nil, fmt.Errorf("field '%s' not found on issue %s", field, issueID)
}

// getIssueFields reads every custom field of an issue or draft with the given projection, page by page, so a
// project with many fields loses none.
func getIssueFields[T any](c *Client, issueID, fields string) ([]T, error) {
	const pageSize = 500
	var all []T
	for skip := 0; ; skip += pageSize {
		var page []T
		path := fmt.Sprintf(pathIssueFields+"?fields=%s&$top=%d&$skip=%d", url.PathEscape(issueID), fields, pageSize, skip)
		if err := c.Get(path, &page); err != nil {
			return nil, err
		}
		all = append(all, page...)
		if len(page) < pageSize {
			return all, nil
		}
	}
}
