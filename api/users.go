package api

import (
	"fmt"
	"net/url"
)

const userFields = "id,login,fullName,email,guest,online"

// AddUserRequest is the body for creating a user.
type AddUserRequest struct {
	Login    string `json:"login"`
	FullName string `json:"fullName"`
	Email    string `json:"email"`
}

func (c *Client) ListUsers(query string) ([]User, error) {
	var result []User
	path := fmt.Sprintf("/api/users?query=%s&fields=%s&$top=-1", url.QueryEscape(query), userFields)
	return result, c.Get(path, &result)
}

// GetUser finds a user by exact login match.
func (c *Client) GetUser(login string) (*User, error) {
	users, err := c.ListUsers(login)
	if err != nil {
		return nil, err
	}
	for i := range users {
		if users[i].Login == login {
			return &users[i], nil
		}
	}
	return nil, fmt.Errorf("user '%s' not found", login)
}

func (c *Client) GetCurrentUser() (*User, error) {
	var result User
	return &result, c.Get("/api/users/me?fields="+userFields, &result)
}

func (c *Client) AddUser(req AddUserRequest) (*User, error) {
	var result User
	return &result, c.Post("/api/users?fields="+userFields, req, &result)
}

func (c *Client) DeleteUser(id string) error {
	return c.Delete(fmt.Sprintf("/api/users/%s", url.PathEscape(id)))
}

func (c *Client) BanUser(id string, ban bool) error {
	return c.Post(fmt.Sprintf("/api/users/%s", url.PathEscape(id)), map[string]any{"banned": ban}, nil)
}

func (c *Client) GetUserTimezone(id string) (string, error) {
	var result struct {
		Timezone struct {
			ID           string `json:"id"`
			Presentation string `json:"presentation"`
		} `json:"timezone"`
	}
	path := fmt.Sprintf("/api/users/%s/profiles/general?fields=timezone(id,presentation)", url.PathEscape(id))
	if err := c.Get(path, &result); err != nil {
		return "", err
	}
	if result.Timezone.Presentation != "" {
		return result.Timezone.Presentation, nil
	}
	return result.Timezone.ID, nil
}

func (c *Client) SetUserTimezone(id, timezoneID string) error {
	path := fmt.Sprintf("/api/users/%s/profiles/general", url.PathEscape(id))
	return c.Post(path, map[string]any{"timezone": map[string]any{"id": timezoneID}}, nil)
}

func (c *Client) SetUserLocale(id, locale string) error {
	path := fmt.Sprintf("/api/users/%s/profiles/general", url.PathEscape(id))
	return c.Post(path, map[string]any{"locale": map[string]any{"locale": locale}}, nil)
}

func (c *Client) GetUserIssueDrafts(id string) ([]IssueDraft, error) {
	var result []IssueDraft
	path := fmt.Sprintf("/api/users/%s/drafts?fields=id,summary,created&$top=-1", url.PathEscape(id))
	return result, c.Get(path, &result)
}

func (c *Client) DeleteAllUserDrafts(id string) error {
	path := fmt.Sprintf("/api/users/%s/drafts", url.PathEscape(id))
	return c.Put(path, []any{}, nil)
}

func (c *Client) GetUserArticleDrafts(id string) ([]ArticleDraft, error) {
	var result []ArticleDraft
	path := fmt.Sprintf("/api/users/%s/articleDrafts?fields=id,summary&$top=-1", url.PathEscape(id))
	return result, c.Get(path, &result)
}

func (c *Client) PinProjectForUser(userID, projectID string) error {
	path := fmt.Sprintf("/api/users/%s/pinnedProjects", url.PathEscape(userID))
	return c.Post(path, map[string]any{"id": projectID}, nil)
}

func (c *Client) SetUserGeneralProfile(id string, body map[string]any) error {
	return c.Post(fmt.Sprintf("/api/users/%s/profiles/general", url.PathEscape(id)), body, nil)
}

func (c *Client) SetUserProfileProperty(id, key, value string) error {
	path := fmt.Sprintf("/api/users/%s/profiles/propertiesProfile/properties", url.PathEscape(id))
	return c.Post(path, map[string]any{"key": key, "value": value}, nil)
}

func (c *Client) SetUserAppearanceSetting(id string, body map[string]any) error {
	return c.Post(fmt.Sprintf("/api/users/%s/profiles/appearance", url.PathEscape(id)), body, nil)
}

func (c *Client) SetUserNotificationsProfile(id string, body map[string]any) error {
	return c.Post(fmt.Sprintf("/api/users/%s/profiles/notifications", url.PathEscape(id)), body, nil)
}

func (c *Client) SetUserGrazieProfile(id string, body map[string]any) error {
	return c.Post(fmt.Sprintf("/api/users/%s/profiles/grazie", url.PathEscape(id)), body, nil)
}

func (c *Client) SetUserArticlesProfile(id string, body map[string]any) error {
	return c.Post(fmt.Sprintf("/api/users/%s/profiles/articles", url.PathEscape(id)), body, nil)
}

func (c *Client) SetUserIssuesListSetting(id string, body map[string]any) error {
	return c.Post(fmt.Sprintf("/api/users/%s/profiles/issuesList", url.PathEscape(id)), body, nil)
}

func (c *Client) SetUserProfileTips(id string, tips map[string]bool) error {
	body := make(map[string]any, len(tips))
	for k, v := range tips {
		body[k] = v
	}
	return c.Post(fmt.Sprintf("/api/users/%s/profiles/tips", url.PathEscape(id)), body, nil)
}

// RawRequest makes a raw authenticated API request and returns the decoded JSON result.
func (c *Client) RawRequest(method, path string, body map[string]any) (any, error) {
	var result any
	switch method {
	case "GET":
		return result, c.Get(path, &result)
	case "DELETE":
		return nil, c.Delete(path)
	case "PUT":
		return result, c.Put(path, body, &result)
	default:
		return result, c.Post(path, body, &result)
	}
}
