//go:build !windows

package api_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
