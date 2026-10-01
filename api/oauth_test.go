package api

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestValidateOAuthEndpoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		hubURL   string
		endpoint string
		wantErr  bool
	}{
		{name: "same origin", hubURL: "https://hub.example.com/hub", endpoint: "https://hub.example.com/api/rest/oauth2/token"},
		{name: "same origin case insensitive", hubURL: "https://HUB.example.com", endpoint: "https://hub.EXAMPLE.com/oauth"},
		{name: "different host", hubURL: "https://hub.example.com", endpoint: "https://evil.example.com/oauth", wantErr: true},
		{name: "different scheme", hubURL: "https://hub.example.com", endpoint: "http://hub.example.com/oauth", wantErr: true},
		{name: "relative endpoint", hubURL: "https://hub.example.com", endpoint: "/oauth", wantErr: true},
		{name: "invalid hub", hubURL: "://bad", endpoint: "https://hub.example.com/oauth", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOAuthEndpoint(tt.hubURL, tt.endpoint, "token_endpoint")
			if tt.wantErr && err == nil {
				t.Fatal("expected an error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestBuildAuthorizationURL(t *testing.T) {
	t.Parallel()

	req := AuthorizationRequest{
		ClientID:      "cli client&x=1",
		RedirectURI:   "http://127.0.0.1:5000",
		Scopes:        []string{"yt-svc", "extra svc"},
		State:         "st&ate",
		CodeChallenge: "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
	}
	got, err := BuildAuthorizationURL("https://hub.example.com/hub/api/rest/oauth2/auth?keep=1", &req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("result is not a URL: %v", err)
	}
	if u.Scheme != "https" || u.Host != "hub.example.com" || u.Path != "/hub/api/rest/oauth2/auth" {
		t.Fatalf("endpoint changed: %s", got)
	}
	q := u.Query()
	want := map[string]string{
		"keep":                  "1",
		"response_type":         "code",
		"client_id":             "cli client&x=1",
		"redirect_uri":          "http://127.0.0.1:5000",
		"scope":                 "yt-svc extra svc",
		"state":                 "st&ate",
		"code_challenge":        "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		"code_challenge_method": "S256",
		"access_type":           "offline",
		"request_credentials":   "default",
	}
	for k, v := range want {
		if q.Get(k) != v {
			t.Errorf("%s = %q, want %q", k, q.Get(k), v)
		}
	}
	// Injected "&x=1" inside the client ID must stay encoded, never become a parameter.
	if _, injected := q["x"]; injected {
		t.Errorf("client_id value injected a query parameter: %s", got)
	}
	if len(q) != len(want) {
		t.Errorf("unexpected parameters in %s", got)
	}
}

func TestBuildAuthorizationURL_Rejects(t *testing.T) {
	t.Parallel()

	valid := AuthorizationRequest{ClientID: "c", RedirectURI: "http://127.0.0.1:1", Scopes: []string{"s"}, State: "s", CodeChallenge: "c"}
	for name, tc := range map[string]struct {
		endpoint string
		req      AuthorizationRequest
	}{
		"relative endpoint":  {"/oauth2/auth", valid},
		"userinfo endpoint":  {"https://user@hub.example.com/auth", valid},
		"javascript scheme":  {"javascript://hub.example.com/auth", valid},
		"fragment endpoint":  {"https://hub.example.com/auth#frag", valid},
		"missing challenge":  {"https://hub.example.com/auth", AuthorizationRequest{ClientID: "c", RedirectURI: "r", Scopes: []string{"s"}, State: "s"}},
		"missing state":      {"https://hub.example.com/auth", AuthorizationRequest{ClientID: "c", RedirectURI: "r", Scopes: []string{"s"}, CodeChallenge: "c"}},
		"missing scope":      {"https://hub.example.com/auth", AuthorizationRequest{ClientID: "c", RedirectURI: "r", State: "s", CodeChallenge: "c"}},
		"missing client id":  {"https://hub.example.com/auth", AuthorizationRequest{RedirectURI: "r", Scopes: []string{"s"}, State: "s", CodeChallenge: "c"}},
		"missing redirect":   {"https://hub.example.com/auth", AuthorizationRequest{ClientID: "c", Scopes: []string{"s"}, State: "s", CodeChallenge: "c"}},
		"unparsable address": {"://bad", valid},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := BuildAuthorizationURL(tc.endpoint, &tc.req); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestParseTokenResponse(t *testing.T) {
	t.Parallel()

	tok, err := parseTokenResponse(200, []byte(`{"access_token":"at","token_type":"Bearer","expires_in":3600,"refresh_token":"rt","scope":"svc"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.AccessToken != "at" || tok.RefreshToken != "rt" || tok.ExpiresIn != 3600 || tok.Scope != "svc" || tok.TokenType != "Bearer" {
		t.Fatalf("unexpected token: %+v", tok)
	}
	// Hub's documented sample omits token_type; that is accepted.
	if _, err := parseTokenResponse(200, []byte(`{"access_token":"at","expires_in":3600}`)); err != nil {
		t.Fatalf("token without token_type: %v", err)
	}

	for name, tc := range map[string]struct {
		status int
		body   string
		want   string
	}{
		"no access token":   {200, `{"expires_in":3600}`, "no access_token"},
		"wrong token type":  {200, `{"access_token":"at","token_type":"mac","expires_in":3600}`, "unsupported token_type"},
		"no expiry":         {200, `{"access_token":"at"}`, "invalid expires_in"},
		"not json":          {200, `<html>`, "not valid JSON"},
		"invalid grant":     {400, `{"error":"invalid_grant","error_description":"Unknown refresh token"}`, "Unknown refresh token (HTTP 400)"},
		"confidential only": {401, `{"error":"invalid_client"}`, "Require PKCE with no Client secret"},
		"plain text error":  {502, `bad gateway`, "HTTP 502: bad gateway"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := parseTokenResponse(tc.status, []byte(tc.body))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want it to contain %q", err, tc.want)
			}
		})
	}

	// A malformed success body must never be echoed: it may hold a token.
	_, err = parseTokenResponse(200, []byte(`{"access_token":"secret-at","expires_in":"soon"}`))
	if err == nil || strings.Contains(err.Error(), "secret-at") {
		t.Fatalf("error leaked the body or was nil: %v", err)
	}
}

func TestIsInvalidGrant(t *testing.T) {
	t.Parallel()

	_, err := parseTokenResponse(400, []byte(`{"error":"invalid_grant"}`))
	if !IsInvalidGrant(fmt.Errorf("wrapped: %w", err)) {
		t.Fatalf("invalid_grant not detected through wrapping: %v", err)
	}
	_, err = parseTokenResponse(400, []byte(`{"error":"invalid_request"}`))
	if IsInvalidGrant(err) {
		t.Fatal("invalid_request misclassified as invalid_grant")
	}
	if IsInvalidGrant(errors.New("network down")) {
		t.Fatal("plain error misclassified as invalid_grant")
	}
}

func TestTokenExpiryAndRefresh(t *testing.T) {
	t.Parallel()

	received := time.Unix(1_000_000, 0)
	exp := TokenExpiry(received, 3600)
	if exp != 1_003_600 {
		t.Fatalf("TokenExpiry = %d, want 1003600", exp)
	}
	skew := 60 * time.Second
	for _, tc := range []struct {
		name string
		now  time.Time
		want bool
	}{
		{"fresh", received, false},
		{"just outside skew", time.Unix(exp-61, 0), false},
		{"inside skew", time.Unix(exp-60, 0), true},
		{"expired", time.Unix(exp+1, 0), true},
	} {
		if got := TokenNeedsRefresh(exp, tc.now, skew); got != tc.want {
			t.Errorf("%s: TokenNeedsRefresh = %v, want %v", tc.name, got, tc.want)
		}
	}
	if TokenNeedsRefresh(0, time.Now(), skew) {
		t.Error("a token without expiry (permanent) must never need refresh")
	}
}

func TestValidateOAuthURL(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		url      string
		insecure bool
		ok       bool
	}{
		{"https://example.youtrack.cloud", false, true},
		{"https://yt.example.com/youtrack", false, true},
		{"http://127.0.0.1:8080", true, true},
		{"http://localhost:8080", true, true},
		{"http://[::1]:8080", true, true},
		{"http://127.0.0.1:8080", false, false},
		{"http://yt.example.com", true, false},
		{"http://yt.example.com", false, false},
		{"example.youtrack.cloud", false, false},
		{"/relative", false, false},
		{"https://user:pw@example.youtrack.cloud", false, false},
		{"javascript://example.com/%0aalert(1)", false, false},
		{"file:///etc/passwd", false, false},
	} {
		err := ValidateOAuthURL(tc.url, tc.insecure)
		if (err == nil) != tc.ok {
			t.Errorf("ValidateOAuthURL(%q, %v) = %v, want ok=%v", tc.url, tc.insecure, err, tc.ok)
		}
	}
}

func TestResolveHubURL(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ base, ref, want string }{
		{"http://127.0.0.1:8080", "/hub", "http://127.0.0.1:8080/hub"},
		{"https://yt.example.com/youtrack", "/hub", "https://yt.example.com/youtrack/hub"},
		{"https://yt.example.com/youtrack/", "/youtrack/hub", "https://yt.example.com/youtrack/hub"},
		{"https://x.youtrack.cloud", "https://x.youtrack.cloud/hub", "https://x.youtrack.cloud/hub"},
		{"https://yt.example.com", "https://hub.example.com", "https://hub.example.com"},
		{"https://yt.example.com", "", ""},
	} {
		if got := ResolveHubURL(tc.base, tc.ref); got != tc.want {
			t.Errorf("ResolveHubURL(%q, %q) = %q, want %q", tc.base, tc.ref, got, tc.want)
		}
	}
}

func TestDefaultOAuthClientID(t *testing.T) {
	t.Parallel()

	for hubURL, want := range map[string]string{
		"https://hub.jetbrains.com":              "localhost-auth",
		"https://HUB.JetBrains.com/":             "localhost-auth",
		"https://example.youtrack.cloud/hub":     "youtrack-cli",
		"https://hub.jetbrains.com.evil.example": "youtrack-cli",
		"https://whitephoenix.eu.ngrok.io/hub":   "youtrack-cli",
	} {
		if got := DefaultOAuthClientID(hubURL); got != want {
			t.Errorf("DefaultOAuthClientID(%q) = %q, want %q", hubURL, got, want)
		}
	}
}
