package api_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Build a real embedding application, then run its commands against the same
// live YouTrack instance as the standalone CLI tests. No API is simulated.
func TestEmbeddedCommands(t *testing.T) {
	requireE2EEnv(t)
	binary := filepath.Join(t.TempDir(), "embedded-youtrack")
	build := exec.Command("go", "build", "-o", binary, "./testdata/embedding")
	out, err := build.CombinedOutput()
	require.NoError(t, err, "%s", out)

	run := func(t *testing.T, mode string, fallback bool, args ...string) (string, string, error) {
		t.Helper()
		command := exec.Command(binary, args...)
		baseURL, token := "", ""
		if fallback {
			baseURL, token = os.Getenv("YOUTRACK_BASE_URL"), os.Getenv("YOUTRACK_TOKEN")
		}
		if mode == "isolated" {
			baseURL, token = os.Getenv("YOUTRACK_BASE_URL"), "invalid-embedding-fallback-token"
		}
		command.Env = append(os.Environ(),
			"HOME="+t.TempDir(),
			"YOUTRACK_BASE_URL="+baseURL, "YOUTRACK_TOKEN="+token,
			"EMBED_BASE_URL="+os.Getenv("YOUTRACK_BASE_URL"), "EMBED_TOKEN="+os.Getenv("YOUTRACK_TOKEN"),
			"EMBED_MODE="+mode,
		)
		var stdout, stderr bytes.Buffer
		command.Stdout, command.Stderr = &stdout, &stderr
		err := command.Run()
		return stdout.String(), stderr.String(), err
	}

	t.Run("provider without CLI credentials", func(t *testing.T) {
		stdout, stderr, err := run(t, "provider", false, "user", "current", "--fields", "login", "--plain")
		require.NoError(t, err, "%s", stderr)
		assert.NotEmpty(t, stdout)
		assert.NotContains(t, stdout, "\t")
		assert.Empty(t, stderr)
	})
	t.Run("provider can fall back", func(t *testing.T) {
		stdout, stderr, err := run(t, "fallback", true, "user", "current", "--json")
		require.NoError(t, err, "%s", stderr)
		var user struct {
			Login string `json:"login"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &user))
		assert.NotEmpty(t, user.Login)
		assert.Empty(t, stderr)
	})
	t.Run("provider is scoped to execution", func(t *testing.T) {
		stdout, stderr, err := run(t, "isolated", false, "user", "current", "--json")
		require.Error(t, err)
		assert.True(t, json.Valid([]byte(stdout)), "%s", stdout)
		assertEmbeddedError(t, stderr, "unauthorized", "HTTP 401", "embedded-youtrack auth get-token")
	})
	t.Run("provider error", func(t *testing.T) {
		stdout, stderr, err := run(t, "error", false, "user", "current", "--json")
		require.Error(t, err)
		assert.Empty(t, stdout)
		assertEmbeddedError(t, stderr, "internal_error", "embedded provider failed", "")
	})
	t.Run("command context", func(t *testing.T) {
		stdout, stderr, err := run(t, "canceled", false, "user", "current", "--json")
		require.Error(t, err)
		assert.Empty(t, stdout)
		assertEmbeddedError(t, stderr, "internal_error", "context canceled", "")
	})
	t.Run("custom HTTP client", func(t *testing.T) {
		stdout, stderr, err := run(t, "timeout", false, "user", "current", "--json")
		require.Error(t, err)
		assert.Empty(t, stdout)
		assertEmbeddedError(t, stderr, "internal_error", "deadline exceeded", "")
	})
	t.Run("shared output formatting", func(t *testing.T) {
		stdout, stderr, err := run(t, "provider", false, "embedded-output", "--json")
		require.NoError(t, err, "%s", stderr)
		assert.JSONEq(t, `{"format":"json"}`, stdout)
		stdout, stderr, err = run(t, "provider", false, "embedded-output", "--plain", "--fields", "format")
		require.NoError(t, err, "%s", stderr)
		assert.Equal(t, "plain\n", stdout)
	})
}

func assertEmbeddedError(t *testing.T, stderr, code, message, suggestion string) {
	t.Helper()
	var envelope struct {
		Error struct {
			Code       string `json:"code"`
			Message    string `json:"message"`
			Suggestion string `json:"suggestion"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal([]byte(stderr), &envelope), "%s", stderr)
	assert.Equal(t, code, envelope.Error.Code)
	assert.Contains(t, envelope.Error.Message, message)
	assert.Contains(t, envelope.Error.Suggestion, suggestion)
}
