package api

import (
	"fmt"
	"net/url"
)

func (c *Client) ListCustomFields() ([]CustomField, error) {
	var result []CustomField
	return result, c.Get("/api/admin/customFieldSettings/customFields?fields=id,name,fieldType(id)&$top=-1", &result)
}

func (c *Client) AddCustomField(name, fieldTypeID string) (*CustomField, error) {
	var result CustomField
	return &result, c.Post("/api/admin/customFieldSettings/customFields?fields=id,name,fieldType(id)", map[string]any{
		"name":      name,
		"fieldType": map[string]any{"id": fieldTypeID},
	}, &result)
}

func (c *Client) DeleteCustomField(id string) error {
	return c.Delete(fmt.Sprintf("/api/admin/customFieldSettings/customFields/%s", url.PathEscape(id)))
}

func (c *Client) ListBundles(bundleType string) ([]Bundle, error) {
	var result []Bundle
	path := fmt.Sprintf("/api/admin/customFieldSettings/bundles/%s?fields=id,name&$top=-1", url.PathEscape(bundleType))
	return result, c.Get(path, &result)
}

// bundleTypeName maps a lowercase path type (e.g. "enum") to the YouTrack $type value
// (e.g. "EnumBundle"). Accepts either form — if already in PascalCase it passes through.
func bundleTypeName(pathType string) string {
	switch pathType {
	case "enum":
		return "EnumBundle"
	case "state":
		return "StateBundle"
	case "version":
		return "VersionBundle"
	case "owned":
		return "OwnedBundle"
	case "build":
		return "BuildBundle"
	case "user":
		return "UserBundle"
	default:
		return pathType
	}
}

func (c *Client) CreateBundle(bundleType, name string) (*Bundle, error) {
	var result Bundle
	path := fmt.Sprintf("/api/admin/customFieldSettings/bundles/%s?fields=id,name", url.PathEscape(bundleType))
	return &result, c.Post(path, map[string]any{
		"$type": bundleTypeName(bundleType),
		"name":  name,
	}, &result)
}

func (c *Client) AddBundleValue(bundleType, bundleID, value string) error {
	path := fmt.Sprintf("/api/admin/customFieldSettings/bundles/%s/%s/values",
		url.PathEscape(bundleType), url.PathEscape(bundleID))
	return c.Post(path, map[string]any{"name": value}, nil)
}

func (c *Client) DeleteBundle(bundleType, bundleID string) error {
	path := fmt.Sprintf("/api/admin/customFieldSettings/bundles/%s/%s",
		url.PathEscape(bundleType), url.PathEscape(bundleID))
	return c.Delete(path)
}

func (c *Client) AttachBundleToField(fieldID, bundleID, bundleType string) error {
	path := fmt.Sprintf("/api/admin/customFieldSettings/customFields/%s/fieldDefaults/bundle",
		url.PathEscape(fieldID))
	return c.Put(path, map[string]any{"id": bundleID, "$type": bundleTypeName(bundleType)}, nil)
}

func (c *Client) ListWorkItemTypes() ([]WorkItemType, error) {
	var result []WorkItemType
	return result, c.Get("/api/admin/timeTrackingSettings/workItemTypes?fields=id,name,autoAttach&$top=-1", &result)
}

func (c *Client) AddWorkItemType(name string, autoAttach bool) (*WorkItemType, error) {
	var result WorkItemType
	return &result, c.Post("/api/admin/timeTrackingSettings/workItemTypes?fields=id,name,autoAttach", map[string]any{
		"name":       name,
		"autoAttach": autoAttach,
	}, &result)
}

func (c *Client) ChangeCustomFieldPrivacy(fieldID string, isPrivate bool) error {
	path := fmt.Sprintf("/api/admin/customFieldSettings/customFields/%s", url.PathEscape(fieldID))
	return c.Post(path, map[string]any{"isPrivate": isPrivate}, nil)
}

func (c *Client) GetWorkTimeSettings() (*WorkTimeSettings, error) {
	var result WorkTimeSettings
	return &result, c.Get("/api/admin/timeTrackingSettings/workTimeSettings?fields=id,daysAWeek,minutesADay,minutesADayPresentation,firstDayOfWeek", &result)
}

func (c *Client) SetWorkTimeSettings(minutesADayPresentation string) error {
	return c.Post("/api/admin/timeTrackingSettings/workTimeSettings", map[string]any{
		"minutesADayPresentation": minutesADayPresentation,
	}, nil)
}

func (c *Client) GetCustomFieldPrototypes() ([]CustomField, error) {
	var result []CustomField
	return result, c.Get("/api/admin/customFieldSettings/customFields?fields=id,name,fieldType(id),isPrivate&$top=-1", &result)
}
