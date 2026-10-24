package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	oauthHTTPTimeout         = 30 * time.Second
	maxOAuthResponseBodySize = 1 << 20 // 1 MiB
)

// noRedirectClient never follows redirects, preventing sensitive OAuth request bodies from being forwarded to another origin.
var noRedirectClient = &http.Client{
	Timeout: oauthHTTPTimeout,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// RingConfig is the ring(...) fragment of /api/config: the Hub authorization server and YouTrack service ID, reachable unauthenticated on all versions.
type RingConfig struct {
	URL            string `json:"url"`
	ServiceID      string `json:"serviceId"`
	HasEmbeddedHub bool   `json:"hasEmbeddedHub"`
}

// GetRingConfigContext fetches Hub connection details from YouTrack's frontend config.
func GetRingConfigContext(ctx context.Context, baseURL string) (*RingConfig, error) {
	endpoint := strings.TrimRight(baseURL, "/") + "/api/config?fields=ring(url,serviceId,hasEmbeddedHub)"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := hubHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching ring config: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxOAuthResponseBodySize))
	if err != nil {
		return nil, fmt.Errorf("reading ring config response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data))
	}
	var wrapper struct {
		Ring RingConfig `json:"ring"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("parsing ring config: %w", err)
	}
	return &wrapper.Ring, nil
}

// OAuthEndpoints holds the Hub OAuth 2.0 endpoints from /.well-known/openid-configuration, with a conventional-path fallback for older Hubs.
type OAuthEndpoints struct {
	Issuer                string   `json:"issuer"`
	AuthorizationEndpoint string   `json:"authorization_endpoint"`
	TokenEndpoint         string   `json:"token_endpoint"`
	CodeChallengeMethods  []string `json:"code_challenge_methods_supported"`
}

// DiscoverOAuthEndpointsContext resolves a Hub's authorization and token endpoints; if it advertises code_challenge_methods_supported, S256 must be among them.
func DiscoverOAuthEndpointsContext(ctx context.Context, hubURL string) (*OAuthEndpoints, error) {
	hubURL = strings.TrimRight(hubURL, "/")
	fallback := &OAuthEndpoints{
		AuthorizationEndpoint: hubURL + "/api/rest/oauth2/auth",
		TokenEndpoint:         hubURL + "/api/rest/oauth2/token",
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, hubURL+"/.well-known/openid-configuration", nil)
	if err != nil {
		return fallback, nil
	}
	resp, err := hubHTTPClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return fallback, nil
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxOAuthResponseBodySize))
	if err != nil || resp.StatusCode >= 300 {
		return fallback, nil
	}
	var eps OAuthEndpoints
	if err := json.Unmarshal(data, &eps); err != nil ||
		eps.AuthorizationEndpoint == "" || eps.TokenEndpoint == "" {
		return fallback, nil
	}
	for name, endpoint := range map[string]string{
		"issuer":                 eps.Issuer,
		"authorization_endpoint": eps.AuthorizationEndpoint,
		"token_endpoint":         eps.TokenEndpoint,
	} {
		if endpoint == "" && name == "issuer" {
			continue
		}
		if err := validateOAuthEndpoint(hubURL, endpoint, name); err != nil {
			return nil, err
		}
	}
	if len(eps.CodeChallengeMethods) > 0 {
		s256 := false
		for _, m := range eps.CodeChallengeMethods {
			if m == "S256" {
				s256 = true
				break
			}
		}
		if !s256 {
			return nil, fmt.Errorf("hub at %s does not support PKCE S256 (supported: %s)",
				hubURL, strings.Join(eps.CodeChallengeMethods, ", "))
		}
	}
	return &eps, nil
}

// validateOAuthEndpoint prevents discovery metadata from sending the authorization code or verifier to another origin.
func validateOAuthEndpoint(hubURL, endpoint, name string) error {
	hub, err := url.Parse(hubURL)
	if err != nil || hub.Scheme == "" || hub.Host == "" {
		return fmt.Errorf("invalid hub url %q", hubURL)
	}
	discovered, err := url.Parse(endpoint)
	if err != nil || discovered.Scheme == "" || discovered.Host == "" {
		return fmt.Errorf("invalid oauth %s %q", name, endpoint)
	}
	if !strings.EqualFold(hub.Scheme, discovered.Scheme) || !strings.EqualFold(hub.Host, discovered.Host) {
		return fmt.Errorf("oauth %s origin %s://%s does not match hub origin %s://%s",
			name, discovered.Scheme, discovered.Host, hub.Scheme, hub.Host)
	}
	return nil
}

// DefaultOAuthClientID returns the client ID the CLI signs in with when none is given: "localhost-auth" on hub.jetbrains.com, "youtrack-cli" on every other Hub.
func DefaultOAuthClientID(hubURL string) string {
	if u, err := url.Parse(hubURL); err == nil && strings.EqualFold(u.Hostname(), "hub.jetbrains.com") {
		return "localhost-auth"
	}
	return "youtrack-cli"
}

// PreflightOAuthClientContext checks, without opening a browser, whether Hub accepts the client_id and redirect_uri in authorizeURL (303 to /auth/oauth/error means rejected).
func PreflightOAuthClientContext(ctx context.Context, authorizeURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, authorizeURL, nil)
	if err != nil {
		return err
	}
	resp, err := noRedirectClient.Do(req)
	if err != nil {
		return fmt.Errorf("preflight authorization request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	loc := resp.Header.Get("Location")
	switch {
	case resp.StatusCode >= 300 && resp.StatusCode < 400 && strings.Contains(loc, "/auth/oauth/error"):
		return fmt.Errorf("hub rejected the oauth client or redirect uri")
	case resp.StatusCode >= 400:
		return fmt.Errorf("authorization endpoint returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// OAuthToken is the Hub token endpoint response for the authorization-code and refresh-token grants.
type OAuthToken struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token"`
}

// OAuthError is an RFC 6749 §5.2 error response from the token endpoint; it carries only the server's error code and description, never request credentials.
type OAuthError struct {
	StatusCode  int
	Code        string
	Description string
}

func (e *OAuthError) Error() string {
	msg := e.Code
	if e.Description != "" {
		msg = e.Description
	}
	if msg == "" {
		msg = "token request failed"
	}
	// The CLI has no secret to send. Hub answers invalid_client when the client lacks the flow, and invalid_grant
	// "Invalid service secret" when the client requires a secret.
	if e.Code == "invalid_client" || e.Code == "unauthorized_client" || e.isServiceSecretError() {
		msg += " — in YouTrack, open Administration > Access Management > OAuth Clients, select the client, and on its Authentication tab enable Authorization Code flow and Require PKCE with no Client secret (with an external Hub, configure the service there)"
	}
	return fmt.Sprintf("%s (HTTP %d)", msg, e.StatusCode)
}

// isServiceSecretError reports Hub's invalid_grant "Invalid service secret", a client-configuration error rather than a dead grant.
func (e *OAuthError) isServiceSecretError() bool {
	return e.Code == "invalid_grant" && strings.Contains(e.Description, "service secret")
}

// IsInvalidGrant reports whether err means the grant (authorization code or refresh token) is invalid, expired, or revoked.
// Hub reports an expired refresh token with its own expired_refresh_token code instead of invalid_grant.
func IsInvalidGrant(err error) bool {
	var oauthErr *OAuthError
	if !errors.As(err, &oauthErr) || oauthErr.isServiceSecretError() {
		return false
	}
	return oauthErr.Code == "invalid_grant" || oauthErr.Code == "expired_refresh_token"
}

// ExchangeOAuthCodeContext redeems an authorization code as a public client — client_id in the form body, no secret, PKCE code_verifier for proof of possession.
func ExchangeOAuthCodeContext(ctx context.Context, tokenEndpoint, clientID, code, redirectURI, codeVerifier string) (*OAuthToken, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("client_id", clientID)
	form.Set("code_verifier", codeVerifier)
	return postTokenForm(ctx, tokenEndpoint, form)
}

// RefreshOAuthTokenContext trades a refresh token for a new access token as a public client; Hub requires scope on refresh and may rotate the refresh token.
func RefreshOAuthTokenContext(ctx context.Context, tokenEndpoint, clientID, refreshToken, scope string) (*OAuthToken, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", clientID)
	if scope != "" {
		form.Set("scope", scope)
	}
	return postTokenForm(ctx, tokenEndpoint, form)
}

// postForm sends an OAuth form POST over noRedirectClient (never forwarding the body to another origin).
func postForm(ctx context.Context, endpoint string, form url.Values) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	if hubVerboseOut != nil {
		// The form body carries codes and tokens, so only the method and endpoint are logged.
		_, _ = fmt.Fprintf(hubVerboseOut, "> %s %s\n", http.MethodPost, endpoint)
	}
	resp, err := noRedirectClient.Do(req)
	if err != nil {
		return nil, err
	}
	if hubVerboseOut != nil {
		_, _ = fmt.Fprintf(hubVerboseOut, "< %d\n", resp.StatusCode)
	}
	return resp, nil
}

func postTokenForm(ctx context.Context, tokenEndpoint string, form url.Values) (*OAuthToken, error) {
	resp, err := postForm(ctx, tokenEndpoint, form)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxOAuthResponseBodySize))
	if err != nil {
		return nil, fmt.Errorf("reading token response: %w", err)
	}
	return parseTokenResponse(resp.StatusCode, data)
}

// parseTokenResponse decodes a token endpoint response; error messages never echo a success body, which holds the tokens.
func parseTokenResponse(status int, data []byte) (*OAuthToken, error) {
	if status >= 300 {
		var hubErr struct {
			ErrorDescription string `json:"error_description"`
			Error            string `json:"error"`
		}
		if jsonErr := json.Unmarshal(data, &hubErr); jsonErr == nil && (hubErr.Error != "" || hubErr.ErrorDescription != "") {
			return nil, &OAuthError{StatusCode: status, Code: hubErr.Error, Description: hubErr.ErrorDescription}
		}
		body := string(data)
		if len(body) > 200 {
			body = body[:200] + "..."
		}
		return nil, fmt.Errorf("HTTP %d: %s", status, body)
	}
	var tok OAuthToken
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, errors.New("parsing token response: response is not valid JSON")
	}
	if tok.AccessToken == "" {
		return nil, errors.New("token response contained no access_token")
	}
	// Tokens are sent as "Authorization: Bearer"; any other type would be misused.
	if tok.TokenType != "" && !strings.EqualFold(tok.TokenType, "bearer") {
		return nil, fmt.Errorf("token response has unsupported token_type %q", tok.TokenType)
	}
	if tok.ExpiresIn <= 0 {
		return nil, fmt.Errorf("token response contained invalid expires_in %d", tok.ExpiresIn)
	}
	return &tok, nil
}

// TokenExpiry converts expires_in into an absolute unix time anchored at receivedAt (taken before the request, so the estimate errs early).
func TokenExpiry(receivedAt time.Time, expiresIn int64) int64 {
	return receivedAt.Unix() + expiresIn
}

// TokenNeedsRefresh reports whether a token expiring at expiresAt (unix; 0 = never) is expired or within skew of expiring.
// The skew absorbs modest clock drift between this machine and Hub as well as request latency.
func TokenNeedsRefresh(expiresAt int64, now time.Time, skew time.Duration) bool {
	if expiresAt == 0 {
		return false
	}
	return now.Add(skew).Unix() >= expiresAt
}

// AuthorizationRequest holds the validated components of a Hub authorization-code request.
type AuthorizationRequest struct {
	ClientID      string
	RedirectURI   string
	Scopes        []string
	State         string
	CodeChallenge string // S256 challenge; plain PKCE is never sent
	Offline       bool   // request a refresh token (access_type=offline)
}

// BuildAuthorizationURL encodes an authorization-code + PKCE (S256) request onto the Hub authorization endpoint.
// Every value goes through url.Values encoding, so user-supplied client IDs or scopes cannot inject parameters.
func BuildAuthorizationURL(endpoint string, r *AuthorizationRequest) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil || !u.IsAbs() || u.Host == "" || u.User != nil || u.Fragment != "" {
		return "", fmt.Errorf("invalid authorization endpoint %q", endpoint)
	}
	if !strings.EqualFold(u.Scheme, "https") && !strings.EqualFold(u.Scheme, "http") {
		return "", fmt.Errorf("authorization endpoint %q must use https", endpoint)
	}
	if r.ClientID == "" || r.RedirectURI == "" || r.State == "" || r.CodeChallenge == "" || len(r.Scopes) == 0 {
		return "", errors.New("authorization request is missing client_id, redirect_uri, scope, state, or code_challenge")
	}
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", r.ClientID)
	q.Set("redirect_uri", r.RedirectURI)
	q.Set("scope", strings.Join(r.Scopes, " "))
	q.Set("state", r.State)
	q.Set("code_challenge", r.CodeChallenge)
	q.Set("code_challenge_method", "S256")
	// access_type=offline makes Hub issue a refresh token on the first code exchange; request_credentials=default shows the login form only when the user has no Hub session.
	if r.Offline {
		q.Set("access_type", "offline")
	}
	q.Set("request_credentials", "default")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// ValidateOAuthURL requires an absolute https URL with a host and no user-info; plain http is accepted only for a loopback host when allowInsecureLoopback is set (local development).
func ValidateOAuthURL(rawURL string, allowInsecureLoopback bool) error {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || !u.IsAbs() || u.Hostname() == "" {
		return fmt.Errorf("%q is not an absolute URL — use the full address, e.g. https://example.youtrack.cloud", rawURL)
	}
	if u.User != nil {
		return fmt.Errorf("%q must not contain user info", rawURL)
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return nil
	case "http":
		if IsInsecureRemoteHTTP(rawURL) {
			return fmt.Errorf("refusing to sign in over plain http to %s — use the https URL", u.Host)
		}
		if !allowInsecureLoopback {
			return fmt.Errorf("%s uses plain http — use the https URL, or pass --insecure to allow http for a local (loopback) instance", rawURL)
		}
		return nil
	default:
		return fmt.Errorf("%q must use https", rawURL)
	}
}

// GeneratePKCEVerifier returns a 32-byte random code verifier as unpadded base64url (43 chars), per RFC 7636 §4.1.
func GeneratePKCEVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating pkce verifier: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// PKCEChallengeS256 derives the S256 code challenge from a verifier, per RFC 7636 §4.2.
func PKCEChallengeS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
