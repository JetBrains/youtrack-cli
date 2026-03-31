package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

var hubHTTPClient = &http.Client{}

// HubUser is the authenticated user record from Hub.
type HubUser struct {
	ID    string `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
}

// HubService is a registered Hub service (used for token scope).
type HubService struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// HubToken is a Hub permanent token.
type HubToken struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Token        string       `json:"token"`
	Scope        []HubService `json:"scope"`
	CreationTime int64        `json:"creationTime"`
}

func hubBasicAuth(login, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(login+":"+password))
}

func hubDo(method, baseURL, path, auth string, body any) ([]byte, error) {
	var bodyReader io.Reader
	contentType := ""
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
		contentType = "application/json"
	}
	req, err := http.NewRequest(method, baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", auth)
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := hubHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Hub API HTTP %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}

// GetHubMe returns the currently authenticated Hub user.
func GetHubMe(baseURL, login, password string) (*HubUser, error) {
	data, err := hubDo("GET", baseURL, "/hub/api/rest/users/me?fields=id,login,name",
		hubBasicAuth(login, password), nil)
	if err != nil {
		return nil, err
	}
	var u HubUser
	return &u, json.Unmarshal(data, &u)
}

// ListHubServices returns all services registered in Hub.
func ListHubServices(baseURL, login, password string) ([]HubService, error) {
	data, err := hubDo("GET", baseURL, "/hub/api/rest/services?fields=id,name&$top=-1",
		hubBasicAuth(login, password), nil)
	if err != nil {
		return nil, err
	}
	// Hub may return {"services":[...]} or a bare array.
	var wrapper struct {
		Services []HubService `json:"services"`
	}
	if err := json.Unmarshal(data, &wrapper); err == nil {
		return wrapper.Services, nil
	}
	var list []HubService
	return list, json.Unmarshal(data, &list)
}

// CreateHubToken creates a permanent token for the given Hub user.
func CreateHubToken(baseURL, login, password, userID, tokenName string, scopeIDs []string) (*HubToken, error) {
	scope := make([]map[string]string, len(scopeIDs))
	for i, id := range scopeIDs {
		scope[i] = map[string]string{"id": id}
	}
	path := fmt.Sprintf(
		"/hub/api/rest/users/%s/permanenttokens?fields=id,name,token,scope(id,name),creationTime",
		url.PathEscape(userID),
	)
	data, err := hubDo("POST", baseURL, path, hubBasicAuth(login, password),
		map[string]any{"name": tokenName, "scope": scope})
	if err != nil {
		return nil, err
	}
	var tok HubToken
	return &tok, json.Unmarshal(data, &tok)
}

// DeleteHubToken deletes a permanent token by ID (used for test cleanup).
func DeleteHubToken(baseURL, login, password, userID, tokenID string) error {
	path := fmt.Sprintf("/hub/api/rest/users/%s/permanenttokens/%s",
		url.PathEscape(userID), url.PathEscape(tokenID))
	_, err := hubDo("DELETE", baseURL, path, hubBasicAuth(login, password), nil)
	return err
}
