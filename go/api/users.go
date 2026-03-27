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
	path := fmt.Sprintf("/api/users?query=%s&fields=%s", url.QueryEscape(query), userFields)
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
