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
	var fields []issueFieldMetadata
	path := fmt.Sprintf(pathIssueFields+"?fields=id,name,$type&$top=500", url.PathEscape(issueID))
	if err := c.Get(path, &fields); err != nil {
		return nil, err
	}
	for i := range fields {
		if fields[i].ID == field || fields[i].Name == field {
			return &fields[i], nil
		}
	}
	return nil, fmt.Errorf("field '%s' not found on issue %s", field, issueID)
}
