package api

import (
	"fmt"
	"net/url"
)

func (c *Client) ListRoles() ([]Role, error) {
	var result []Role
	return result, c.Get(pathRoles+"?fields=id,name,description&$top=-1", &result)
}

func (c *Client) AddRole(name, description string, permissionIDs []string) (*Role, error) {
	var result Role
	perms := make([]map[string]any, len(permissionIDs))
	for i, id := range permissionIDs {
		perms[i] = map[string]any{"id": id}
	}
	return &result, c.Post(pathRoles+"?fields=id,name,description", map[string]any{
		"name":        name,
		"description": description,
		"permissions": perms,
	}, &result)
}

func (c *Client) DeleteRole(id string) error {
	return c.Delete(fmt.Sprintf(pathRole, url.PathEscape(id)))
}

func (c *Client) UpdateRole(id string, permissionIDs []string) error {
	perms := make([]map[string]any, len(permissionIDs))
	for i, pid := range permissionIDs {
		perms[i] = map[string]any{"id": pid}
	}
	return c.Post(fmt.Sprintf(pathRole, url.PathEscape(id)), map[string]any{
		"permissions": perms,
	}, nil)
}

func (c *Client) GetPermissions() ([]Permission, error) {
	var result []Permission
	return result, c.Get(pathPermissions+"?fields=id,name,description&$top=-1", &result)
}

func (c *Client) GrantRoleToUser(userID, roleID, projectID string) error {
	body := map[string]any{
		"holder": map[string]any{"id": userID, "$type": "User"},
		"role":   map[string]any{"id": roleID},
	}
	if projectID != "" {
		body["scope"] = map[string]any{
			"$type":   "ProjectBasedTagShareSettings",
			"project": map[string]any{"id": projectID},
		}
	}
	return c.Post(pathAssignedRoles, body, nil)
}

func (c *Client) GrantRoleToGroup(groupID, roleID, projectID string) error {
	body := map[string]any{
		"holder": map[string]any{"id": groupID, "$type": "UserGroup"},
		"role":   map[string]any{"id": roleID},
	}
	if projectID != "" {
		body["scope"] = map[string]any{
			"$type":   "ProjectBasedTagShareSettings",
			"project": map[string]any{"id": projectID},
		}
	}
	return c.Post(pathAssignedRoles, body, nil)
}
