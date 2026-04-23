package api

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

const groupDeleteSyncTimeout = 30 * time.Second

func (c *Client) ListGroups(query string) ([]Group, error) {
	var result []Group
	path := pathGroups + fmt.Sprintf("?fields=id,name&query=%s&$top=-1", url.QueryEscape(query))
	err := c.Get(path, &result)
	return result, err
}

func (c *Client) GetGroup(identifier string) (*Group, error) {
	var result Group
	path := fmt.Sprintf(pathGroup+"?fields=id,name", url.PathEscape(identifier))
	if err := c.Get(path, &result); err == nil {
		return &result, nil
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
	return nil, fmt.Errorf("group '%s' not found", identifier)
}

func (c *Client) AddGroup(name string) (*Group, error) {
	var result Group
	err := c.Post(pathGroups+"?fields=id,name", map[string]any{"name": name}, &result)
	return &result, err
}

func (c *Client) DeleteGroup(identifier, successorIdentifier string) error {
	group, err := c.GetGroup(identifier)
	if err != nil {
		return err
	}
	successor, err := c.GetGroup(successorIdentifier)
	if err != nil {
		return err
	}
	if group.ID == successor.ID {
		return fmt.Errorf("successor must be different from group '%s'", identifier)
	}
	path := fmt.Sprintf(pathGroup, url.PathEscape(group.ID)) + "?successor=" + url.QueryEscape(successor.ID)
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
		time.Sleep(entityDeletePollInterval)
	}
}

func isGroupDeletionObserved(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "HTTP 404") || strings.Contains(message, "was removed")
}
