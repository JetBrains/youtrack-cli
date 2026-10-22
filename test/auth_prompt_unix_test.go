//go:build !windows

package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

// ptySession drives the CLI through a sized pty, so term.IsTerminal is true and huh renders its fields. Tests run with TERM=tmux-256color, for which termenv skips its terminal color query; the pty never answers it, so each run would otherwise stall for termenv.OSCTimeout.
type ptySession struct {
	t    *testing.T
	ptmx *os.File
	cmd  *exec.Cmd
	mu   sync.Mutex
	out  bytes.Buffer
	done chan struct{}
}

func startPTY(t *testing.T, env []string, args ...string) *ptySession {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Env = env
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: 120, Rows: 40})
	require.NoError(t, err)
	s := &ptySession{t: t, ptmx: ptmx, cmd: cmd, done: make(chan struct{})}
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			s.mu.Lock()
			s.out.Write(buf[:n])
			s.mu.Unlock()
			if err != nil {
				close(s.done)
				return
			}
		}
	}()
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = ptmx.Close()
	})
	return s
}

func (s *ptySession) output() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.out.String()
}

func (s *ptySession) waitFor(text string) {
	s.t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for !strings.Contains(s.output(), text) {
		if time.Now().After(deadline) {
			s.t.Fatalf("timed out waiting for %q; output so far:\n%s", text, s.output())
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (s *ptySession) send(keys string) {
	s.t.Helper()
	_, err := s.ptmx.WriteString(keys)
	require.NoError(s.t, err)
}

func (s *ptySession) wait() error {
	s.t.Helper()
	errCh := make(chan error, 1)
	go func() { errCh <- s.cmd.Wait() }()
	select {
	case err := <-errCh:
		<-s.done
		return err
	case <-time.After(30 * time.Second):
		s.t.Fatalf("command did not exit; output so far:\n%s", s.output())
		return nil
	}
}

func TestAuthLogin_InteractivePromptsPrefillURLFromConfig(t *testing.T) {
	requireE2EEnv(t)
	token := os.Getenv("YOUTRACK_TOKEN")
	baseURL := os.Getenv("YOUTRACK_BASE_URL")

	home := t.TempDir()
	cfg := `{"currentInstance":"harness","instances":{"harness":{"url":"` + baseURL + `"}}}`
	require.NoError(t, os.WriteFile(filepath.Join(home, ".youtrack-cli-config.json"), []byte(cfg), 0o600))

	s := startPTY(t, []string{"PATH=" + os.Getenv("PATH"), "HOME=" + home, "YOUTRACK_DISABLE_KEYRING=1", "TERM=tmux-256color"},
		"--plain", "auth", "login", "--no-browser", "--save=false")

	s.waitFor("YouTrack URL")
	s.waitFor(baseURL) // pre-filled from the active instance
	s.send("\r")
	s.waitFor("Paste your token")
	s.send(token + "\r")
	require.NoError(t, s.wait(), "output:\n%s", s.output())

	out := s.output()
	assert.Contains(t, out, "Press Ctrl+C at any time to cancel")
	assert.Contains(t, out, baseURL+"/users/me?tab=account-security")
	assert.Contains(t, out, "Authenticated as")
	assert.NotContains(t, out, token, "the pasted token must not be echoed")
}

func TestAuthLogin_InteractivePromptCtrlCCancels(t *testing.T) {
	s := startPTY(t, []string{"PATH=" + os.Getenv("PATH"), "HOME=" + t.TempDir(), "YOUTRACK_DISABLE_KEYRING=1", "TERM=tmux-256color"},
		"auth", "login", "--no-browser", "--save=false")

	s.waitFor("https://youtrack.jetbrains.com") // the default when no instance is configured
	s.send("\x03")
	_ = s.wait()
	assert.Contains(t, s.output(), "Canceled.")
}

// ptyEnv isolates a pty run: a fresh HOME, no keyring, and no browser launcher on PATH, so browser sign-in prints the sign-in URL and waits instead of opening a real browser.
func ptyEnv(t *testing.T) []string {
	t.Helper()
	return []string{"PATH=" + t.TempDir(), "HOME=" + t.TempDir(), "YOUTRACK_DISABLE_KEYRING=1", "TERM=tmux-256color"}
}

// harnessHub is the harness's real Hub with a public PKCE client registered for one test, as an administrator registers the CLI's client.
type harnessHub struct {
	baseURL   string
	hubURL    string
	serviceID string
	clientID  string
}

func newHarnessHub(t *testing.T) *harnessHub {
	t.Helper()
	requireE2EEnv(t)
	if os.Getenv("YOUTRACK_LOGIN") == "" || os.Getenv("YOUTRACK_PASSWORD") == "" {
		t.Skip("set YOUTRACK_LOGIN and YOUTRACK_PASSWORD to approve sign-ins in the harness Hub")
	}
	baseURL := strings.TrimRight(os.Getenv("YOUTRACK_BASE_URL"), "/")
	ring, err := api.GetRingConfigContext(t.Context(), baseURL)
	require.NoError(t, err)
	h := &harnessHub{baseURL: baseURL, hubURL: api.ResolveHubURL(baseURL, ring.URL), serviceID: ring.ServiceID}

	key := fmt.Sprintf("youtrack-cli-e2e-%d", time.Now().UnixNano())
	var created struct {
		ID string `json:"id"`
	}
	h.hubAdmin(t, http.MethodPost, "/api/rest/services?fields=id", map[string]any{
		"name": key, "key": key, "homeUrl": "http://127.0.0.1", "redirectUris": []string{"http://127.0.0.1"},
		"trusted": true, "authCodeFlowEnabled": true, "pkceRequired": true,
	}, &created)
	require.NotEmpty(t, created.ID)
	t.Cleanup(func() { h.hubAdmin(t, http.MethodDelete, "/api/rest/services/"+created.ID, nil, nil) })
	h.hubAdmin(t, http.MethodPost, "/api/rest/services/"+created.ID, map[string]any{"secret": ""}, nil) // public: no client secret
	h.clientID = created.ID
	return h
}

// hubAdmin calls the harness Hub REST API as the harness administrator.
func (h *harnessHub) hubAdmin(t *testing.T, method, path string, body, out any) {
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
	defer func() { _ = resp.Body.Close() }()
	data, _ := io.ReadAll(resp.Body)
	require.Less(t, resp.StatusCode, 300, "%s %s: %s", method, path, data)
	if out != nil {
		require.NoError(t, json.Unmarshal(data, out))
	}
}

var authorizeURLRe = regexp.MustCompile(`https?://[^\s\x1b]+/oauth2/auth\?[^\s\x1b]+`)

// authorizeURL waits for the sign-in URL the CLI prints and parses it.
func (s *ptySession) authorizeURL() *url.URL {
	s.t.Helper()
	s.waitFor("use this URL")
	deadline := time.Now().Add(5 * time.Second)
	for {
		if m := authorizeURLRe.FindString(s.output()); m != "" {
			u, err := url.Parse(m)
			require.NoError(s.t, err)
			return u
		}
		if time.Now().After(deadline) {
			s.t.Fatalf("no authorize URL in output:\n%s", s.output())
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// approve plays the user approving in the browser: the harness user opens the authorize URL, Hub redirects to the CLI's loopback callback, and the callback is delivered.
func (h *harnessHub) approve(t *testing.T, authorize *url.URL) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, authorize.String(), nil)
	require.NoError(t, err)
	req.SetBasicAuth(os.Getenv("YOUTRACK_LOGIN"), os.Getenv("YOUTRACK_PASSWORD"))
	noFollow := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := noFollow.Do(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	callback := resp.Header.Get("Location")
	require.True(t, strings.HasPrefix(callback, authorize.Query().Get("redirect_uri")), "Hub did not approve (HTTP %d, Location %q)", resp.StatusCode, callback)
	require.Contains(t, callback, "code=")
	resp, err = http.Get(callback)
	require.NoError(t, err)
	_ = resp.Body.Close()
}

// runInEnv runs the CLI with exactly env (no pty) and returns its combined output.
func runInEnv(env []string, args ...string) (string, error) {
	cmd := exec.Command(binaryPath, args...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port
}

// envHome returns the HOME of a ptyEnv environment.
func envHome(env []string) string {
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, "HOME="); ok {
			return v
		}
	}
	return ""
}

// savedToken returns the token saved for alias in the isolated config (the keyring is disabled there).
func savedToken(t *testing.T, env []string, alias string) string {
	t.Helper()
	data, err := os.ReadFile(configFile(envHome(env)))
	require.NoError(t, err)
	var cfg struct {
		Instances map[string]struct {
			Token string `json:"token"`
		} `json:"instances"`
	}
	require.NoError(t, json.Unmarshal(data, &cfg))
	return cfg.Instances[alias].Token
}

// hubTokenNames lists the harness user's permanent tokens by ID.
func (h *harnessHub) hubTokenNames(t *testing.T) map[string]string {
	t.Helper()
	var page struct {
		Tokens []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"permanenttokens"`
	}
	h.hubAdmin(t, http.MethodGet, "/api/rest/users/me/permanenttokens?fields=id,name&$top=1000", nil, &page)
	names := map[string]string{}
	for _, tok := range page.Tokens {
		names[tok.ID] = tok.Name
	}
	return names
}

// deleteTokensNamed removes the harness user's permanent tokens called name when the test ends, so runs don't pile them up.
func (h *harnessHub) deleteTokensNamed(t *testing.T, name string) {
	t.Helper()
	t.Cleanup(func() {
		for id, n := range h.hubTokenNames(t) {
			if n == name {
				h.hubAdmin(t, http.MethodDelete, "/api/rest/users/me/permanenttokens/"+url.PathEscape(id), nil, nil)
			}
		}
	})
}

// signIn runs a browser sign-in in a pty, approves it as the harness user, and returns the CLI output.
func (h *harnessHub) signIn(t *testing.T, env []string, check func(q url.Values), args ...string) string {
	t.Helper()
	s := startPTY(t, env, append([]string{"--plain", "--base-url", h.baseURL, "auth", "login", "--client-id", h.clientID}, args...)...)
	authorize := s.authorizeURL()
	if check != nil {
		check(authorize.Query())
	}
	h.approve(t, authorize)
	require.NoError(t, s.wait(), "output:\n%s", s.output())
	return s.output()
}

func TestAuthLogin_BrowserSignInCreatesPermanentToken(t *testing.T) {
	h := newHarnessHub(t)
	env := ptyEnv(t)
	name := fmt.Sprintf("youtrack-cli-e2e-token-%d", time.Now().UnixNano())
	h.deleteTokensNamed(t, name)

	out := h.signIn(t, env, func(q url.Values) {
		assert.Equal(t, h.clientID, q.Get("client_id"))
		assert.Equal(t, "S256", q.Get("code_challenge_method"))
		assert.NotEmpty(t, q.Get("code_challenge"))
		assert.Equal(t, h.serviceID+" 0-0-0-0-0", q.Get("scope"), "Hub's own scope is requested to create the token")
	}, "--alias", "browser", "--token-name", name)
	assert.Contains(t, out, fmt.Sprintf("Permanent token %q saved as \"browser\"", name))

	first := savedToken(t, env, "browser")
	_, firstID, ok := api.ParsePermanentToken(first)
	require.True(t, ok, "the saved token is a permanent token: %q", first)
	assert.Equal(t, name, h.hubTokenNames(t)[firstID])
	cli, err := runInEnv(env, "--server", "browser", "--plain", "user", "current")
	require.NoError(t, err, cli)
	assert.Contains(t, cli, os.Getenv("YOUTRACK_LOGIN"))

	// Signing in again replaces the token and revokes the old one in Hub.
	h.signIn(t, env, nil, "--alias", "browser", "--token-name", name)
	_, secondID, ok := api.ParsePermanentToken(savedToken(t, env, "browser"))
	require.True(t, ok)
	require.NotEqual(t, firstID, secondID)
	tokens := h.hubTokenNames(t)
	assert.NotContains(t, tokens, firstID, "the replaced token must be revoked")
	assert.Equal(t, name, tokens[secondID])
}

func TestAuthLogin_SessionReplacingPermanentTokenWarns(t *testing.T) {
	h := newHarnessHub(t)
	env := ptyEnv(t)
	name := fmt.Sprintf("youtrack-cli-e2e-token-%d", time.Now().UnixNano())
	h.deleteTokensNamed(t, name)

	h.signIn(t, env, nil, "--alias", "renew", "--token-name", name)
	_, permID, ok := api.ParsePermanentToken(savedToken(t, env, "renew"))
	require.True(t, ok)

	out := h.signIn(t, env, func(q url.Values) {
		assert.Equal(t, h.serviceID, q.Get("scope"), "--session does not request Hub's scope")
	}, "--alias", "renew", "--session")
	assert.Contains(t, out, `Session token saved as "renew"`)
	assert.Contains(t, out, `the permanent token previously saved for "renew" is still valid in Hub`)
	_, _, isPerm := api.ParsePermanentToken(savedToken(t, env, "renew"))
	assert.False(t, isPerm, "--session saves the access token")
	assert.Contains(t, h.hubTokenNames(t), permID, "the session token cannot revoke it, so it stays")
}

func TestAuthLogin_BrowserFlagsShapeAuthorizeURL(t *testing.T) {
	h := newHarnessHub(t)
	port := freePort(t)

	for name, tc := range map[string]struct {
		flags []string
		check func(t *testing.T, q url.Values)
	}{
		"port": {flags: []string{"--port", fmt.Sprint(port)}, check: func(t *testing.T, q url.Values) {
			assert.Equal(t, fmt.Sprintf("http://127.0.0.1:%d", port), q.Get("redirect_uri"))
		}},
		"scope": {flags: []string{"--scope", "0-0-0-0-0"}, check: func(t *testing.T, q url.Values) {
			assert.Equal(t, h.serviceID+" 0-0-0-0-0", q.Get("scope"), "Hub's scope is not requested twice")
		}},
	} {
		t.Run(name, func(t *testing.T) {
			s := startPTY(t, ptyEnv(t), append([]string{"--plain", "--base-url", h.baseURL, "auth", "login", "--save=false", "--client-id", h.clientID, "--timeout", "2s"}, tc.flags...)...)
			tc.check(t, s.authorizeURL().Query())
			require.Error(t, s.wait(), "login must end in the timeout, output:\n%s", s.output())
			assert.Contains(t, s.output(), "timeout waiting for browser callback (2s)")
		})
	}
}

func TestAuthLogin_PortFlagFailsWhenPortIsBusy(t *testing.T) {
	requireE2EEnv(t)
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = busy.Close() })
	port := busy.Addr().(*net.TCPAddr).Port

	s := startPTY(t, ptyEnv(t), "--plain", "--base-url", os.Getenv("YOUTRACK_BASE_URL"), "auth", "login", "--save=false", "--port", fmt.Sprint(port))
	require.Error(t, s.wait())
	assert.Contains(t, s.output(), fmt.Sprintf("port %d is in use", port))
}

func TestAuthLogin_FallsBackToPasteWithoutOAuthClient(t *testing.T) {
	requireE2EEnv(t)
	token := os.Getenv("YOUTRACK_TOKEN")

	// The harness Hub has no "localhost-auth" client, so the pre-flight fails and login falls back to pasting a token.
	s := startPTY(t, ptyEnv(t), "--plain", "instance", "login", os.Getenv("YOUTRACK_BASE_URL"), "--save=false")
	s.waitFor(`no public CLI OAuth client ("localhost-auth")`)
	s.waitFor("Open browser to create a token?")
	s.send("n")
	s.waitFor("Paste your token")
	s.send(token + "\r")
	require.NoError(t, s.wait(), "output:\n%s", s.output())
	assert.Contains(t, s.output(), "Authenticated as")
}

// The one test here with a stand-in server: the scenario is a hostile instance advertising a deceptive Hub URL, which the harness cannot play.
func TestAuthLogin_RejectsUnsafeHubURL(t *testing.T) {
	var hubURL string
	// Answers every request with 200 so the OAuth pre-flight would pass — only Hub URL validation stands between the advertised URL and the browser.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/config" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"ring": map[string]any{"url": hubURL, "serviceId": "svc"}})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	// Userinfo makes the link read as decoy.example while it actually targets the fake server.
	hubURL = strings.Replace(srv.URL, "://", "://decoy.example@", 1)

	s := startPTY(t, ptyEnv(t), "--plain", "--base-url", srv.URL, "auth", "login", "--save=false", "--timeout", "2s")
	s.waitFor("falling back to the paste-token flow")
	s.waitFor("Open browser to create a token?")
	s.send("n")
	s.waitFor("Paste your token")
	s.send("perm:not-a-real-token\r")
	_ = s.wait() // verification against the fake instance fails; only what was offered to the user matters.
	assert.NotContains(t, s.output(), "use this URL")
	assert.NotContains(t, s.output(), "oauth2/auth")
}

func TestAuthLogin_FlagsRejectedWithBrowserSignIn(t *testing.T) {
	requireE2EEnv(t)
	// --token-name alone is accepted: browser sign-in names the permanent token it creates. With --session there is no such token.
	for name, tc := range map[string]struct {
		flags []string
		want  string
	}{
		"insecure":             {flags: []string{"--insecure"}, want: "--insecure has no effect with browser sign-in"},
		"token-name w/session": {flags: []string{"--session", "--token-name", "build-agent"}, want: "--token-name has no effect with --session"},
	} {
		t.Run(name, func(t *testing.T) {
			s := startPTY(t, ptyEnv(t), append([]string{"--plain", "--base-url", os.Getenv("YOUTRACK_BASE_URL"), "auth", "login", "--save=false"}, tc.flags...)...)
			require.Error(t, s.wait())
			assert.Contains(t, s.output(), tc.want)
			assert.NotContains(t, s.output(), "Discovering Hub", "the flag must be rejected before sign-in starts")
		})
	}
}
