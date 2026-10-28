package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const issueFieldValueFields = "id,name,localizedName,login,fullName,archived,banned,isResolved"

// IssueFieldValue is a selectable value returned in an issue's field context.
// The underlying entity may be a bundle element or a user; raw preserves every
// requested response field without imposing one entity's schema on the others.
type IssueFieldValue struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Login    string `json:"login,omitempty"`
	FullName string `json:"fullName,omitempty"`
	raw      json.RawMessage
}

func (v *IssueFieldValue) UnmarshalJSON(data []byte) error {
	type valueAlias IssueFieldValue
	if err := json.Unmarshal(data, (*valueAlias)(v)); err != nil {
		return err
	}
	v.raw = append(json.RawMessage(nil), data...)
	return nil
}

func (v *IssueFieldValue) MarshalJSON() ([]byte, error) {
	if v.raw != nil {
		return v.raw, nil
	}
	type valueAlias IssueFieldValue
	return json.Marshal((*valueAlias)(v))
}

// ListIssueFieldValues returns one page of choices visible in this issue's field
// context. field accepts its name or opaque issue-field ID. This uses the
// instance's possibleValues endpoint; unavailable or forbidden discovery is
// reported rather than falling back to a project's unrestricted bundle.
func (c *Client) ListIssueFieldValues(issueID, field string, limit, offset int) ([]IssueFieldValue, error) {
	if limit <= 0 || offset < 0 {
		return nil, fmt.Errorf("limit must be positive and offset must not be negative")
	}
	metadata, err := c.resolveIssueField(issueID, field)
	if err != nil {
		return nil, err
	}
	if !hasIssueFieldChoices(metadata.Type) {
		return nil, fmt.Errorf("field %q does not expose selectable values; scalar and state-machine fields are not supported", metadata.Name)
	}
	return c.listIssueFieldValues(issueID, metadata.ID, limit, offset)
}

func (c *Client) listIssueFieldValues(issueID, fieldID string, limit, offset int) ([]IssueFieldValue, error) {
	var result []IssueFieldValue
	path := fmt.Sprintf("/api/issues/%s/customFields/%s/possibleValues?fields=%s&$top=%d&$skip=%d",
		url.PathEscape(issueID), url.PathEscape(fieldID), issueFieldValueFields, limit, offset)
	err := c.Get(path, &result)
	return result, err
}

func hasIssueFieldChoices(fieldType string) bool {
	switch fieldType {
	case "SingleEnumIssueCustomField", "MultiEnumIssueCustomField", "StateIssueCustomField",
		"SingleVersionIssueCustomField", "MultiVersionIssueCustomField",
		"SingleBuildIssueCustomField", "MultiBuildIssueCustomField",
		"SingleOwnedIssueCustomField", "MultiOwnedIssueCustomField",
		"SingleUserIssueCustomField", "MultiUserIssueCustomField":
		return true
	default:
		return false
	}
}

func (c *Client) customFieldValueError(err error, issueID string, field *issueFieldMetadata, fieldType string) error {
	var response *APIError
	if !errors.As(err, &response) || response.StatusCode != http.StatusBadRequest || fieldType != "" || !hasIssueFieldChoices(field.Type) {
		return err
	}
	const maxChoices = 10
	values, lookupErr := c.listIssueFieldValues(issueID, field.ID, maxChoices+1, 0)
	if lookupErr != nil || len(values) == 0 {
		return err
	}
	more := len(values) > maxChoices
	if more {
		values = values[:maxChoices]
	}
	var names []string
	for _, value := range values {
		if value.Name != "" {
			names = append(names, fmt.Sprintf("%q", value.Name))
		}
	}
	if len(names) == 0 {
		return err
	}
	suffix := ""
	if more {
		suffix = ", ..."
	}
	return fmt.Errorf("%w; accessible values for %q: %s%s; use issue field values to list choices", err, field.Name, strings.Join(names, ", "), suffix)
}
