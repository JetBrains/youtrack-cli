//go:build !windows

package api_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/creack/pty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedBufferedOutputWithTerminalStdout(t *testing.T) {
	requireE2EEnv(t)
	binary := filepath.Join(t.TempDir(), "embedded-youtrack")
	build := exec.Command("go", "build", "-o", binary, "./testdata/embedding")
	out, err := build.CombinedOutput()
	require.NoError(t, err, "%s", out)
	for _, mode := range []string{"provider", "error"} {
		t.Run(mode, func(t *testing.T) {
			terminal, stdout, err := pty.Open()
			require.NoError(t, err)
			t.Cleanup(func() { _ = terminal.Close(); _ = stdout.Close() })
			command := exec.Command(binary, "user", "current")
			captured := filepath.Join(t.TempDir(), "output.json")
			command.Env = append(os.Environ(), "HOME="+t.TempDir(),
				"EMBED_BUFFER_OUTPUT="+captured, "EMBED_MODE="+mode,
				"EMBED_BASE_URL="+os.Getenv("YOUTRACK_BASE_URL"), "EMBED_TOKEN="+os.Getenv("YOUTRACK_TOKEN"))
			var stderr bytes.Buffer
			command.Stdout, command.Stderr = stdout, &stderr
			err = command.Run()
			if mode == "error" {
				require.Error(t, err)
				assertEmbeddedError(t, stderr.String(), "internal_error", "embedded provider failed", "")
				return
			}
			require.NoError(t, err, "%s", stderr.String())
			output, readErr := os.ReadFile(captured)
			require.NoError(t, readErr)
			var user struct {
				Login string `json:"login"`
			}
			require.NoError(t, json.Unmarshal(output, &user), "%s", output)
			assert.NotEmpty(t, user.Login)
		})
	}
}
