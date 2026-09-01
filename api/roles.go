package api

import (
	"fmt"
	"net/url"
)

const roleFields = "id,name,description,permissions(id,name,key)"

func (c *Client) GetRole(id string) (*Role, error) {
	var result Role
	path := fmt.Sprintf(pathRole+"?fields="+roleFields, url.PathEscape(id))
	err := c.Get(path, &result)
	return &result, err
}

func (c *Client) AddRolePermission(roleID, permID string) error {
	role, err := c.GetRole(roleID)
	if err != nil {
		return err
	}
	perms := make([]map[string]any, 0, len(role.Permissions)+1)
	for _, p := range role.Permissions {
		perms = append(perms, map[string]any{"id": p.ID})
	}
	for _, p := range role.Permissions {
		if p.ID == permID {
			return nil // already present
		}
	}
	perms = append(perms, map[string]any{"id": permID})
	return c.Post(fmt.Sprintf(pathRole, url.PathEscape(roleID)), map[string]any{"permissions": perms}, nil)
}

func (c *Client) RemoveRolePermission(roleID, permID string) error {
	role, err := c.GetRole(roleID)
	if err != nil {
		return err
	}
	perms := make([]map[string]any, 0, len(role.Permissions))
	for _, p := range role.Permissions {
		if p.ID != permID {
			perms = append(perms, map[string]any{"id": p.ID})
		}
	}
	return c.Post(fmt.Sprintf(pathRole, url.PathEscape(roleID)), map[string]any{"permissions": perms}, nil)
}

func (c *Client) ListRoles() ([]Role, error) {
	var result []Role
	err := c.Get(pathRoles+"?fields="+roleFields+"&$top=500", &result)
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
	err := c.Get(pathPermissions+"?fields=id,name,description,key&$top=500", &result)
	return result, err
}

func roleScopeBody(projectID, orgID string) map[string]any {
	switch {
	case projectID != "":
		return map[string]any{
			"$type":   "ProjectScope",
			"project": map[string]any{"id": projectID},
		}
	case orgID != "":
		return map[string]any{
			"$type":        "OrganizationScope",
			"organization": map[string]any{"id": orgID},
		}
	default:
		return map[string]any{"$type": "GlobalScope"}
	}
}

func (c *Client) grantRole(holderID, holderType, roleID, projectID, orgID string) error {
	return c.Post(pathAssignedRoles, map[string]any{
		"holder": map[string]any{"id": holderID, "$type": holderType},
		"role":   map[string]any{"id": roleID},
		"scope":  roleScopeBody(projectID, orgID),
	}, nil)
}

func (c *Client) GrantRoleToUser(userID, roleID, projectID, orgID string) error {
	return c.grantRole(userID, "User", roleID, projectID, orgID)
}

func (c *Client) GrantRoleToGroup(groupID, roleID, projectID, orgID string) error {
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
	return c.grantRole(groupID, groupType, roleID, projectID, orgID)
}

const assignedRoleFields = "id,role(id,name,description),holder($type,id,name,login,email)," +
	"scope($type,id,project(id,name,shortName),organization(id,name))"

func (c *Client) ListAssignedRoles(holderID string) ([]AssignedRole, error) {
	path := pathAssignedRoles + "?fields=" + assignedRoleFields + "&$top=500"
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
