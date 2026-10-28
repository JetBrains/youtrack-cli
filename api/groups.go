package api

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const groupDeleteSyncTimeout = 30 * time.Second

func (c *Client) ListGroups(query string) ([]Group, error) {
	var result []Group
	path := pathGroups + fmt.Sprintf("?fields=id,name&searchQuery=%s&$top=500", url.QueryEscape(query))
	err := c.Get(path, &result)
	return result, err
}

func (c *Client) GetGroup(identifier string) (*Group, error) {
	var result Group
	path := fmt.Sprintf(pathGroup+"?fields=id,name", url.PathEscape(identifier))
	if err := c.Get(path, &result); err == nil {
		return &result, nil
	} else {
		var apiErr *APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != 404 {
			return nil, err
		}
	}

	groups, err := c.ListGroups(identifier)
	if err != nil {
		return nil, err
	}
	for i := range groups {
		if groups[i].Name == identifier || groups[i].ID == identifier {
			return &groups[i], nil
		}
	}
	return nil, fmt.Errorf("group '%s' %w", identifier, ErrNotFound)
}

func (c *Client) AddGroup(name string) (*Group, error) {
	var result Group
	err := c.Post(pathGroups+"?fields=id,name", map[string]any{"name": name}, &result)
	return &result, err
}

func (c *Client) ListGroupMembers(groupID string) ([]User, error) {
	var result []User
	path := fmt.Sprintf(pathGroupMembers+"?fields="+userFields+"&$top=500", url.PathEscape(groupID))
	err := c.Get(path, &result)
	return result, err
}

func (c *Client) AddGroupMember(groupID, userID string) error {
	path := fmt.Sprintf(pathGroupMembers, url.PathEscape(groupID))
	return c.Post(path, map[string]any{"id": userID}, nil)
}

func (c *Client) RemoveGroupMember(groupID, userID string) error {
	return c.Delete(fmt.Sprintf(pathGroupMember, url.PathEscape(groupID), url.PathEscape(userID)))
}

func (c *Client) ListGroupProjectRoles(groupID string) ([]AssignedRole, error) {
	return c.ListAssignedRoles(groupID)
}

func (c *Client) DeleteGroup(identifier, successorIdentifier string) error {
	group, err := c.GetGroup(identifier)
	if err != nil {
		return err
	}
	path := fmt.Sprintf(pathGroup, url.PathEscape(group.ID))
	if successorIdentifier != "" {
		successor, err := c.GetGroup(successorIdentifier)
		if err != nil {
			return err
		}
		if group.ID == successor.ID {
			return fmt.Errorf("successor must be different from group '%s'", identifier)
		}
		path += "?successor=" + url.QueryEscape(successor.ID)
	}
	if err := c.Delete(path); err != nil {
		return err
	}
	return c.waitForGroupDeletion(group.Name, group.ID)
}

func (c *Client) waitForGroupDeletion(name, id string) error {
	deadline := time.Now().Add(groupDeleteSyncTimeout)
	for {
		var result Group
		err := c.Get(fmt.Sprintf(pathGroup+"?fields=id,name", url.PathEscape(id)), &result)
		switch {
		case err == nil:
			// Still resolvable by exact id, so the merge/delete has not finished yet.
		case isGroupDeletionObserved(err):
			return nil
		default:
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("group '%s' is still resolvable after delete; YouTrack sync may still be in progress", name)
		}
		if err := c.wait(entityDeletePollInterval); err != nil {
			return err
		}
	}
}

func isGroupDeletionObserved(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "HTTP 404") || strings.Contains(message, "was removed")
}
