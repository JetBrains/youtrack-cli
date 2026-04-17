package api

import (
	"fmt"
	"net/url"
)

func (c *Client) ListCustomFields() ([]CustomField, error) {
	var result []CustomField
	err := c.Get(pathAdminCustomFields+"?fields=id,name,fieldType(id)&$top=-1", &result)
	return result, err
}

func (c *Client) AddCustomField(name, fieldTypeID string) (*CustomField, error) {
	var result CustomField
	err := c.Post(pathAdminCustomFields+"?fields=id,name,fieldType(id)", map[string]any{
		"name":      name,
		"fieldType": map[string]any{"id": fieldTypeID},
	}, &result)
	return &result, err
}

func (c *Client) DeleteCustomField(id string) error {
	return c.Delete(fmt.Sprintf(pathAdminCustomField, url.PathEscape(id)))
}

func (c *Client) ListBundles(bundleType string) ([]Bundle, error) {
	var result []Bundle
	path := fmt.Sprintf(pathAdminBundles+"?fields=id,name&$top=-1", url.PathEscape(bundleType))
	err := c.Get(path, &result)
	return result, err
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
	case "owned", "ownedField":
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
	path := fmt.Sprintf(pathAdminBundles+"?fields=id,name", url.PathEscape(bundleType))
	err := c.Post(path, map[string]any{
		"$type": bundleTypeName(bundleType),
		"name":  name,
	}, &result)
	return &result, err
}

// bundleElementTypeName maps a bundle path type to the YouTrack $type for its elements.
func bundleElementTypeName(pathType string) string {
	switch pathType {
	case "state":
		return "StateBundleElement"
	case "version":
		return "VersionBundleElement"
	case "build":
		return "BuildBundleElement"
	case "owned", "ownedField":
		return "OwnedBundleElement"
	default: // "enum" and unknown types
		return "EnumBundleElement"
	}
}

func (c *Client) AddBundleValue(bundleType, bundleID, value string) error {
	path := fmt.Sprintf(pathAdminBundleValues,
		url.PathEscape(bundleType), url.PathEscape(bundleID))
	return c.Post(path, map[string]any{
		"$type": bundleElementTypeName(bundleType),
		"name":  value,
	}, nil)
}

func (c *Client) DeleteBundle(bundleType, bundleID string) error {
	path := fmt.Sprintf(pathAdminBundle,
		url.PathEscape(bundleType), url.PathEscape(bundleID))
	return c.Delete(path)
}

func (c *Client) AttachBundleToField(fieldID, bundleID, bundleType string) error {
	path := fmt.Sprintf(pathAdminCustomFieldBundle,
		url.PathEscape(fieldID))
	return c.Put(path, map[string]any{"id": bundleID, "$type": bundleTypeName(bundleType)}, nil)
}

func (c *Client) ListWorkItemTypes() ([]WorkItemType, error) {
	var result []WorkItemType
	err := c.Get(pathAdminWorkItemTypes+"?fields=id,name,autoAttach&$top=-1", &result)
	return result, err
}

func (c *Client) AddWorkItemType(name string, autoAttach bool) (*WorkItemType, error) {
	var result WorkItemType
	err := c.Post(pathAdminWorkItemTypes+"?fields=id,name,autoAttach", map[string]any{
		"name":       name,
		"autoAttach": autoAttach,
	}, &result)
	return &result, err
}

func (c *Client) ChangeCustomFieldPrivacy(fieldID string, isPrivate bool) error {
	path := fmt.Sprintf(pathAdminCustomField, url.PathEscape(fieldID))
	return c.Post(path, map[string]any{"isPrivate": isPrivate}, nil)
}

func (c *Client) GetWorkTimeSettings() (*WorkTimeSettings, error) {
	var result WorkTimeSettings
	err := c.Get(pathAdminWorkTimeSettings+"?fields=id,daysAWeek,minutesADay,minutesADayPresentation,firstDayOfWeek", &result)
	return &result, err
}

func (c *Client) SetWorkTimeSettings(minutesADayPresentation string) error {
	return c.Post(pathAdminWorkTimeSettings, map[string]any{
		"minutesADayPresentation": minutesADayPresentation,
	}, nil)
}

func (c *Client) GetCustomFieldPrototypes() ([]CustomField, error) {
	var result []CustomField
	err := c.Get(pathAdminCustomFields+"?fields=id,name,fieldType(id),isPrivate&$top=-1", &result)
	return result, err
}
