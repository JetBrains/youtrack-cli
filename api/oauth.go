package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// noRedirectClient never follows redirects — used to inspect Hub's Location headers.
var noRedirectClient = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// RingConfig is the ring(...) fragment of YouTrack's /api/config response.
// It identifies the Hub authorization server and the YouTrack service ID
// registered there. Reachable unauthenticated on all YouTrack versions.
type RingConfig struct {
	URL            string `json:"url"`
	ServiceID      string `json:"serviceId"`
	HasEmbeddedHub bool   `json:"hasEmbeddedHub"`
}

// GetRingConfig fetches Hub connection details from YouTrack's frontend config.
func GetRingConfig(baseURL string) (*RingConfig, error) {
	endpoint := strings.TrimRight(baseURL, "/") + "/api/config?fields=ring(url,serviceId,hasEmbeddedHub)"
	resp, err := hubHTTPClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("fetching ring config: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
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

// OAuthEndpoints holds the Hub OAuth 2.0 endpoints discovered via
// /.well-known/openid-configuration (with a conventional-path fallback for
// older Hub versions that do not serve the discovery document).
type OAuthEndpoints struct {
	AuthorizationEndpoint string   `json:"authorization_endpoint"`
	TokenEndpoint         string   `json:"token_endpoint"`
	CodeChallengeMethods  []string `json:"code_challenge_methods_supported"`
}

// DiscoverOAuthEndpoints resolves the authorization and token endpoints for a Hub.
// When the discovery document advertises code_challenge_methods_supported it must
// include S256 — the CLI only implements PKCE with S256.
func DiscoverOAuthEndpoints(hubURL string) (*OAuthEndpoints, error) {
	hubURL = strings.TrimRight(hubURL, "/")
	fallback := &OAuthEndpoints{
		AuthorizationEndpoint: hubURL + "/api/rest/oauth2/auth",
		TokenEndpoint:         hubURL + "/api/rest/oauth2/token",
	}
	resp, err := hubHTTPClient.Get(hubURL + "/.well-known/openid-configuration")
	if err != nil {
		return fallback, nil
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil || resp.StatusCode >= 300 {
		return fallback, nil
	}
	var eps OAuthEndpoints
	if err := json.Unmarshal(data, &eps); err != nil ||
		eps.AuthorizationEndpoint == "" || eps.TokenEndpoint == "" {
		return fallback, nil
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

// PreflightOAuthClient checks — without opening a browser — whether Hub accepts
// the client_id and redirect_uri baked into authorizeURL. Hub answers the
// authorization request with a 303: to its login page when the client is valid,
// or to /auth/oauth/error when the client is unknown or the redirect is rejected.
func PreflightOAuthClient(authorizeURL string) error {
	resp, err := noRedirectClient.Get(authorizeURL)
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

// ExchangeOAuthCode redeems an authorization code for an access token.
// The client is public: no Authorization header and no secret — client_id goes
// in the form body, proof of possession comes from the PKCE code_verifier.
func ExchangeOAuthCode(tokenEndpoint, clientID, code, redirectURI, codeVerifier string) (*OAuthToken, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("client_id", clientID)
	form.Set("code_verifier", codeVerifier)

	resp, err := hubHTTPClient.PostForm(tokenEndpoint, form)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
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

// GeneratePKCEVerifier returns a 32-byte random code verifier encoded as
// unpadded base64url (43 characters), per RFC 7636 §4.1.
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
