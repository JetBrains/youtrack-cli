package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

var hubHTTPClient = &http.Client{Timeout: defaultTimeout, CheckRedirect: sameOriginRedirect}

// hubVerboseOut, when non-nil, receives verbose Hub HTTP log lines (same role as Client.Verbose).
var hubVerboseOut io.Writer

// SetHubVerbose sets the writer for Hub API verbose output. Pass nil to disable.
// Call this before Hub API operations and defer SetHubVerbose(nil) to clean up.
func SetHubVerbose(w io.Writer) { hubVerboseOut = w }

// HubUser is the authenticated user record from Hub.
type HubUser struct {
	ID    string `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
}

// HubServiceID is the ID of Hub's own service; an OAuth token needs it in scope to call the Hub REST API.
const HubServiceID = "0-0-0-0-0"

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

// OAuthProtectedResource is the response from /.well-known/oauth-protected-resource/mcp.
// It identifies which Hub authorization server the YouTrack instance uses.
type OAuthProtectedResource struct {
	Resource             string   `json:"resource"`
	AuthorizationServers []string `json:"authorization_servers"`
	ScopesSupported      []string `json:"scopes_supported"`
}

// GetOAuthProtectedResourceContext fetches OAuth protected resource metadata from the YouTrack
// instance. This reveals which Hub authorization server it is configured to use.
func GetOAuthProtectedResourceContext(ctx context.Context, baseURL string) (*OAuthProtectedResource, error) {
	endpoint := strings.TrimRight(baseURL, "/") + "/.well-known/oauth-protected-resource/mcp"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := hubHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching oauth-protected-resource: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodySize))
	if err != nil {
		return nil, fmt.Errorf("reading oauth-protected-resource response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data))
	}
	var r OAuthProtectedResource
	err = json.Unmarshal(data, &r)
	return &r, err
}

// DetectHubURLContext discovers the Hub URL from YouTrack's oauth-protected-resource metadata.
// Returns the hub URL and whether Hub is internal (same host as YouTrack) or external.
func DetectHubURLContext(ctx context.Context, baseURL string) (hubURL string, isInternal bool, err error) {
	resource, err := GetOAuthProtectedResourceContext(ctx, baseURL)
	if err != nil {
		return "", false, err
	}
	if len(resource.AuthorizationServers) == 0 {
		return "", false, fmt.Errorf("no authorization servers in oauth-protected-resource response")
	}
	hubURL = strings.TrimRight(resource.AuthorizationServers[0], "/")

	ytParsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return "", false, err
	}
	if err := ValidateHubURL(baseURL, hubURL); err != nil {
		return "", false, err
	}
	hubParsed, err := url.Parse(hubURL)
	if err != nil {
		return "", false, err
	}
	isInternal = ytParsed.Hostname() == hubParsed.Hostname()
	return hubURL, isInternal, nil
}

// ValidateHubURL rejects a server-advertised Hub URL before it reaches the browser or a terminal hyperlink:
// it must be https (http only when the instance itself is http), have a host, and carry no userinfo.
func ValidateHubURL(baseURL, hubURL string) error {
	hub, err := url.Parse(hubURL)
	if err != nil || hub.Hostname() == "" || hub.User != nil {
		return fmt.Errorf("invalid hub url %q", hubURL)
	}
	switch strings.ToLower(hub.Scheme) {
	case "https":
		return nil
	case "http":
		if yt, err := url.Parse(strings.TrimSpace(baseURL)); err == nil && strings.EqualFold(yt.Scheme, "http") {
			return nil
		}
		return fmt.Errorf("hub url %q uses http but the instance uses https", hubURL)
	default:
		return fmt.Errorf("hub url %q must use https", hubURL)
	}
}

// IsInsecureRemoteHTTP reports whether rawURL uses plain http to a non-loopback host, where credentials travel in clear text.
func IsInsecureRemoteHTTP(rawURL string) bool {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || !strings.EqualFold(u.Scheme, "http") {
		return false
	}
	host := u.Hostname()
	if strings.EqualFold(host, "localhost") {
		return false
	}
	ip := net.ParseIP(host)
	return ip == nil || !ip.IsLoopback()
}

func hubBasicAuth(login, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(login+":"+password))
}

func hubDo(ctx context.Context, method, baseURL, path, auth string, body any) ([]byte, error) {
	// Hub calls bypass Client, so YOUTRACK_RO needs its own guard here (e.g. permanent-token creation).
	if method != http.MethodGet && readOnly() {
		return nil, errReadOnly
	}
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
	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", auth)
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if hubVerboseOut != nil {
		_, _ = fmt.Fprintf(hubVerboseOut, "> %s %s\n", method, baseURL+path)
	}
	resp, err := hubHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodySize))
	if err != nil {
		return nil, err
	}
	// Log status and size only — the token-creation body carries the permanent token in cleartext.
	if hubVerboseOut != nil {
		_, _ = fmt.Fprintf(hubVerboseOut, "< %d (%d bytes)\n", resp.StatusCode, len(data))
	}
	// Detect HTML responses — the Hub API endpoint is unreachable and returned a web page.
	// This typically means basic auth is not available for this instance (e.g. cloud/SSO-only).
	if strings.Contains(resp.Header.Get("Content-Type"), "text/html") ||
		bytes.HasPrefix(bytes.TrimSpace(data), []byte("<")) {
		return nil, fmt.Errorf("basic auth is not available for this instance; use the web browser flow instead")
	}
	if resp.StatusCode >= 300 {
		// Extract a human-readable message from the Hub JSON error body when available.
		var hubErr struct {
			ErrorDescription string `json:"error_description"`
			Error            string `json:"error"`
		}
		if jsonErr := json.Unmarshal(data, &hubErr); jsonErr == nil {
			if hubErr.ErrorDescription != "" {
				return nil, fmt.Errorf("%s (HTTP %d)", hubErr.ErrorDescription, resp.StatusCode)
			}
			if hubErr.Error != "" {
				return nil, fmt.Errorf("%s (HTTP %d)", hubErr.Error, resp.StatusCode)
			}
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}

// GetHubMeContext returns the currently authenticated Hub user.
func GetHubMeContext(ctx context.Context, baseURL, login, password string) (*HubUser, error) {
	data, err := hubDo(ctx, "GET", baseURL, "/hub/api/rest/users/me?fields=id,login,name",
		hubBasicAuth(login, password), nil)
	if err != nil {
		return nil, err
	}
	var u HubUser
	err = json.Unmarshal(data, &u)
	return &u, err
}

// ListHubServicesContext returns all services registered in Hub.
func ListHubServicesContext(ctx context.Context, baseURL, login, password string) ([]HubService, error) {
	data, err := hubDo(ctx, "GET", baseURL, "/hub/api/rest/services?fields=id,name&$top=500",
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
	err = json.Unmarshal(data, &list)
	return list, err
}

// CreateHubTokenContext creates a permanent token for the given Hub user.
func CreateHubTokenContext(ctx context.Context, baseURL, login, password, userID, tokenName string, scopeIDs []string) (*HubToken, error) {
	return createHubToken(ctx, baseURL+"/hub", url.PathEscape(userID), hubBasicAuth(login, password), tokenName, scopeIDs)
}

// CreateHubTokenWithAccessTokenContext creates a permanent token for the user behind an OAuth access token; the access token must carry Hub's own scope (HubServiceID).
func CreateHubTokenWithAccessTokenContext(ctx context.Context, hubURL, accessToken, tokenName string, scopeIDs []string) (*HubToken, error) {
	return createHubToken(ctx, strings.TrimRight(hubURL, "/"), "me", "Bearer "+accessToken, tokenName, scopeIDs)
}

func createHubToken(ctx context.Context, hubURL, userID, auth, tokenName string, scopeIDs []string) (*HubToken, error) {
	scope := make([]map[string]string, len(scopeIDs))
	for i, id := range scopeIDs {
		scope[i] = map[string]string{"id": id}
	}
	path := "/api/rest/users/" + userID + "/permanenttokens?fields=id,name,token,scope(id,name),creationTime"
	data, err := hubDo(ctx, "POST", hubURL, path, auth, map[string]any{"name": tokenName, "scope": scope})
	if err != nil {
		return nil, err
	}
	var tok HubToken
	err = json.Unmarshal(data, &tok)
	return &tok, err
}

// DeleteHubTokenContext deletes a permanent token by ID (used for test cleanup).
func DeleteHubTokenContext(ctx context.Context, baseURL, login, password, userID, tokenID string) error {
	path := fmt.Sprintf("/hub/api/rest/users/%s/permanenttokens/%s",
		url.PathEscape(userID), url.PathEscape(tokenID))
	_, err := hubDo(ctx, "DELETE", baseURL, path, hubBasicAuth(login, password), nil)
	return err
}
