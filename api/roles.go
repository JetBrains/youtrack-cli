package api

import (
	"fmt"
	"net/url"
)

func (c *Client) ListRoles() ([]Role, error) {
	var result []Role
	err := c.Get(pathRoles+"?fields=id,name,description&$top=-1", &result)
	return result, err
}

func (c *Client) AddRole(name, description string, permissionIDs []string) (*Role, error) {
	var result Role
	perms := make([]map[string]any, len(permissionIDs))
	for i, id := range permissionIDs {
		perms[i] = map[string]any{"id": id}
	}
	err := c.Post(pathRoles+"?fields=id,name,description", map[string]any{
		"name":        name,
		"description": description,
		"permissions": perms,
	}, &result)
	return &result, err
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
	err := c.Get(pathPermissions+"?fields=id,name,description&$top=-1", &result)
	return result, err
}

func (c *Client) GrantRoleToUser(userID, roleID, projectID string) error {
	body := map[string]any{
		"holder": map[string]any{"id": userID, "$type": "User"},
		"role":   map[string]any{"id": roleID},
	}
	if projectID != "" {
		body["scope"] = map[string]any{
			"$type":   "ProjectScope",
			"project": map[string]any{"id": projectID},
		}
	} else {
		body["scope"] = map[string]any{"$type": "GlobalScope"}
	}
	return c.Post(pathAssignedRoles, body, nil)
}

func (c *Client) GrantRoleToGroup(groupID, roleID, projectID string) error {
	// Fetch the group to get its concrete $type (NestedGroup, RegisteredUsersGroup, AllUsersGroup)
	// required by YouTrack's polymorphic Jackson deserialization on the holder field.
	var g struct {
		ID   string `json:"id"`
		Type string `json:"$type"`
	}
	if err := c.Get(fmt.Sprintf(pathGroup+"?fields=id,$type", url.PathEscape(groupID)), &g); err != nil {
		return err
	}
	groupType := g.Type
	if groupType == "" {
		groupType = "NestedGroup"
	}
	body := map[string]any{
		"holder": map[string]any{"id": groupID, "$type": groupType},
		"role":   map[string]any{"id": roleID},
	}
	if projectID != "" {
		body["scope"] = map[string]any{
			"$type":   "ProjectScope",
			"project": map[string]any{"id": projectID},
		}
	} else {
		body["scope"] = map[string]any{"$type": "GlobalScope"}
	}
	return c.Post(pathAssignedRoles, body, nil)
}

func (c *Client) ListAssignedRoles(holderID string) ([]AssignedRole, error) {
	path := pathAssignedRoles +
		"?fields=id,role(id,name,description),holder($type,id,name,login,email)," +
		"scope($type,id,project(id,name,shortName))&$top=-1"
	if holderID != "" {
		path += "&query=" + url.QueryEscape("holder: "+holderID)
	}
	var result []AssignedRole
	err := c.Get(path, &result)
	return result, err
}

func (c *Client) RevokeRole(assignedRoleID string) error {
	return c.Delete(fmt.Sprintf(pathAssignedRole, url.PathEscape(assignedRoleID)))
}
