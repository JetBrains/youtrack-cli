package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
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

// noRedirectClient never follows redirects — used to inspect Hub's Location headers.
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

// OAuthToken is the Hub token endpoint response for an authorization-code grant.
type OAuthToken struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	Scope       string `json:"scope"`
}

// ExchangeOAuthCodeContext redeems an authorization code as a public client — client_id in the form body, no secret, PKCE code_verifier for proof of possession.
func ExchangeOAuthCodeContext(ctx context.Context, tokenEndpoint, clientID, code, redirectURI, codeVerifier string) (*OAuthToken, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("client_id", clientID)
	form.Set("code_verifier", codeVerifier)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := hubHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxOAuthResponseBodySize))
	if err != nil {
		return nil, fmt.Errorf("reading token response: %w", err)
	}
	if resp.StatusCode >= 300 {
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
	var tok OAuthToken
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, fmt.Errorf("parsing token response: %w", err)
	}
	if tok.AccessToken == "" {
		return nil, fmt.Errorf("token response contained no access_token")
	}
	return &tok, nil
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
