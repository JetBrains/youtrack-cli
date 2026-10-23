//go:build !windows

package api_test

import (
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigWritersWaitForLock checks that every command rewriting the config takes the same lock as OAuth refresh,
// so none of them can overwrite a refresh token that another process rotates in the meantime.
func TestConfigWritersWaitForLock(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		stdin string
		check func(t *testing.T, home string)
	}{
		{args: []string{"instance", "use", "fake"}},
		{args: []string{"instance", "delete", "fake", "--force"}},
		{args: []string{"auth", "logout", "--force"}},
		{args: []string{"auth", "refresh-token", "fake"}, stdin: "2\n{token}\n", check: func(t *testing.T, home string) {
			assert.Equal(t, os.Getenv("YOUTRACK_TOKEN"), savedInstance(t, home, "fake")["token"])
		}},
		{args: []string{"auth", "get-token", "--alias", "other"}, stdin: "{url}\n3\n{token}\n", check: func(t *testing.T, home string) {
			assert.Equal(t, os.Getenv("YOUTRACK_TOKEN"), savedInstance(t, home, "other")["token"])
		}},
	} {
		args := tc.args
		t.Run(args[0]+" "+args[1], func(t *testing.T) {
			hub := newOAuthHub(t, true)
			home := t.TempDir()
			hub.login(t, home, "fake")

			lock, err := os.OpenFile(configFile(home)+".lock", os.O_RDWR|os.O_CREATE, 0600)
			require.NoError(t, err)
			defer func() { _ = lock.Close() }()
			require.NoError(t, syscall.Flock(int(lock.Fd()), syscall.LOCK_EX))

			cmd := exec.Command(binaryPath, args...)
			cmd.Env = isolatedEnv(home)
			cmd.Stdin = strings.NewReader(strings.NewReplacer("{url}", hub.baseURL, "{token}", os.Getenv("YOUTRACK_TOKEN")).Replace(tc.stdin))
			require.NoError(t, cmd.Start())
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()

			select {
			case err := <-done:
				t.Fatalf("%v finished while another process held the config lock (err: %v)", args, err)
			case <-time.After(700 * time.Millisecond):
			}
			require.NoError(t, syscall.Flock(int(lock.Fd()), syscall.LOCK_UN))
			select {
			case err := <-done:
				require.NoError(t, err)
				if tc.check != nil {
					tc.check(t, home)
				}
			case <-time.After(15 * time.Second):
				_ = cmd.Process.Kill()
				t.Fatalf("%v did not finish after the config lock was released", args)
			}
		})
	}
}
