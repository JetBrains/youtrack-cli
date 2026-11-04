package api

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ProjectFieldSchema is what a project says about one of its custom fields: enough to write a value for it.
type ProjectFieldSchema struct {
	Field *struct {
		Name      string `json:"name"`
		FieldType *struct {
			ValueType    string `json:"valueType"`
			IsMultiValue bool   `json:"isMultiValue"`
		} `json:"fieldType"`
	} `json:"field"`
}

func (s ProjectFieldSchema) name() string {
	if s.Field == nil {
		return ""
	}
	return s.Field.Name
}

// IssueCustomFieldValue is one custom field value in an issue create request.
type IssueCustomFieldValue struct {
	Name  string `json:"name"`
	Type  string `json:"$type"`
	Value any    `json:"value"`
}

// GetProjectFieldSchemas returns the custom fields of a project with their value types.
func (c *Client) GetProjectFieldSchemas(projectID string) ([]ProjectFieldSchema, error) {
	var result []ProjectFieldSchema
	path := fmt.Sprintf(pathAdminProjectFields+"?fields=field(name,fieldType(valueType,isMultiValue))&$top=500", url.PathEscape(projectID))
	err := c.Get(path, &result)
	return result, err
}

// FindProjectField matches a field name case-insensitively, as the command parser does.
func FindProjectField(schemas []ProjectFieldSchema, name string) (ProjectFieldSchema, bool) {
	for _, s := range schemas {
		if strings.EqualFold(s.name(), name) {
			return s, true
		}
	}
	return ProjectFieldSchema{}, false
}

// ProjectFieldNames lists the field names of a project, for error messages.
func ProjectFieldNames(schemas []ProjectFieldSchema) []string {
	names := make([]string, 0, len(schemas))
	for _, s := range schemas {
		names = append(names, s.name())
	}
	return names
}

// IsUserField reports whether the field holds users, whose values are logins.
func (s ProjectFieldSchema) IsUserField() bool {
	return s.Field != nil && s.Field.FieldType != nil && s.Field.FieldType.ValueType == "user"
}

// BuildIssueCustomField turns a field value given as text into the typed value the issue create request expects.
// An empty value leaves the field empty.
func BuildIssueCustomField(schema ProjectFieldSchema, value string) (IssueCustomFieldValue, error) {
	if schema.Field == nil || schema.Field.FieldType == nil {
		return IssueCustomFieldValue{}, fmt.Errorf("field %q has no type", schema.name())
	}
	name, valueType, multi := schema.Field.Name, schema.Field.FieldType.ValueType, schema.Field.FieldType.IsMultiValue
	result := IssueCustomFieldValue{Name: name}

	bundled := map[string]string{
		"enum": "Enum", "version": "Version", "build": "Build", "ownedField": "Owned", "user": "User", "group": "Group",
	}
	if kind, ok := bundled[valueType]; ok {
		key := "name"
		if valueType == "user" {
			key = "login"
		}
		if multi {
			result.Type = "Multi" + kind + "IssueCustomField"
			result.Value = []map[string]string{}
			if value != "" {
				result.Value = []map[string]string{{key: value}}
			}
		} else {
			result.Type = "Single" + kind + "IssueCustomField"
			if value != "" {
				result.Value = map[string]string{key: value}
			}
		}
		return result, nil
	}

	switch valueType {
	case "state":
		result.Type = "StateIssueCustomField"
		if value != "" {
			result.Value = map[string]string{"name": value}
		}
	case "text":
		result.Type = "TextIssueCustomField"
		result.Value = map[string]string{"text": value}
	case "period":
		result.Type = "PeriodIssueCustomField"
		if value != "" {
			result.Value = map[string]string{"presentation": value}
		}
	case "string":
		result.Type = "SimpleIssueCustomField"
		if value != "" {
			result.Value = value
		}
	case "integer":
		result.Type = "SimpleIssueCustomField"
		if value != "" {
			n, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return result, fmt.Errorf("field %q needs an integer, got %q", name, value)
			}
			result.Value = n
		}
	case "float":
		result.Type = "SimpleIssueCustomField"
		if value != "" {
			f, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return result, fmt.Errorf("field %q needs a number, got %q", name, value)
			}
			result.Value = f
		}
	case "date", "date and time":
		result.Type = "SimpleIssueCustomField"
		if valueType == "date" {
			result.Type = "DateIssueCustomField"
		}
		if value != "" {
			ms, err := parseDateMillis(value, valueType == "date")
			if err != nil {
				return result, fmt.Errorf("field %q needs a date such as 2026-01-31 or 2026-01-31T14:00:00Z, got %q", name, value)
			}
			result.Value = ms
		}
	default:
		return result, fmt.Errorf("field %q has type %q, which issue create cannot set; set it after creating with issue field set", name, valueType)
	}
	return result, nil
}

// parseDateMillis reads epoch milliseconds or a UTC date/time. A bare date for a date-only field
// is noon UTC, matching YouTrack's date-only storage convention.
func parseDateMillis(value string, dateOnly bool) (int64, error) {
	if ms, err := strconv.ParseInt(value, 10, 64); err == nil {
		return ms, nil
	}
	if t, err := time.Parse("2006-01-02", value); err == nil {
		if dateOnly {
			t = t.Add(12 * time.Hour)
		}
		return t.UnixMilli(), nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04", "2006-01-02 15:04"} {
		if t, err := time.Parse(layout, value); err == nil {
			return t.UnixMilli(), nil
		}
	}
	return 0, fmt.Errorf("unrecognized date %q", value)
}
