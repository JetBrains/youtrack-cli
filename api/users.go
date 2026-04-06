package api

import (
	"fmt"
	"net/url"
	"time"
)

const userFields = "id,login,fullName,email,guest,online"

const (
	entityDeleteSyncTimeout  = 5 * time.Second
	entityDeletePollInterval = 100 * time.Millisecond
)

// AddUserRequest is the body for creating a user.
type AddUserRequest struct {
	Login    string `json:"login"`
	FullName string `json:"fullName"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (c *Client) ListUsers(query string) ([]User, error) {
	var result []User
	path := pathUsers + fmt.Sprintf("?query=%s&fields=%s&$top=-1", url.QueryEscape(query), userFields)
	err := c.Get(path, &result)
	return result, err
}

// GetUser finds a user by exact login match.
func (c *Client) GetUser(login string) (*User, error) {
	var result User
	path := fmt.Sprintf(pathUser+"?fields=%s", url.PathEscape(login), userFields)
	if err := c.Get(path, &result); err == nil {
		return &result, nil
	}

	users, err := c.ListUsers(login)
	if err != nil {
		return nil, err
	}
	for i := range users {
		if users[i].Login == login || users[i].ID == login {
			return &users[i], nil
		}
	}
	return nil, fmt.Errorf("user '%s' not found", login)
}

func (c *Client) GetCurrentUser() (*User, error) {
	var result User
	err := c.Get(pathUserMe+"?fields="+userFields, &result)
	return &result, err
}

func (c *Client) AddUser(req AddUserRequest) (*User, error) {
	var result User
	err := c.Post(pathUsers+"?fields="+userFields, req, &result)
	return &result, err
}

func (c *Client) DeleteUser(identifier, successorIdentifier string) error {
	user, err := c.GetUser(identifier)
	if err != nil {
		return err
	}
	successor, err := c.GetUser(successorIdentifier)
	if err != nil {
		return err
	}
	if user.ID == successor.ID {
		return fmt.Errorf("successor must be different from user '%s'", identifier)
	}
	path := fmt.Sprintf(pathUser, url.PathEscape(user.ID)) + "?successor=" + url.QueryEscape(successor.ID)
	if err := c.Delete(path); err != nil {
		return err
	}
	return c.waitForUserDeletion(user.Login, user.ID)
}

func (c *Client) BanUser(identifier string, ban bool) error {
	user, err := c.GetUser(identifier)
	if err != nil {
		return err
	}
	return c.Post(fmt.Sprintf(pathUser, url.PathEscape(user.ID)), map[string]any{"banned": ban}, nil)
}

func (c *Client) GetUserTimezone(id string) (string, error) {
	var result struct {
		Timezone struct {
			ID           string `json:"id"`
			Presentation string `json:"presentation"`
		} `json:"timezone"`
	}
	path := fmt.Sprintf(pathUserProfileGeneral+"?fields=timezone(id,presentation)", url.PathEscape(id))
	if err := c.Get(path, &result); err != nil {
		return "", err
	}
	if result.Timezone.Presentation != "" {
		return result.Timezone.Presentation, nil
	}
	return result.Timezone.ID, nil
}

func (c *Client) SetUserTimezone(id, timezoneID string) error {
	path := fmt.Sprintf(pathUserProfileGeneral, url.PathEscape(id))
	return c.Post(path, map[string]any{"timezone": map[string]any{"id": timezoneID}}, nil)
}

func (c *Client) SetUserLocale(id, locale string) error {
	path := fmt.Sprintf(pathUserProfileGeneral, url.PathEscape(id))
	return c.Post(path, map[string]any{"locale": map[string]any{"locale": locale}}, nil)
}

func (c *Client) GetUserIssueDrafts(id string) ([]IssueDraft, error) {
	var result []IssueDraft
	path := fmt.Sprintf(pathUserDrafts+"?fields=id,summary,created&$top=-1", url.PathEscape(id))
	err := c.Get(path, &result)
	return result, err
}

func (c *Client) DeleteAllUserDrafts(id string) error {
	path := fmt.Sprintf(pathUserDrafts, url.PathEscape(id))
	return c.Put(path, []any{}, nil)
}

func (c *Client) GetUserArticleDrafts(id string) ([]ArticleDraft, error) {
	var result []ArticleDraft
	path := fmt.Sprintf(pathUserArticleDrafts+"?fields=id,summary&$top=-1", url.PathEscape(id))
	err := c.Get(path, &result)
	return result, err
}

func (c *Client) PinProjectForUser(userID, projectID string) error {
	path := fmt.Sprintf(pathUserPinnedProjects, url.PathEscape(userID))
	return c.Post(path, map[string]any{"id": projectID}, nil)
}

func (c *Client) SetUserGeneralProfile(id string, body map[string]any) error {
	return c.Post(fmt.Sprintf(pathUserProfileGeneral, url.PathEscape(id)), body, nil)
}

func (c *Client) SetUserProfileProperty(id, key, value string) error {
	path := fmt.Sprintf(pathUserProfileProperties, url.PathEscape(id))
	return c.Post(path, map[string]any{"key": key, "value": value}, nil)
}

func (c *Client) SetUserAppearanceSetting(id string, body map[string]any) error {
	return c.Post(fmt.Sprintf(pathUserProfileAppearance, url.PathEscape(id)), body, nil)
}

func (c *Client) SetUserNotificationsProfile(id string, body map[string]any) error {
	return c.Post(fmt.Sprintf(pathUserProfileNotifications, url.PathEscape(id)), body, nil)
}

func (c *Client) SetUserGrazieProfile(id string, body map[string]any) error {
	return c.Post(fmt.Sprintf(pathUserProfileGrazie, url.PathEscape(id)), body, nil)
}

func (c *Client) SetUserArticlesProfile(id string, body map[string]any) error {
	return c.Post(fmt.Sprintf(pathUserProfileArticles, url.PathEscape(id)), body, nil)
}

func (c *Client) SetUserIssuesListSetting(id string, body map[string]any) error {
	return c.Post(fmt.Sprintf(pathUserProfileIssuesList, url.PathEscape(id)), body, nil)
}

func (c *Client) SetUserProfileTips(id string, tips map[string]bool) error {
	body := make(map[string]any, len(tips))
	for k, v := range tips {
		body[k] = v
	}
	return c.Post(fmt.Sprintf(pathUserProfileTips, url.PathEscape(id)), body, nil)
}

// RawRequest makes a raw authenticated API request and returns the decoded JSON result.
func (c *Client) RawRequest(method, path string, body map[string]any) (any, error) {
	var result any
	var err error
	switch method {
	case "GET":
		err = c.Get(path, &result)
	case "DELETE":
		return nil, c.Delete(path)
	case "PUT":
		err = c.Put(path, body, &result)
	default:
		err = c.Post(path, body, &result)
	}
	return result, err
}

func (c *Client) waitForUserDeletion(login, id string) error {
	deadline := time.Now().Add(entityDeleteSyncTimeout)
	for {
		users, err := c.ListUsers(login)
		if err != nil {
			return err
		}

		found := false
		for i := range users {
			if users[i].Login == login || users[i].ID == id {
				found = true
				break
			}
		}
		if !found {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("user '%s' is still visible after delete; YouTrack sync may still be in progress", login)
		}
		time.Sleep(entityDeletePollInterval)
	}
}
