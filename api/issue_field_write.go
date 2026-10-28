package api

import (
	"fmt"
	"net/url"
	"strconv"
)

// SetCustomFieldValues replaces an issue field's value with values; an empty list clears it.
// More than one value needs a multi-value field.
func (c *Client) SetCustomFieldValues(issueID, fieldName string, values []string, fieldType string) error {
	field, err := c.resolveIssueField(issueID, fieldName)
	if err != nil {
		return err
	}
	multi := isMultiValueField(field.Type)
	if len(values) > 1 && !multi {
		return fmt.Errorf("field '%s' holds a single value; pass one value or --clear", fieldName)
	}
	converted := make([]any, len(values))
	for i, v := range values {
		if converted[i], err = c.customFieldWriteValue(v, fieldType); err != nil {
			return err
		}
	}
	var value any
	switch {
	case multi:
		value = converted
	case len(converted) == 1:
		value = converted[0]
	}
	err = c.Post(fmt.Sprintf(pathIssueField, url.PathEscape(issueID), url.PathEscape(field.ID)),
		map[string]any{"value": value}, nil)
	return err
}

// customFieldWriteValue converts a CLI value to the request shape for a --type, as SetCustomField does.
func (c *Client) customFieldWriteValue(value, fieldType string) (any, error) {
	switch fieldType {
	case "period":
		return map[string]any{"presentation": value}, nil
	case "text":
		return map[string]any{"text": value}, nil
	case "user":
		user, err := c.GetUser(value)
		if err != nil {
			return nil, err
		}
		return map[string]any{"id": user.ID}, nil
	case "int":
		n, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("invalid integer value '%s': %w", value, err)
		}
		return n, nil
	case "string":
		return value, nil
	default:
		return map[string]any{"name": value}, nil
	}
}
