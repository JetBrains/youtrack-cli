package api_test

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	fakeClientID  = "youtrack-cli-test"
	fakeServiceID = "yt-service-id"
)

// fakeHub is an in-process YouTrack + Hub: discovery, the OAuth authorize/token/revoke endpoints, and /api/users/me.
// It enforces PKCE (S256), the public client ID, and bearer tokens, and records what the CLI sent.
type fakeHub struct {
	srv *httptest.Server

	mu            sync.Mutex
	challenges    map[string]string // code → code_challenge (a code is issued when a test "approves" in the browser)
	exchanged     []string          // codes redeemed at the token endpoint
	validAccess   map[string]bool
	validRefresh  map[string]bool
	authHeaders   []string // Authorization headers seen by the YouTrack API
	revoked       []string // token_type_hint of each revocation request
	tokenRequests atomic.Int32
	refreshCalls  atomic.Int32
	serial        atomic.Int32

	// Behavior switches.
	rotateRefresh   bool          // issue a new refresh token on refresh and revoke the old one
	exchangeError   string        // when set, the code exchange fails with this OAuth error code
	tokenDelay      time.Duration // slows the token endpoint to widen race windows
	advertiseRevoke bool
	revokeStatus    int
}

func newFakeHub(t *testing.T) *fakeHub {
	t.Helper()
	h := &fakeHub{
		challenges:   map[string]string{},
		validAccess:  map[string]bool{},
		validRefresh: map[string]bool{},
		revokeStatus: http.StatusOK,
	}
	h.srv = httptest.NewServer(http.HandlerFunc(h.serve))
	t.Cleanup(h.srv.Close)
	return h
}

func (h *fakeHub) hubURL() string { return h.srv.URL + "/hub" }

func (h *fakeHub) next(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, h.serial.Add(1))
}

func (h *fakeHub) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *fakeHub) serve(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/config":
		h.writeJSON(w, 200, map[string]any{"ring": map[string]any{"url": "/hub", "serviceId": fakeServiceID}})
	case "/hub/.well-known/openid-configuration":
		meta := map[string]any{
			"issuer":                           h.hubURL(),
			"authorization_endpoint":           h.hubURL() + "/api/rest/oauth2/auth",
			"token_endpoint":                   h.hubURL() + "/api/rest/oauth2/token",
			"code_challenge_methods_supported": []string{"S256"},
		}
		if h.advertiseRevoke {
			meta["revocation_endpoint"] = h.hubURL() + "/api/rest/oauth2/revoke"
		}
		h.writeJSON(w, 200, meta)
	case "/hub/api/rest/oauth2/auth":
		// Pre-flight: an unknown client is sent to Hub's error page, like the real Hub.
		if r.URL.Query().Get("client_id") != fakeClientID {
			http.Redirect(w, r, "/hub/auth/oauth/error", http.StatusSeeOther)
			return
		}
		w.WriteHeader(http.StatusOK)
	case "/hub/api/rest/oauth2/token":
		h.token(w, r)
	case "/hub/api/rest/oauth2/revoke":
		_ = r.ParseForm()
		h.mu.Lock()
		h.revoked = append(h.revoked, r.PostForm.Get("token_type_hint"))
		h.mu.Unlock()
		w.WriteHeader(h.revokeStatus)
	case "/api/users/me":
		auth := r.Header.Get("Authorization")
		h.mu.Lock()
		h.authHeaders = append(h.authHeaders, auth)
		ok := h.validAccess[strings.TrimPrefix(auth, "Bearer ")] || strings.HasPrefix(auth, "Bearer perm:")
		h.mu.Unlock()
		if !ok {
			h.writeJSON(w, 401, map[string]any{"error": "invalid_token"})
			return
		}
		h.writeJSON(w, 200, map[string]any{"id": "1-1", "login": "alice", "fullName": "Alice Example"})
	default:
		http.NotFound(w, r)
	}
}

func (h *fakeHub) token(w http.ResponseWriter, r *http.Request) {
	h.tokenRequests.Add(1)
	time.Sleep(h.tokenDelay)
	if err := r.ParseForm(); err != nil || r.Header.Get("Authorization") != "" {
		// A public client must never send client credentials.
		h.writeJSON(w, 400, map[string]any{"error": "invalid_request"})
		return
	}
	f := r.PostForm
	if f.Get("client_id") != fakeClientID {
		h.writeJSON(w, 401, map[string]any{"error": "invalid_client"})
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	switch f.Get("grant_type") {
	case "authorization_code":
		code := f.Get("code")
		h.exchanged = append(h.exchanged, code)
		if h.exchangeError != "" {
			h.writeJSON(w, 400, map[string]any{"error": h.exchangeError, "error_description": "exchange refused by test"})
			return
		}
		challenge, ok := h.challenges[code]
		delete(h.challenges, code)
		sum := sha256.Sum256([]byte(f.Get("code_verifier")))
		if !ok || base64.RawURLEncoding.EncodeToString(sum[:]) != challenge {
			h.writeJSON(w, 400, map[string]any{"error": "invalid_grant", "error_description": "PKCE verification failed"})
			return
		}
	case "refresh_token":
		h.refreshCalls.Add(1)
		rt := f.Get("refresh_token")
		if !h.validRefresh[rt] || f.Get("scope") == "" {
			h.writeJSON(w, 400, map[string]any{"error": "invalid_grant", "error_description": "Unknown refresh token"})
			return
		}
		access := h.next("access")
		h.validAccess[access] = true
		resp := map[string]any{"access_token": access, "token_type": "Bearer", "expires_in": 3600, "scope": f.Get("scope")}
		if h.rotateRefresh {
			delete(h.validRefresh, rt)
			nrt := h.next("refresh")
			h.validRefresh[nrt] = true
			resp["refresh_token"] = nrt
		}
		h.writeJSON(w, 200, resp)
		return
	default:
		h.writeJSON(w, 400, map[string]any{"error": "unsupported_grant_type"})
		return
	}
	access, refresh := h.next("access"), h.next("refresh")
	h.validAccess[access] = true
	h.validRefresh[refresh] = true
	h.writeJSON(w, 200, map[string]any{
		"access_token": access, "refresh_token": refresh, "token_type": "Bearer", "expires_in": 3600, "scope": fakeServiceID,
	})
}

// approve plays the user approving in the browser: Hub issues a code bound to the request's PKCE challenge and
// redirects to redirect_uri. It returns that final redirect URL.
func (h *fakeHub) approve(t *testing.T, authorizeURL string) string {
	t.Helper()
	u, err := url.Parse(authorizeURL)
	require.NoError(t, err)
	q := u.Query()
	require.Equal(t, "S256", q.Get("code_challenge_method"), "PKCE must use S256")
	require.NotEmpty(t, q.Get("code_challenge"))
	require.Equal(t, "offline", q.Get("access_type"), "a refresh token must be requested")
	require.Equal(t, fakeClientID, q.Get("client_id"))
	require.Contains(t, q.Get("redirect_uri"), "http://127.0.0.1:")
	code := h.next("code")
	h.mu.Lock()
	h.challenges[code] = q.Get("code_challenge")
	h.mu.Unlock()
	return q.Get("redirect_uri") + "?" + url.Values{"code": {code}, "state": {q.Get("state")}}.Encode()
}

// seedSession writes a config holding an OAuth session for this fake (already issued by the fake), expiring at expiresAt.
func (h *fakeHub) seedSession(t *testing.T, home string, expiresAt int64) (access, refresh string) {
	t.Helper()
	access, refresh = h.next("access"), h.next("refresh")
	h.mu.Lock()
	h.validAccess[access] = true
	h.validRefresh[refresh] = true
	h.mu.Unlock()
	cfg := map[string]any{
		"currentInstance": "fake",
		"instances": map[string]any{
			"fake": map[string]any{
				"url": h.srv.URL, "token": access, "refreshToken": refresh, "authMethod": "oauth", "tokenExpiresAt": expiresAt,
				"tokenType": "Bearer", "scope": fakeServiceID, "hubUrl": h.hubURL(), "clientId": fakeClientID,
				"tokenEndpoint": h.hubURL() + "/api/rest/oauth2/token", "revocationEndpoint": h.hubURL() + "/api/rest/oauth2/revoke",
			},
		},
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(home, ".youtrack-cli-config.json"), data, 0600))
	return access, refresh
}

func readConfig(t *testing.T, home string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(home, ".youtrack-cli-config.json"))
	require.NoError(t, err)
	var cfg map[string]any
	require.NoError(t, json.Unmarshal(data, &cfg))
	return cfg
}

func savedInstance(t *testing.T, home, alias string) map[string]any {
	t.Helper()
	inst, _ := readConfig(t, home)["instances"].(map[string]any)[alias].(map[string]any)
	return inst
}

// isolatedEnv is a minimal environment: no inherited YOUTRACK_* credentials, file-only token storage.
func isolatedEnv(home string, extra ...string) []string {
	return append([]string{"PATH=" + os.Getenv("PATH"), "HOME=" + home, "YOUTRACK_DISABLE_KEYRING=1", "YOUTRACK_NO_UPDATE=1"}, extra...)
}

// loginProc is a running `instance login`; the test drives it through stderr (the printed URL) and stdin (a pasted URL).
type loginProc struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout bytes.Buffer
	mu     sync.Mutex
	stderr bytes.Buffer
	urlCh  chan string
	done   chan error
}

var authorizeURLPattern = regexp.MustCompile(`https?://\S+/oauth2/auth\?\S+`)

func startLogin(t *testing.T, home string, args ...string) *loginProc {
	t.Helper()
	base := []string{"--json", "instance", "login", "--insecure", "--no-browser", "--client-id", fakeClientID, "--timeout", "20s"}
	p := &loginProc{urlCh: make(chan string, 1), done: make(chan error, 1)}
	p.cmd = exec.Command(binaryPath, append(base, args...)...)
	p.cmd.Env = isolatedEnv(home)
	p.cmd.Stdout = &p.stdout
	var err error
	p.stdin, err = p.cmd.StdinPipe()
	require.NoError(t, err)
	stderr, err := p.cmd.StderrPipe()
	require.NoError(t, err)
	require.NoError(t, p.cmd.Start())
	t.Cleanup(func() { _ = p.cmd.Process.Kill() })

	go func() {
		sc := bufio.NewScanner(stderr)
		sc.Buffer(make([]byte, 64*1024), 1<<20)
		sent := false
		for sc.Scan() {
			line := sc.Text()
			p.mu.Lock()
			p.stderr.WriteString(line + "\n")
			p.mu.Unlock()
			if m := authorizeURLPattern.FindString(line); m != "" && !sent {
				p.urlCh <- m
				sent = true
			}
		}
		p.done <- p.cmd.Wait()
	}()
	return p
}

func (p *loginProc) authorizeURL(t *testing.T) string {
	t.Helper()
	select {
	case u := <-p.urlCh:
		return u
	case err := <-p.done:
		t.Fatalf("login exited before printing the authorization URL (%v); stderr:\n%s", err, p.errText())
	case <-time.After(15 * time.Second):
		t.Fatalf("no authorization URL printed; stderr:\n%s", p.errText())
	}
	return ""
}

func (p *loginProc) wait(t *testing.T) error {
	t.Helper()
	select {
	case err := <-p.done:
		return err
	case <-time.After(30 * time.Second):
		t.Fatalf("login did not finish; stderr:\n%s", p.errText())
	}
	return nil
}

func (p *loginProc) errText() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stderr.String()
}

func TestOAuthLogin_LoopbackFlow(t *testing.T) {
	hub := newFakeHub(t)
	hub.advertiseRevoke = true
	home := t.TempDir()

	p := startLogin(t, home, hub.srv.URL, "--alias", "fake")
	redirect := hub.approve(t, p.authorizeURL(t))

	// The browser follows the redirect to the loopback listener.
	resp, err := http.Get(redirect)
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	page := string(body)
	assert.Contains(t, page, "Authentication successful")
	code := regexp.MustCompile(`code=([^&]+)`).FindStringSubmatch(redirect)[1]
	assert.NotContains(t, page, code, "success page must not echo the authorization code")
	assert.NotRegexp(t, `(src|href)=["']?https?:`, page, "success page must not load remote resources")
	assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "default-src 'none'")
	assert.Equal(t, "no-referrer", resp.Header.Get("Referrer-Policy"))

	require.NoError(t, p.wait(t), "stderr:\n%s", p.errText())

	// Exactly one callback is accepted: the listener is closed afterwards.
	if resp, err := http.Get(redirect); err == nil {
		resp.Body.Close()
		t.Fatalf("loopback listener still accepting requests after the flow (HTTP %d)", resp.StatusCode)
	}

	var out map[string]any
	require.NoError(t, json.Unmarshal(p.stdout.Bytes(), &out), "stdout: %s", p.stdout.String())
	assert.Equal(t, "alice", out["login"])
	assert.Equal(t, true, out["hasRefreshToken"])
	assert.Equal(t, hub.hubURL(), out["hubUrl"])
	assert.Equal(t, fakeClientID, out["clientId"])
	assert.NotContains(t, out, "token")

	inst := savedInstance(t, home, "fake")
	require.NotNil(t, inst)
	assert.Equal(t, "oauth", inst["authMethod"])
	assert.Equal(t, hub.srv.URL, inst["url"])
	assert.Equal(t, hub.hubURL(), inst["hubUrl"])
	assert.Equal(t, fakeClientID, inst["clientId"])
	assert.Equal(t, "Bearer", inst["tokenType"])
	assert.Equal(t, fakeServiceID, inst["scope"])
	assert.Equal(t, hub.hubURL()+"/api/rest/oauth2/token", inst["tokenEndpoint"])
	assert.Equal(t, hub.hubURL()+"/api/rest/oauth2/revoke", inst["revocationEndpoint"])
	assert.NotEmpty(t, inst["refreshToken"])
	assert.NotEmpty(t, inst["token"])
	expiresAt, _ := inst["tokenExpiresAt"].(float64)
	assert.InDelta(t, time.Now().Unix()+3600, expiresAt, 30, "expiry is absolute: receipt time + expires_in")

	// Tokens appear in neither stdout nor stderr.
	for _, secret := range []string{inst["token"].(string), inst["refreshToken"].(string)} {
		assert.NotContains(t, p.stdout.String(), secret)
		assert.NotContains(t, p.errText(), secret)
	}
	if runtime.GOOS != "windows" {
		fi, err := os.Stat(filepath.Join(home, ".youtrack-cli-config.json"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), fi.Mode().Perm())
	}
}

func TestOAuthLogin_StateMismatchRejected(t *testing.T) {
	hub := newFakeHub(t)
	home := t.TempDir()

	p := startLogin(t, home, hub.srv.URL, "--alias", "fake")
	redirect := hub.approve(t, p.authorizeURL(t))

	forged, err := url.Parse(redirect)
	require.NoError(t, err)
	q := forged.Query()
	forgedCode := "forged-code"
	q.Set("code", forgedCode)
	q.Set("state", "attacker-state")
	forged.RawQuery = q.Encode()

	resp, err := http.Get(forged.String())
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "a callback with the wrong state must be rejected")

	// The genuine callback still completes; the forged code was never exchanged.
	resp, err = http.Get(redirect)
	require.NoError(t, err)
	resp.Body.Close()
	require.NoError(t, p.wait(t), "stderr:\n%s", p.errText())

	hub.mu.Lock()
	defer hub.mu.Unlock()
	assert.NotContains(t, hub.exchanged, forgedCode)
	assert.Len(t, hub.exchanged, 1)
}

func TestOAuthLogin_AccessDenied(t *testing.T) {
	hub := newFakeHub(t)
	home := t.TempDir()

	p := startLogin(t, home, hub.srv.URL, "--alias", "fake")
	u, err := url.Parse(p.authorizeURL(t))
	require.NoError(t, err)
	denied := u.Query().Get("redirect_uri") + "?" + url.Values{"error": {"access_denied"}, "state": {u.Query().Get("state")}}.Encode()

	resp, err := http.Get(denied)
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, string(body), "Authentication failed")

	require.Error(t, p.wait(t))
	assert.Contains(t, p.errText(), "access_denied")
	assert.Zero(t, hub.tokenRequests.Load(), "no token request after a denied authorization")
	_, err = os.Stat(filepath.Join(home, ".youtrack-cli-config.json"))
	assert.True(t, os.IsNotExist(err), "nothing is saved after a denied authorization")
}

func TestOAuthLogin_TokenExchangeFailure(t *testing.T) {
	for _, tc := range []struct {
		code, want string
	}{
		{"invalid_grant", "exchange refused by test"},
		{"invalid_client", "public client"},
	} {
		t.Run(tc.code, func(t *testing.T) {
			hub := newFakeHub(t)
			hub.exchangeError = tc.code
			home := t.TempDir()

			p := startLogin(t, home, hub.srv.URL, "--alias", "fake")
			resp, err := http.Get(hub.approve(t, p.authorizeURL(t)))
			require.NoError(t, err)
			resp.Body.Close()

			require.Error(t, p.wait(t))
			assert.Contains(t, p.errText(), "exchanging authorization code")
			assert.Contains(t, p.errText(), tc.want)
			_, err = os.Stat(filepath.Join(home, ".youtrack-cli-config.json"))
			assert.True(t, os.IsNotExist(err), "nothing is saved when the exchange fails")
		})
	}
}

func TestOAuthLogin_NoBrowserPastedRedirectURL(t *testing.T) {
	hub := newFakeHub(t)
	home := t.TempDir()

	p := startLogin(t, home, hub.srv.URL, "--alias", "fake")
	redirect := hub.approve(t, p.authorizeURL(t))

	// The user copies the URL from the browser on another device (the loopback request never arrives).
	_, err := io.WriteString(p.stdin, "not a url\n"+redirect+"\n")
	require.NoError(t, err)
	require.NoError(t, p.wait(t), "stderr:\n%s", p.errText())
	assert.Contains(t, p.errText(), "Paste that full URL here")
	assert.Contains(t, p.errText(), "paste the full URL", "an invalid paste is explained and the prompt keeps waiting")
	assert.NotEmpty(t, savedInstance(t, home, "fake")["refreshToken"])
}

func TestOAuthLogin_PastedURLWithWrongStateRejected(t *testing.T) {
	hub := newFakeHub(t)
	home := t.TempDir()

	p := startLogin(t, home, hub.srv.URL, "--alias", "fake")
	redirect := hub.approve(t, p.authorizeURL(t))
	wrong := regexp.MustCompile(`state=[^&]+`).ReplaceAllString(redirect, "state=from-another-attempt")

	_, err := io.WriteString(p.stdin, wrong+"\n")
	require.NoError(t, err)
	require.Error(t, p.wait(t))
	assert.Contains(t, p.errText(), "state mismatch")
	assert.Zero(t, hub.tokenRequests.Load(), "a code with the wrong state is never exchanged")
}

func TestOAuthLogin_RequiresHTTPS(t *testing.T) {
	// Plain http to a remote host is refused even with --insecure; the guard runs before any request.
	cmd := exec.Command(binaryPath, "instance", "login", "http://youtrack.invalid", "--insecure", "--no-browser", "--save=false")
	cmd.Env = isolatedEnv(t.TempDir())
	out, err := cmd.CombinedOutput()
	require.Error(t, err)
	assert.Contains(t, string(out), "refusing to sign in over plain http")

	// Loopback http needs an explicit --insecure.
	hub := newFakeHub(t)
	cmd = exec.Command(binaryPath, "instance", "login", hub.srv.URL, "--no-browser", "--save=false")
	cmd.Env = isolatedEnv(t.TempDir())
	out, err = cmd.CombinedOutput()
	require.Error(t, err)
	assert.Contains(t, string(out), "--insecure")
	assert.Zero(t, hub.tokenRequests.Load())
}

func TestOAuthRefresh(t *testing.T) {
	for _, rotate := range []bool{false, true} {
		t.Run(fmt.Sprintf("rotate=%v", rotate), func(t *testing.T) {
			hub := newFakeHub(t)
			hub.rotateRefresh = rotate
			home := t.TempDir()
			// Inside the 60s refresh window, so the token is refreshed before the request.
			oldAccess, oldRefresh := hub.seedSession(t, home, time.Now().Unix()+30)

			cmd := exec.Command(binaryPath, "--json", "user", "current")
			cmd.Env = isolatedEnv(home)
			out, err := cmd.CombinedOutput()
			require.NoError(t, err, "output: %s", out)
			assert.Contains(t, string(out), `"login": "alice"`)
			assert.EqualValues(t, 1, hub.refreshCalls.Load())

			inst := savedInstance(t, home, "fake")
			assert.NotEqual(t, oldAccess, inst["token"], "the refreshed access token is persisted")
			hub.mu.Lock()
			assert.Equal(t, []string{"Bearer " + inst["token"].(string)}, hub.authHeaders, "the request used the refreshed token")
			hub.mu.Unlock()
			if rotate {
				assert.NotEqual(t, oldRefresh, inst["refreshToken"], "a rotated refresh token replaces the old one")
			} else {
				assert.Equal(t, oldRefresh, inst["refreshToken"], "without rotation the refresh token is kept")
			}
			expiresAt, _ := inst["tokenExpiresAt"].(float64)
			assert.InDelta(t, time.Now().Unix()+3600, expiresAt, 30)
			matches, _ := filepath.Glob(filepath.Join(home, ".youtrack-cli-config.json.*.tmp"))
			assert.Empty(t, matches, "atomic write leaves no temp files")

			// A fresh token is not refreshed again.
			cmd = exec.Command(binaryPath, "--json", "user", "current")
			cmd.Env = isolatedEnv(home)
			out, err = cmd.CombinedOutput()
			require.NoError(t, err, "output: %s", out)
			assert.EqualValues(t, 1, hub.refreshCalls.Load())
		})
	}
}

func TestOAuthRefresh_InvalidGrantClearsCredentials(t *testing.T) {
	hub := newFakeHub(t)
	home := t.TempDir()
	access, refresh := hub.seedSession(t, home, time.Now().Unix()-10)
	hub.mu.Lock()
	delete(hub.validRefresh, refresh) // revoked at Hub
	hub.mu.Unlock()

	cmd := exec.Command(binaryPath, "--format", "table", "user", "current")
	cmd.Env = isolatedEnv(home)
	out, err := cmd.CombinedOutput()
	require.Error(t, err)
	assert.Contains(t, string(out), "expired or was revoked")
	assert.Contains(t, string(out), "youtrack instance login fake")
	assert.NotContains(t, string(out), refresh)

	inst := savedInstance(t, home, "fake")
	assert.Empty(t, inst["token"], "stale access token removed")
	assert.Empty(t, inst["refreshToken"], "stale refresh token removed")
	assert.Equal(t, hub.srv.URL, inst["url"], "the profile is kept for re-login")
	data, _ := os.ReadFile(filepath.Join(home, ".youtrack-cli-config.json"))
	assert.NotContains(t, string(data), access)

	// Later commands keep pointing at re-login and do not retry the dead grant.
	cmd = exec.Command(binaryPath, "--format", "table", "user", "current")
	cmd.Env = isolatedEnv(home)
	out, err = cmd.CombinedOutput()
	require.Error(t, err)
	assert.Contains(t, string(out), "youtrack instance login fake")
	assert.EqualValues(t, 1, hub.refreshCalls.Load())
}

func TestOAuthRefresh_ConcurrentProcessesRefreshOnce(t *testing.T) {
	hub := newFakeHub(t)
	hub.rotateRefresh = true // a second refresh with the old token would fail, so a race would surface as an error
	hub.tokenDelay = 300 * time.Millisecond
	home := t.TempDir()
	hub.seedSession(t, home, time.Now().Unix()+10)

	const n = 6
	var wg sync.WaitGroup
	errs := make([]error, n)
	outs := make([][]byte, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cmd := exec.Command(binaryPath, "--json", "user", "current")
			cmd.Env = isolatedEnv(home)
			outs[i], errs[i] = cmd.CombinedOutput()
		}()
	}
	wg.Wait()
	for i := range n {
		require.NoError(t, errs[i], "process %d: %s", i, outs[i])
	}
	assert.EqualValues(t, 1, hub.refreshCalls.Load(), "concurrent processes must share a single refresh")
}

func TestOAuthRefresh_VerboseOutputHasNoSecrets(t *testing.T) {
	hub := newFakeHub(t)
	hub.rotateRefresh = true
	home := t.TempDir()
	oldAccess, oldRefresh := hub.seedSession(t, home, time.Now().Unix()+5)

	cmd := exec.Command(binaryPath, "--verbose", "--json", "user", "current")
	cmd.Env = isolatedEnv(home)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "output: %s", out)
	text := string(out)
	assert.Contains(t, text, "refreshing OAuth access token", "verbose output shows the refresh")
	assert.Contains(t, text, "> GET "+hub.srv.URL+"/api/users/me")

	inst := savedInstance(t, home, "fake")
	for _, secret := range []string{oldAccess, oldRefresh, inst["token"].(string), inst["refreshToken"].(string)} {
		assert.NotContains(t, text, secret)
	}
	assert.NotContains(t, text, "Authorization")
	assert.NotContains(t, text, "Bearer")
	assert.NotContains(t, text, "refresh_token=")
}

func TestOAuthPrecedence_ExplicitTokensBeatStoredSession(t *testing.T) {
	hub := newFakeHub(t)
	home := t.TempDir()
	hub.seedSession(t, home, time.Now().Unix()-10) // expired: using it would trigger a refresh

	cmd := exec.Command(binaryPath, "--json", "user", "current")
	cmd.Env = isolatedEnv(home, "YOUTRACK_TOKEN=perm:from-env")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "output: %s", out)

	cmd = exec.Command(binaryPath, "--token", "perm:from-flag", "--json", "user", "current")
	cmd.Env = isolatedEnv(home, "YOUTRACK_TOKEN=perm:from-env")
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, "output: %s", out)

	hub.mu.Lock()
	defer hub.mu.Unlock()
	assert.Equal(t, []string{"Bearer perm:from-env", "Bearer perm:from-flag"}, hub.authHeaders)
	assert.Zero(t, hub.refreshCalls.Load(), "an explicit token never touches the stored session")
}

func TestOAuthStatus_ShowsSessionWithoutTokens(t *testing.T) {
	hub := newFakeHub(t)
	home := t.TempDir()
	access, refresh := hub.seedSession(t, home, time.Now().Unix()+3600)

	cmd := exec.Command(binaryPath, "--json", "auth", "status")
	cmd.Env = isolatedEnv(home)
	out, err := cmd.Output()
	require.NoError(t, err)
	var statuses []map[string]any
	require.NoError(t, json.Unmarshal(out, &statuses), "output: %s", out)
	require.Len(t, statuses, 1)
	assert.Equal(t, "authenticated", statuses[0]["status"])
	assert.Equal(t, "oauth", statuses[0]["authMethod"])
	assert.Equal(t, true, statuses[0]["hasRefreshToken"])
	assert.NotEmpty(t, statuses[0]["expiresAt"])
	assert.NotContains(t, string(out), access)
	assert.NotContains(t, string(out), refresh)

	cmd = exec.Command(binaryPath, "--format", "table", "--server", "fake", "auth", "status")
	cmd.Env = isolatedEnv(home)
	out, err = cmd.Output()
	require.NoError(t, err)
	assert.Contains(t, string(out), "OAuth session (profile fake)")
	assert.Contains(t, string(out), "refresh token stored")
	assert.NotContains(t, string(out), access)
}

func TestOAuthLogout_RevokesBestEffort(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusInternalServerError} {
		t.Run(fmt.Sprintf("revoke HTTP %d", status), func(t *testing.T) {
			hub := newFakeHub(t)
			hub.revokeStatus = status
			home := t.TempDir()
			hub.seedSession(t, home, time.Now().Unix()+3600)

			cmd := exec.Command(binaryPath, "--json", "auth", "logout", "--force")
			cmd.Env = isolatedEnv(home)
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			require.NoError(t, cmd.Run(), "logout must succeed even when revocation fails; stderr: %s", stderr.String())

			var res map[string]any
			require.NoError(t, json.Unmarshal(stdout.Bytes(), &res))
			assert.Equal(t, "logged out", res["status"])
			hub.mu.Lock()
			assert.Equal(t, []string{"refresh_token", "access_token"}, hub.revoked)
			hub.mu.Unlock()
			if status != http.StatusOK {
				assert.Contains(t, stderr.String(), "could not revoke")
			}
			assert.Empty(t, readConfig(t, home)["instances"], "credentials removed")
		})
	}
}

func TestOAuthInstanceList_RefreshesExpiringSession(t *testing.T) {
	hub := newFakeHub(t)
	home := t.TempDir()
	hub.seedSession(t, home, time.Now().Unix()-10)

	cmd := exec.Command(binaryPath, "--json", "instance", "list")
	cmd.Env = isolatedEnv(home)
	out, err := cmd.Output()
	require.NoError(t, err)
	var items []map[string]any
	require.NoError(t, json.Unmarshal(out, &items), "output: %s", out)
	require.Len(t, items, 1)
	assert.Equal(t, "authenticated", items[0]["status"], "an expired OAuth session is refreshed, not reported unavailable")
	assert.EqualValues(t, 1, hub.refreshCalls.Load())
}
