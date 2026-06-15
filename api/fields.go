package api

import (
	"fmt"
	"net/url"
)

func (c *Client) ListCustomFields() ([]CustomField, error) {
	var result []CustomField
	err := c.Get(pathAdminCustomFields+"?fields=id,name,fieldType(id),isPrivate,isAutoAttached,isDisplayedInIssueList,localizedName,isUpdateable&$top=-1", &result)
	return result, err
}

func (c *Client) GetCustomField(id string) (*CustomField, error) {
	var result CustomField
	path := fmt.Sprintf(pathAdminCustomField+"?fields=id,name,fieldType(id),isPrivate,isAutoAttached,isDisplayedInIssueList,localizedName,isUpdateable,fieldDefaults", url.PathEscape(id))
	err := c.Get(path, &result)
	return &result, err
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
	bundleType = NormalizeBundlePathType(bundleType)
	var result []Bundle
	path := fmt.Sprintf(pathAdminBundles+"?fields=id,name,isUpdateable,$type&$top=-1", url.PathEscape(bundleType))
	err := c.Get(path, &result)
	return result, err
}

// bundleTypeName maps a lowercase path type (e.g. "enum") to the YouTrack $type value
// (e.g. "EnumBundle"). It expects a normalized lowercase path key (e.g., "enum", "state")
// as returned by NormalizeBundlePathType.
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
	bundleType = NormalizeBundlePathType(bundleType)
	var result Bundle
	path := fmt.Sprintf(pathAdminBundles+"?fields=id,name,isUpdateable,$type", url.PathEscape(bundleType))
	body := map[string]any{
		"$type": bundleTypeName(bundleType),
		"name":  name,
	}
	if bundleType == "state" {
		// StateBundle requires isUpdateable per the YouTrack API schema.
		body["isUpdateable"] = true
	}
	err := c.Post(path, body, &result)
	return &result, err
}

// NormalizeBundlePathType accepts either the YouTrack $type name ("EnumBundle") or the
// lowercase URL path key ("enum") and always returns the correct URL path key.
func NormalizeBundlePathType(t string) string {
	switch t {
	case "enum", "EnumBundle":
		return "enum"
	case "state", "StateBundle":
		return "state"
	case "version", "VersionBundle":
		return "version"
	case "build", "BuildBundle":
		return "build"
	case "ownedField", "owned", "OwnedBundle":
		return "ownedField"
	case "user", "UserBundle":
		return "user"
	default:
		return t
	}
}

// bundleElementTypeName maps a bundle path type to the YouTrack $type for its elements.
// It expects a normalized lowercase path key (e.g., "enum", "state") as returned by
// NormalizeBundlePathType, not a PascalCase $type name.
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
	bundleType = NormalizeBundlePathType(bundleType)
	path := fmt.Sprintf(pathAdminBundleValues,
		url.PathEscape(bundleType), url.PathEscape(bundleID))
	return c.Post(path, map[string]any{
		"$type": bundleElementTypeName(bundleType),
		"name":  value,
	}, nil)
}

func (c *Client) DeleteBundle(bundleType, bundleID string) error {
	bundleType = NormalizeBundlePathType(bundleType)
	path := fmt.Sprintf(pathAdminBundle,
		url.PathEscape(bundleType), url.PathEscape(bundleID))
	return c.Delete(path)
}

func (c *Client) AttachBundleToField(fieldID, bundleID, bundleType string) error {
	bundleType = NormalizeBundlePathType(bundleType)
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
	return c.Post(path, map[string]any{
		"fieldDefaults": map[string]any{"isPublic": !isPrivate},
	}, nil)
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
