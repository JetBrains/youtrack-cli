package api_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeBrowserSignInInstance serves the YouTrack and embedded Hub endpoints that browser sign-in calls, issuing "code-1" as the only valid code.
func fakeBrowserSignInInstance(t *testing.T) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/config":
			_, _ = io.WriteString(w, `{"ring":{"url":"`+srv.URL+`/hub","serviceId":"yt-service"}}`)
		case r.URL.Path == "/hub/api/rest/oauth2/auth":
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == "/hub/api/rest/oauth2/token":
			if r.FormValue("code") != "code-1" || r.FormValue("code_verifier") == "" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, `{"error":"invalid_grant"}`)
				return
			}
			_, _ = io.WriteString(w, `{"access_token":"access-1","token_type":"Bearer","expires_in":3600,"scope":"yt-service"}`)
		case r.URL.Path == "/api/users/me" && r.Header.Get("Authorization") == "Bearer access-1":
			_, _ = io.WriteString(w, `{"login":"jane","fullName":"Jane Doe"}`)
		case r.URL.Path == "/hub/api/rest/users/me/permanenttokens" && r.Method == http.MethodPost:
			_, _ = io.WriteString(w, `{"id":"t1","name":"youtrack-cli","token":"perm-amFuZQ==.dDE=.secret"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func webLoginCmd(t *testing.T, home string, args ...string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	// An empty PATH makes opening the browser fail, standing in for a machine where it does not open.
	cmd.Env = []string{"PATH=" + t.TempDir(), "HOME=" + home, "YOUTRACK_DISABLE_KEYRING=1", "YOUTRACK_NO_UPDATE=1"}
	cmd.Stdin = strings.NewReader("")
	return cmd
}

func TestAuthLogin_WebSignsInWithoutTerminal(t *testing.T) {
	srv := fakeBrowserSignInInstance(t)
	home := t.TempDir()

	cmd := webLoginCmd(t, home, "--json", "auth", "login", srv.URL, "--web", "--alias", "app")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	stderr, err := cmd.StderrPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	t.Cleanup(func() { _ = cmd.Process.Kill() })

	// With JSON output, stderr carries only JSON lines; the first one is the authorization URL.
	var event struct {
		AuthorizationURL string `json:"authorizationUrl"`
	}
	lines := bufio.NewScanner(stderr)
	require.True(t, lines.Scan(), "no authorization URL line on stderr")
	require.NoError(t, json.Unmarshal(lines.Bytes(), &event), "stderr line: %s", lines.Text())
	authorize, err := url.Parse(event.AuthorizationURL)
	require.NoError(t, err)
	assert.Equal(t, srv.URL+"/hub/api/rest/oauth2/auth", authorize.Scheme+"://"+authorize.Host+authorize.Path)

	// Complete the sign-in the way the Hub redirect would.
	q := authorize.Query()
	callback := q.Get("redirect_uri") + "/?" + url.Values{"code": {"code-1"}, "state": {q.Get("state")}}.Encode()
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Get(callback)
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	rest, _ := io.ReadAll(stderr)
	require.NoError(t, cmd.Wait(), "stderr: %s", rest)
	for _, line := range strings.Split(strings.TrimSpace(string(rest)), "\n") {
		if line != "" {
			assert.True(t, json.Valid([]byte(line)), "stderr line is not JSON: %s", line)
		}
	}

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result), "stdout: %s", stdout.String())
	assert.Equal(t, "jane", result["login"])
	assert.NotContains(t, stdout.String(), "secret")

	data, err := os.ReadFile(configFile(home))
	require.NoError(t, err)
	assert.Contains(t, string(data), `"app"`)
	assert.Contains(t, string(data), "perm-amFuZQ==.dDE=.secret")
}

func TestAuthLogin_WebFailsInsteadOfPrompting(t *testing.T) {
	for name, tc := range map[string]struct {
		args []string
		want string
	}{
		"no instance":         {[]string{"auth", "login", "--web"}, "--web needs the instance"},
		"no browser":          {[]string{"auth", "login", "https://youtrack.invalid", "--web", "--no-browser"}, "[no-browser web] were all set"},
		"basic auth":          {[]string{"auth", "login", "https://youtrack.invalid", "--web", "--login", "x"}, "[login web] were all set"},
		"no oauth client":     {[]string{"auth", "login", "--web"}, "browser sign-in is not available: this Hub has no public CLI OAuth client"},
		"no youtrack service": {[]string{"auth", "login", "--web"}, "browser sign-in is not available: could not discover the YouTrack service ID"},
	} {
		t.Run(name, func(t *testing.T) {
			args := tc.args
			switch name {
			case "no oauth client":
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/api/config" {
						_, _ = io.WriteString(w, `{"ring":{"url":"/hub","serviceId":"yt-service"}}`)
						return
					}
					http.NotFound(w, r)
				}))
				t.Cleanup(srv.Close)
				args = append(args, srv.URL)
			case "no youtrack service":
				srv := httptest.NewServer(http.NotFoundHandler())
				t.Cleanup(srv.Close)
				args = append(args, srv.URL)
			}
			cmd := webLoginCmd(t, t.TempDir(), append([]string{"--plain"}, args...)...)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			done := make(chan error, 1)
			require.NoError(t, cmd.Start())
			go func() { done <- cmd.Wait() }()
			select {
			case err := <-done:
				require.Error(t, err)
			case <-time.After(30 * time.Second):
				_ = cmd.Process.Kill()
				t.Fatalf("--web waited instead of failing; stderr: %s", stderr.String())
			}
			assert.Contains(t, stderr.String(), tc.want)
			assert.NotContains(t, stderr.String(), "Paste your token")
		})
	}
}
