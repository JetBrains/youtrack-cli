package api_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

// The OAuth tests run the CLI against the harness's real Hub. Each test registers its own public PKCE client
// (as an administrator registers "youtrack-cli") and plays the user's browser approval as an authorize
// request authenticated with the harness user's credentials.

const refreshLogLine = "refreshing OAuth access token"

// oauthHub is the harness Hub as seen by one test: its URL, the YouTrack service ID (the OAuth scope), and a client registered for the test.
type oauthHub struct {
	baseURL   string
	hubURL    string
	serviceID string
	clientID  string
}

func requireOAuthEnv(t *testing.T) {
	t.Helper()
	requireE2EEnv(t)
	if os.Getenv("YOUTRACK_LOGIN") == "" || os.Getenv("YOUTRACK_PASSWORD") == "" {
		t.Skip("skipping OAuth E2E test: set YOUTRACK_LOGIN and YOUTRACK_PASSWORD")
	}
}

// newOAuthHub registers an OAuth client in the harness Hub and deletes it when the test ends.
// A public client has no secret, as the CLI requires; a confidential one keeps the secret Hub generates.
func newOAuthHub(t *testing.T, public bool) *oauthHub {
	t.Helper()
	requireOAuthEnv(t)
	baseURL := strings.TrimRight(os.Getenv("YOUTRACK_BASE_URL"), "/")
	ring, err := api.GetRingConfigContext(t.Context(), baseURL)
	require.NoError(t, err)
	h := &oauthHub{baseURL: baseURL, hubURL: api.ResolveHubURL(baseURL, ring.URL), serviceID: ring.ServiceID}

	key := fmt.Sprintf("youtrack-cli-e2e-%d", time.Now().UnixNano())
	var created struct {
		ID string `json:"id"`
	}
	h.hubAdmin(t, http.MethodPost, "/api/rest/services?fields=id", map[string]any{
		"name": key, "key": key, "homeUrl": "http://127.0.0.1", "redirectUris": []string{"http://127.0.0.1"},
		"trusted": true, "authCodeFlowEnabled": true, "pkceRequired": true,
	}, &created)
	require.NotEmpty(t, created.ID)
	h.clientID = created.ID
	t.Cleanup(func() { h.hubAdmin(t, http.MethodDelete, "/api/rest/services/"+created.ID, nil, nil) })
	if public {
		h.hubAdmin(t, http.MethodPost, "/api/rest/services/"+created.ID, map[string]any{"secret": ""}, nil)
	}
	return h
}

// hubAdmin calls the Hub REST API with the harness administrator token.
func (h *oauthHub) hubAdmin(t *testing.T, method, path string, body, out any) {
	t.Helper()
	var r io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		require.NoError(t, err)
		r = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, h.hubURL+path, r)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+os.Getenv("YOUTRACK_TOKEN"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	require.Less(t, resp.StatusCode, 300, "%s %s: %s", method, path, data)
	if out != nil {
		require.NoError(t, json.Unmarshal(data, out))
	}
}

// approve plays the user approving in the browser: the signed-in user opens the authorization URL and Hub
// redirects to redirect_uri with a code. It returns that final redirect URL without following it.
func (h *oauthHub) approve(t *testing.T, authorizeURL string) string {
	t.Helper()
	u, err := url.Parse(authorizeURL)
	require.NoError(t, err)
	q := u.Query()
	require.Equal(t, "S256", q.Get("code_challenge_method"), "PKCE must use S256")
	require.NotEmpty(t, q.Get("code_challenge"))
	require.Equal(t, "offline", q.Get("access_type"), "a refresh token must be requested")
	require.Equal(t, h.clientID, q.Get("client_id"))
	require.Contains(t, q.Get("redirect_uri"), "http://127.0.0.1:")

	req, err := http.NewRequest(http.MethodGet, authorizeURL, nil)
	require.NoError(t, err)
	req.SetBasicAuth(os.Getenv("YOUTRACK_LOGIN"), os.Getenv("YOUTRACK_PASSWORD"))
	noFollow := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := noFollow.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	redirect := resp.Header.Get("Location")
	require.True(t, strings.HasPrefix(redirect, q.Get("redirect_uri")), "Hub did not approve (HTTP %d, Location %q)", resp.StatusCode, redirect)
	require.Contains(t, redirect, "code=")
	return redirect
}

// login signs in through the loopback flow and returns the saved profile.
func (h *oauthHub) login(t *testing.T, home, alias string) map[string]any {
	t.Helper()
	p := startLogin(t, home, h.clientID, h.baseURL, "--alias", alias)
	resp, err := http.Get(h.approve(t, p.authorizeURL(t)))
	require.NoError(t, err)
	resp.Body.Close()
	require.NoError(t, p.wait(t), "stderr:\n%s", p.errText())
	inst := savedInstance(t, home, alias)
	require.NotEmpty(t, inst["refreshToken"], "Hub must issue a refresh token")
	return inst
}

// patchInstance edits a saved profile in the config file (e.g. to move its expiry into the refresh window).
func patchInstance(t *testing.T, home, alias string, edit func(inst map[string]any)) {
	t.Helper()
	cfg := readConfig(t, home)
	inst := cfg["instances"].(map[string]any)[alias].(map[string]any)
	edit(inst)
	data, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configFile(home), data, 0600))
}

func readConfig(t *testing.T, home string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(configFile(home))
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

func configExists(home string) bool {
	_, err := os.Stat(configFile(home))
	return err == nil
}

// isolatedEnv is a minimal environment: no inherited YOUTRACK_* credentials, file-only token storage.
func isolatedEnv(home string, extra ...string) []string {
	return append([]string{"PATH=" + os.Getenv("PATH"), "HOME=" + home, "USERPROFILE=" + home, "YOUTRACK_DISABLE_KEYRING=1", "YOUTRACK_NO_UPDATE=1"}, extra...)
}

// runInHome runs the CLI in the isolated environment and returns stdout and stderr separately.
func runInHome(home string, extraEnv []string, args ...string) (stdout, stderr string, err error) {
	cmd := exec.Command(binaryPath, args...)
	cmd.Env = isolatedEnv(home, extraEnv...)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	err = cmd.Run()
	return out.String(), errOut.String(), err
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

func startLogin(t *testing.T, home, clientID string, args ...string) *loginProc {
	t.Helper()
	base := []string{"--json", "instance", "login", "--session", "--no-browser", "--client-id", clientID, "--timeout", "20s"}
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
	hub := newOAuthHub(t, true)
	home := t.TempDir()

	p := startLogin(t, home, hub.clientID, hub.baseURL, "--alias", "e2e")
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
	assert.Equal(t, os.Getenv("YOUTRACK_LOGIN"), out["login"])
	assert.Equal(t, true, out["hasRefreshToken"])
	assert.Equal(t, hub.hubURL, out["hubUrl"])
	assert.Equal(t, hub.clientID, out["clientId"])
	assert.Contains(t, out["token"], "****", "the access token is masked without --show-token")

	inst := savedInstance(t, home, "e2e")
	require.NotNil(t, inst)
	assert.Equal(t, "oauth", inst["authMethod"])
	assert.Equal(t, hub.baseURL, inst["url"])
	assert.Equal(t, hub.hubURL, inst["hubUrl"])
	assert.Equal(t, hub.clientID, inst["clientId"])
	assert.Equal(t, "Bearer", inst["tokenType"])
	assert.Equal(t, hub.serviceID, inst["scope"])
	assert.Equal(t, hub.hubURL+"/api/rest/oauth2/token", inst["tokenEndpoint"])
	assert.NotEmpty(t, inst["refreshToken"])
	assert.NotEmpty(t, inst["token"])
	expiresAt, _ := inst["tokenExpiresAt"].(float64)
	assert.InDelta(t, time.Now().Unix()+3600, expiresAt, 60, "expiry is absolute: receipt time + expires_in")

	// Tokens appear in neither stdout nor stderr.
	for _, secret := range []string{inst["token"].(string), inst["refreshToken"].(string)} {
		assert.NotContains(t, p.stdout.String(), secret)
		assert.NotContains(t, p.errText(), secret)
	}
	if runtime.GOOS != "windows" {
		fi, err := os.Stat(configFile(home))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), fi.Mode().Perm())
	}

	// The saved session works against YouTrack.
	stdout, stderr, err := runInHome(home, nil, "--json", "user", "current")
	require.NoError(t, err, "stderr: %s", stderr)
	assert.Contains(t, stdout, `"login": "`+os.Getenv("YOUTRACK_LOGIN")+`"`)
}

func TestOAuthLogin_StateMismatchRejected(t *testing.T) {
	hub := newOAuthHub(t, true)
	home := t.TempDir()

	p := startLogin(t, home, hub.clientID, hub.baseURL, "--alias", "e2e")
	redirect := hub.approve(t, p.authorizeURL(t))

	forged, err := url.Parse(redirect)
	require.NoError(t, err)
	q := forged.Query()
	q.Set("code", "forged-code")
	q.Set("state", "attacker-state")
	forged.RawQuery = q.Encode()

	resp, err := http.Get(forged.String())
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "a callback with the wrong state must be rejected")

	// The genuine callback still completes: the forged callback did not consume the single accepted callback.
	resp, err = http.Get(redirect)
	require.NoError(t, err)
	resp.Body.Close()
	require.NoError(t, p.wait(t), "stderr:\n%s", p.errText())
	assert.NotEmpty(t, savedInstance(t, home, "e2e")["refreshToken"])
}

func TestOAuthLogin_AccessDenied(t *testing.T) {
	hub := newOAuthHub(t, true)
	home := t.TempDir()

	p := startLogin(t, home, hub.clientID, hub.baseURL, "--alias", "e2e")
	u, err := url.Parse(p.authorizeURL(t))
	require.NoError(t, err)
	// Hub's redirect when the user clicks "Deny".
	denied := u.Query().Get("redirect_uri") + "?" + url.Values{"error": {"access_denied"}, "state": {u.Query().Get("state")}}.Encode()

	resp, err := http.Get(denied)
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, string(body), "Authentication failed")

	require.Error(t, p.wait(t))
	assert.Contains(t, p.errText(), "access_denied")
	assert.False(t, configExists(home), "nothing is saved after a denied authorization")
}

func TestOAuthLogin_TokenExchangeFailure(t *testing.T) {
	t.Run("unknown code", func(t *testing.T) {
		hub := newOAuthHub(t, true)
		home := t.TempDir()

		p := startLogin(t, home, hub.clientID, hub.baseURL, "--alias", "e2e")
		// A callback with this attempt's state but a code Hub never issued.
		redirect := regexp.MustCompile(`code=[^&]+`).ReplaceAllString(hub.approve(t, p.authorizeURL(t)), "code=never-issued")
		resp, err := http.Get(redirect)
		require.NoError(t, err)
		resp.Body.Close()

		require.Error(t, p.wait(t))
		assert.Contains(t, p.errText(), "exchanging authorization code")
		assert.Contains(t, p.errText(), "Unknown authorization code")
		assert.False(t, configExists(home), "nothing is saved when the exchange fails")
	})
	t.Run("client with a secret", func(t *testing.T) {
		hub := newOAuthHub(t, false)
		home := t.TempDir()

		p := startLogin(t, home, hub.clientID, hub.baseURL, "--alias", "e2e")
		resp, err := http.Get(hub.approve(t, p.authorizeURL(t)))
		require.NoError(t, err)
		resp.Body.Close()

		require.Error(t, p.wait(t))
		assert.Contains(t, p.errText(), "exchanging authorization code")
		assert.Contains(t, p.errText(), "Require PKCE with no Client secret", "the error explains how to fix the client")
		assert.False(t, configExists(home))
	})
}

func TestOAuthLogin_NoBrowserPastedRedirectURL(t *testing.T) {
	hub := newOAuthHub(t, true)
	home := t.TempDir()

	p := startLogin(t, home, hub.clientID, hub.baseURL, "--alias", "e2e")
	redirect := hub.approve(t, p.authorizeURL(t))

	// The user copies the URL from the browser on another device (the loopback request never arrives).
	_, err := io.WriteString(p.stdin, "not a url\n"+redirect+"\n")
	require.NoError(t, err)
	require.NoError(t, p.wait(t), "stderr:\n%s", p.errText())
	assert.Contains(t, p.errText(), "Paste that full URL here")
	assert.Contains(t, p.errText(), "paste the full URL", "an invalid paste is explained and the prompt keeps waiting")
	assert.NotEmpty(t, savedInstance(t, home, "e2e")["refreshToken"])
}

func TestOAuthLogin_PastedURLWithWrongStateRejected(t *testing.T) {
	hub := newOAuthHub(t, true)
	home := t.TempDir()

	p := startLogin(t, home, hub.clientID, hub.baseURL, "--alias", "e2e")
	redirect := hub.approve(t, p.authorizeURL(t))
	wrong := regexp.MustCompile(`state=[^&]+`).ReplaceAllString(redirect, "state=from-another-attempt")

	_, err := io.WriteString(p.stdin, wrong+"\n")
	require.NoError(t, err)
	require.Error(t, p.wait(t))
	assert.Contains(t, p.errText(), "state mismatch")
	assert.False(t, configExists(home), "a code with the wrong state is never exchanged")
}

func TestOAuthLogin_RequiresHTTPS(t *testing.T) {
	// Plain http to a remote host is refused; the guard runs before any request.
	cmd := exec.Command(binaryPath, "instance", "login", "http://youtrack.invalid", "--session", "--no-browser", "--save=false")
	cmd.Env = isolatedEnv(t.TempDir())
	out, err := cmd.CombinedOutput()
	require.Error(t, err)
	assert.Contains(t, string(out), "refusing to sign in over plain http")

	// Loopback http passes the guard (RFC 8252 §8.3); the flow then waits for the pasted redirect URL, which stdin does not hold.
	requireOAuthEnv(t)
	cmd = exec.Command(binaryPath, "instance", "login", os.Getenv("YOUTRACK_BASE_URL"), "--session", "--no-browser", "--save=false")
	cmd.Env = isolatedEnv(t.TempDir())
	out, err = cmd.CombinedOutput()
	require.Error(t, err)
	assert.NotContains(t, string(out), "plain http")
}

func TestOAuthRefresh(t *testing.T) {
	hub := newOAuthHub(t, true)
	home := t.TempDir()
	signedIn := hub.login(t, home, "e2e")
	// Inside the 60s refresh window, so the token is refreshed before the request.
	patchInstance(t, home, "e2e", func(inst map[string]any) { inst["tokenExpiresAt"] = time.Now().Unix() + 30 })

	stdout, stderr, err := runInHome(home, nil, "--verbose", "--json", "user", "current")
	require.NoError(t, err, "stderr: %s", stderr)
	assert.Contains(t, stdout, `"login": "`+os.Getenv("YOUTRACK_LOGIN")+`"`)
	assert.Contains(t, stderr, refreshLogLine)

	inst := savedInstance(t, home, "e2e")
	// Hub signs access tokens with second-precision claims, so a refresh within the login's second may return the same token; the expiry shows the refresh was persisted.
	// Hub does not rotate refresh tokens: the same one comes back and is kept.
	assert.Equal(t, signedIn["refreshToken"], inst["refreshToken"])
	expiresAt, _ := inst["tokenExpiresAt"].(float64)
	assert.InDelta(t, time.Now().Unix()+3600, expiresAt, 60)
	matches, _ := filepath.Glob(configFile(home) + ".*.tmp")
	assert.Empty(t, matches, "atomic write leaves no temp files")

	// A fresh token is not refreshed again.
	_, stderr, err = runInHome(home, nil, "--verbose", "--json", "user", "current")
	require.NoError(t, err, "stderr: %s", stderr)
	assert.NotContains(t, stderr, refreshLogLine)
	assert.Equal(t, inst["token"], savedInstance(t, home, "e2e")["token"])
}

func TestOAuthRefresh_InvalidGrantClearsCredentials(t *testing.T) {
	hub := newOAuthHub(t, true)
	home := t.TempDir()
	signedIn := hub.login(t, home, "e2e")
	const deadRefresh = "refresh-token-unknown-to-hub"
	patchInstance(t, home, "e2e", func(inst map[string]any) {
		inst["tokenExpiresAt"] = time.Now().Unix() - 10
		inst["refreshToken"] = deadRefresh
	})

	_, stderr, err := runInHome(home, nil, "--format", "table", "user", "current")
	require.Error(t, err)
	assert.Contains(t, stderr, "expired or was revoked")
	assert.Contains(t, stderr, "youtrack auth login --session e2e")
	assert.NotContains(t, stderr, deadRefresh)

	inst := savedInstance(t, home, "e2e")
	assert.Empty(t, inst["token"], "stale access token removed")
	assert.Empty(t, inst["refreshToken"], "stale refresh token removed")
	assert.Equal(t, hub.baseURL, inst["url"], "the profile is kept for re-login")
	data, _ := os.ReadFile(configFile(home))
	assert.NotContains(t, string(data), signedIn["token"].(string))

	// Later commands keep pointing at re-login and do not retry the dead grant.
	_, stderr, err = runInHome(home, nil, "--verbose", "--format", "table", "user", "current")
	require.Error(t, err)
	assert.Contains(t, stderr, "youtrack auth login --session e2e")
	assert.NotContains(t, stderr, refreshLogLine)
}

func TestOAuthRefresh_ConcurrentProcessesRefreshOnce(t *testing.T) {
	hub := newOAuthHub(t, true)
	home := t.TempDir()
	hub.login(t, home, "e2e")
	patchInstance(t, home, "e2e", func(inst map[string]any) { inst["tokenExpiresAt"] = time.Now().Unix() + 10 })

	const n = 6
	var wg sync.WaitGroup
	errs := make([]error, n)
	stderrs := make([]string, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, stderrs[i], errs[i] = runInHome(home, nil, "--verbose", "--json", "user", "current")
		}()
	}
	wg.Wait()
	refreshes := 0
	for i := range n {
		require.NoError(t, errs[i], "process %d: %s", i, stderrs[i])
		refreshes += strings.Count(stderrs[i], refreshLogLine)
	}
	assert.Equal(t, 1, refreshes, "concurrent processes must share a single refresh")
}

func TestOAuthRefresh_VerboseOutputHasNoSecrets(t *testing.T) {
	hub := newOAuthHub(t, true)
	home := t.TempDir()
	signedIn := hub.login(t, home, "e2e")
	patchInstance(t, home, "e2e", func(inst map[string]any) { inst["tokenExpiresAt"] = time.Now().Unix() + 5 })

	cmd := exec.Command(binaryPath, "--verbose", "--json", "user", "current")
	cmd.Env = isolatedEnv(home)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "output: %s", out)
	text := string(out)
	assert.Contains(t, text, refreshLogLine, "verbose output shows the refresh")
	assert.Contains(t, text, "> GET "+hub.baseURL+"/api/users/me")

	inst := savedInstance(t, home, "e2e")
	for _, secret := range []string{signedIn["token"].(string), signedIn["refreshToken"].(string), inst["token"].(string)} {
		assert.NotContains(t, text, secret)
	}
	assert.NotContains(t, text, "Authorization")
	assert.NotContains(t, text, "Bearer")
	assert.NotContains(t, text, "refresh_token=")
}

func TestOAuthPrecedence_ExplicitTokensBeatStoredSession(t *testing.T) {
	hub := newOAuthHub(t, true)
	home := t.TempDir()
	hub.login(t, home, "e2e")
	// Expired with a dead refresh token: using the session would fail and clear it.
	const deadRefresh = "refresh-token-unknown-to-hub"
	patchInstance(t, home, "e2e", func(inst map[string]any) {
		inst["tokenExpiresAt"] = time.Now().Unix() - 10
		inst["refreshToken"] = deadRefresh
	})
	token := os.Getenv("YOUTRACK_TOKEN")

	_, stderr, err := runInHome(home, []string{"YOUTRACK_TOKEN=" + token}, "--json", "user", "current")
	require.NoError(t, err, "stderr: %s", stderr)
	_, stderr, err = runInHome(home, []string{"YOUTRACK_TOKEN=perm:not-used"}, "--token", token, "--json", "user", "current")
	require.NoError(t, err, "stderr: %s", stderr)

	assert.Equal(t, deadRefresh, savedInstance(t, home, "e2e")["refreshToken"], "an explicit token never touches the stored session")
}

func TestOAuthStatus_ShowsSessionWithoutTokens(t *testing.T) {
	hub := newOAuthHub(t, true)
	home := t.TempDir()
	inst := hub.login(t, home, "e2e")
	access, refresh := inst["token"].(string), inst["refreshToken"].(string)

	out, stderr, err := runInHome(home, nil, "--json", "auth", "status")
	require.NoError(t, err, "stderr: %s", stderr)
	var statuses []map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &statuses), "output: %s", out)
	require.Len(t, statuses, 1)
	assert.Equal(t, "authenticated", statuses[0]["status"])
	assert.Equal(t, "oauth", statuses[0]["authMethod"])
	assert.Equal(t, true, statuses[0]["hasRefreshToken"])
	assert.NotEmpty(t, statuses[0]["expiresAt"])
	assert.NotContains(t, out, access)
	assert.NotContains(t, out, refresh)

	out, stderr, err = runInHome(home, nil, "--format", "table", "--server", "e2e", "auth", "status")
	require.NoError(t, err, "stderr: %s", stderr)
	assert.Contains(t, out, "OAuth session (profile e2e)")
	assert.Contains(t, out, "refresh token stored")
	assert.NotContains(t, out, access)
}

func TestOAuthLogout_WarnsRefreshTokenStaysValid(t *testing.T) {
	hub := newOAuthHub(t, true)
	home := t.TempDir()
	hub.login(t, home, "e2e")

	stdout, stderr, err := runInHome(home, nil, "--json", "auth", "logout", "--force")
	require.NoError(t, err, "stderr: %s", stderr)
	var res map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &res))
	assert.Equal(t, "logged out", res["status"])
	assert.Contains(t, stderr, "stays valid at Hub until it expires")
	assert.Empty(t, readConfig(t, home)["instances"], "credentials removed")
}

func TestOAuthInstanceList_RefreshesExpiringSession(t *testing.T) {
	hub := newOAuthHub(t, true)
	home := t.TempDir()
	hub.login(t, home, "e2e")
	patchInstance(t, home, "e2e", func(inst map[string]any) { inst["tokenExpiresAt"] = time.Now().Unix() - 10 })

	out, stderr, err := runInHome(home, nil, "--verbose", "--json", "instance", "list")
	require.NoError(t, err, "stderr: %s", stderr)
	assert.Contains(t, stderr, refreshLogLine)
	var items []map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &items), "output: %s", out)
	require.Len(t, items, 1)
	assert.Equal(t, "authenticated", items[0]["status"], "an expired OAuth session is refreshed, not reported unavailable")
	expiresAt, _ := savedInstance(t, home, "e2e")["tokenExpiresAt"].(float64)
	assert.InDelta(t, time.Now().Unix()+3600, expiresAt, 60, "the refreshed session is persisted")
}

func TestOAuthStatus_FailedCheckKeepsSessionMetadata(t *testing.T) {
	hub := newOAuthHub(t, true)
	home := t.TempDir()
	hub.login(t, home, "e2e")
	const deadRefresh = "refresh-token-unknown-to-hub"
	patchInstance(t, home, "e2e", func(inst map[string]any) {
		inst["tokenExpiresAt"] = time.Now().Unix() - 10
		inst["refreshToken"] = deadRefresh
	})

	// First check: the refresh is rejected while checking. Second check: the cleared profile is reported as such, without a new refresh.
	for i, wantPrefix := range []string{"invalid token: ", "not configured: "} {
		out, stderr, err := runInHome(home, nil, "--verbose", "--json", "--server", "e2e", "auth", "status")
		require.NoError(t, err, "stderr: %s", stderr)
		var s map[string]any
		require.NoError(t, json.Unmarshal([]byte(out), &s), "output: %s", out)
		assert.True(t, strings.HasPrefix(s["status"].(string), wantPrefix), "status: %v", s["status"])
		assert.Contains(t, s["status"], "youtrack auth login --session e2e")
		assert.Equal(t, "profile", s["source"])
		assert.Equal(t, "e2e", s["alias"])
		assert.Equal(t, "oauth", s["authMethod"])
		assert.Equal(t, false, s["hasRefreshToken"])
		assert.NotContains(t, out, deadRefresh)
		assert.Equal(t, i == 0, strings.Contains(stderr, refreshLogLine), "the dead grant is not retried")
	}
}
