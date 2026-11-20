package api

import (
	"fmt"
	"net/url"
)

const issueFieldListFields = "id,name,$type,value(name,presentation,text,login,fullName,minutes)," +
	"projectCustomField(canBeEmpty,emptyFieldText,field(fieldType(valueType,isMultiValue)))"

// IssueFieldInfo describes one custom field of an issue or draft with what a caller needs to change
// it: the type of its values, whether it holds several, whether it may be empty, and whether its
// values can be listed with ListIssueFieldValues.
type IssueFieldInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ValueType  string `json:"valueType"`
	MultiValue bool   `json:"multiValue"`
	Required   bool   `json:"required"`
	HasChoices bool   `json:"hasChoices"`
	EmptyText  string `json:"emptyText,omitempty"`
	Value      any    `json:"value"`
}

type issueFieldListEntry struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Type               string `json:"$type"`
	Value              any    `json:"value"`
	ProjectCustomField *struct {
		CanBeEmpty     *bool  `json:"canBeEmpty"`
		EmptyFieldText string `json:"emptyFieldText"`
		Field          *struct {
			FieldType *FieldValueType `json:"fieldType"`
		} `json:"field"`
	} `json:"projectCustomField"`
}

// ListIssueFields returns every custom field of an issue or draft. It reads the issue's own fields,
// so it needs only permission to read the issue.
func (c *Client) ListIssueFields(issueID string) ([]IssueFieldInfo, error) {
	const pageSize = 500
	fields := []IssueFieldInfo{}
	for skip := 0; ; skip += pageSize {
		var page []issueFieldListEntry
		path := fmt.Sprintf(pathIssueFields+"?fields=%s&$top=%d&$skip=%d",
			url.PathEscape(issueID), issueFieldListFields, pageSize, skip)
		if err := c.Get(path, &page); err != nil {
			return nil, err
		}
		for _, e := range page {
			fields = append(fields, issueFieldInfo(e))
		}
		if len(page) < pageSize {
			return fields, nil
		}
	}
}

func issueFieldInfo(e issueFieldListEntry) IssueFieldInfo {
	info := IssueFieldInfo{ID: e.ID, Name: e.Name, Value: e.Value, HasChoices: hasIssueFieldChoices(e.Type)}
	if pcf := e.ProjectCustomField; pcf != nil {
		info.Required = pcf.CanBeEmpty != nil && !*pcf.CanBeEmpty
		info.EmptyText = pcf.EmptyFieldText
		if pcf.Field != nil && pcf.Field.FieldType != nil {
			info.ValueType = pcf.Field.FieldType.ValueType
			info.MultiValue = pcf.Field.FieldType.IsMultiValue
		}
	}
	return info
}
