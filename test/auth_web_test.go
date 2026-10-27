//go:build !windows

package api_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

func webLoginCmd(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	// An empty PATH makes opening the browser fail, standing in for a machine where it does not open.
	cmd.Env = []string{"PATH=" + t.TempDir(), "HOME=" + t.TempDir(), "YOUTRACK_DISABLE_KEYRING=1", "YOUTRACK_NO_UPDATE=1"}
	cmd.Stdin = strings.NewReader("")
	return cmd
}

// TestAuthLogin_WebSignsInWithoutTerminal runs a full --web sign-in against the harness Hub with stdin, stdout, and stderr all pipes, as an app would.
func TestAuthLogin_WebSignsInWithoutTerminal(t *testing.T) {
	h := newHarnessHub(t)
	name := fmt.Sprintf("youtrack-cli-e2e-web-%d", time.Now().UnixNano())
	h.deleteTokensNamed(t, name)

	cmd := webLoginCmd(t, "--json", "auth", "login", h.baseURL, "--web", "--client-id", h.clientID, "--alias", "app", "--token-name", name)
	home := envHome(cmd.Env)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	stderr, err := cmd.StderrPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	t.Cleanup(func() { _ = cmd.Process.Kill() })

	// With JSON output, the first stderr line is the authorization URL as JSON; no progress text comes before it.
	var event struct {
		AuthorizationURL string `json:"authorizationUrl"`
	}
	lines := bufio.NewReader(stderr)
	first, err := lines.ReadBytes('\n')
	require.NoError(t, err, "no authorization URL line on stderr")
	require.NoError(t, json.Unmarshal(first, &event), "stderr line: %s", first)
	authorize, err := url.Parse(event.AuthorizationURL)
	require.NoError(t, err)
	assert.Equal(t, h.clientID, authorize.Query().Get("client_id"))

	h.approve(t, authorize)

	rest, _ := io.ReadAll(lines)
	require.NoError(t, cmd.Wait(), "stderr: %s", rest)
	assert.Empty(t, strings.TrimSpace(string(rest)), "no text after the URL line")

	var result struct {
		Login string `json:"login"`
		Token string `json:"token"`
	}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result), "stdout: %s", stdout.String())
	assert.Equal(t, os.Getenv("YOUTRACK_LOGIN"), result.Login)

	saved := savedToken(t, []string{"HOME=" + home}, "app")
	_, id, ok := api.ParsePermanentToken(saved)
	require.True(t, ok, "the saved token is a permanent token: %q", saved)
	assert.Equal(t, name, h.hubTokenNames(t)[id])
	assert.NotContains(t, stdout.String(), saved, "the token is masked")
}

func TestAuthLogin_WebFailsInsteadOfPrompting(t *testing.T) {
	cases := map[string]struct {
		args []string
		want string
		live bool
	}{
		"no instance": {[]string{"auth", "login", "--web"}, "--web needs the instance", false},
		"no browser":  {[]string{"auth", "login", "https://youtrack.invalid", "--web", "--no-browser"}, "[no-browser web] were all set", false},
		"basic auth":  {[]string{"auth", "login", "https://youtrack.invalid", "--web", "--login", "x"}, "[login web] were all set", false},
		// The harness Hub has no "localhost-auth" client, so a terminal run would fall back to pasting a token here.
		"no oauth client": {[]string{"auth", "login", "--web"}, "browser sign-in is not available: this Hub has no public CLI OAuth client", true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			args := tc.args
			if tc.live {
				requireE2EEnv(t)
				args = append(args, os.Getenv("YOUTRACK_BASE_URL"))
			}
			cmd := webLoginCmd(t, append([]string{"--plain"}, args...)...)
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
