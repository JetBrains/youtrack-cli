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

// configFixture writes a pre-built ~/.youtrack-cli-config.json into tmpHome.
func configFixture(t *testing.T, tmpHome, content string) string {
	t.Helper()
	path := filepath.Join(tmpHome, ".youtrack-cli-config.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}

// twoInstanceConfig is a fixture with "work" as active and "personal" as secondary.
const twoInstanceConfig = `{
  "currentInstance": "work",
  "instances": {
    "work":     {"url": "https://work.example.com",     "token": "perm:aaa"},
    "personal": {"url": "https://personal.example.com", "token": "perm:bbb"}
  }
}`

func TestInstance_List(t *testing.T) {
	tmpHome := t.TempDir()
	configFixture(t, tmpHome, twoInstanceConfig)

	out := runCmdWithEnv(t, "", []string{"HOME=" + tmpHome}, "instance", "list")

	assert.Contains(t, out, "work")
	assert.Contains(t, out, "https://work.example.com")
	assert.Contains(t, out, "[active]")
	assert.Contains(t, out, "personal")
	assert.Contains(t, out, "https://personal.example.com")
	assert.NotContains(t, out, "No instances configured")
}

func TestInstance_List_Empty(t *testing.T) {
	tmpHome := t.TempDir()

	out := runCmdWithEnv(t, "", []string{"HOME=" + tmpHome}, "instance", "list")

	assert.Contains(t, out, "No instances configured")
}

func TestInstance_Use(t *testing.T) {
	tmpHome := t.TempDir()
	cfgPath := configFixture(t, tmpHome, twoInstanceConfig)

	out := runCmdWithEnv(t, "", []string{"HOME=" + tmpHome}, "instance", "use", "personal")

	assert.Contains(t, out, `"personal"`)

	data, err := os.ReadFile(cfgPath)
	require.NoError(t, err)
	var result struct {
		CurrentInstance string `json:"currentInstance"`
	}
	require.NoError(t, json.Unmarshal(data, &result))
	assert.Equal(t, "personal", result.CurrentInstance)
}

func TestInstance_Use_UnknownAlias(t *testing.T) {
	tmpHome := t.TempDir()
	configFixture(t, tmpHome, twoInstanceConfig)

	cmd := exec.Command(binaryPath, "instance", "use", "doesnotexist")
	cmd.Env = append(os.Environ(), "HOME="+tmpHome)
	out, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(out), "doesnotexist")
	assert.Contains(t, string(out), "not found")
}

func TestInstance_Remove_NonActive(t *testing.T) {
	tmpHome := t.TempDir()
	cfgPath := configFixture(t, tmpHome, twoInstanceConfig)

	out := runCmdWithEnv(t, "y\n", []string{"HOME=" + tmpHome}, "instance", "remove", "personal")

	assert.Contains(t, out, `"personal"`)
	assert.Contains(t, out, "removed")

	data, err := os.ReadFile(cfgPath)
	require.NoError(t, err)
	var result struct {
		CurrentInstance string                     `json:"currentInstance"`
		Instances       map[string]json.RawMessage `json:"instances"`
	}
	require.NoError(t, json.Unmarshal(data, &result))
	assert.NotContains(t, result.Instances, "personal")
	assert.Equal(t, "work", result.CurrentInstance)
}

func TestInstance_Remove_Active(t *testing.T) {
	tmpHome := t.TempDir()
	cfgPath := configFixture(t, tmpHome, twoInstanceConfig)

	out := runCmdWithEnv(t, "y\n", []string{"HOME=" + tmpHome}, "instance", "remove", "work")

	assert.Contains(t, out, `"work"`)
	assert.Contains(t, out, "removed")
	assert.Contains(t, out, "youtrack instance use")

	data, err := os.ReadFile(cfgPath)
	require.NoError(t, err)
	var result struct {
		CurrentInstance string                     `json:"currentInstance"`
		Instances       map[string]json.RawMessage `json:"instances"`
	}
	require.NoError(t, json.Unmarshal(data, &result))
	assert.Equal(t, "", result.CurrentInstance)
	assert.NotContains(t, result.Instances, "work")
}

func TestInstance_Remove_Force(t *testing.T) {
	tmpHome := t.TempDir()
	cfgPath := configFixture(t, tmpHome, twoInstanceConfig)

	out := runCmdWithEnv(t, "", []string{"HOME=" + tmpHome}, "instance", "remove", "--force", "personal")

	assert.Contains(t, out, "removed")

	data, err := os.ReadFile(cfgPath)
	require.NoError(t, err)
	var result struct {
		Instances map[string]json.RawMessage `json:"instances"`
	}
	require.NoError(t, json.Unmarshal(data, &result))
	assert.NotContains(t, result.Instances, "personal")
}

func TestInstance_Remove_Cancel(t *testing.T) {
	tmpHome := t.TempDir()
	cfgPath := configFixture(t, tmpHome, twoInstanceConfig)
	original, err := os.ReadFile(cfgPath)
	require.NoError(t, err)

	out := runCmdWithEnv(t, "n\n", []string{"HOME=" + tmpHome}, "instance", "remove", "personal")

	assert.Contains(t, out, "Canceled")

	data, err := os.ReadFile(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, string(original), string(data))
}

func TestLegacyEnvFallback(t *testing.T) {
	requireE2EEnv(t)
	tmpHome := t.TempDir()
	out := runCmdWithEnv(t, "", []string{"HOME=" + tmpHome},
		"--format", "json", "auth", "status")

	var result struct {
		Status string `json:"status"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "authenticated", result.Status)
}

func TestGetClient_ActiveConfigInstance(t *testing.T) {
	requireE2EEnv(t)
	tmpHome := t.TempDir()
	baseURL := os.Getenv("YOUTRACK_BASE_URL")
	token := os.Getenv("YOUTRACK_TOKEN")
	cfg := `{
  "currentInstance": "myinst",
  "instances": {
    "myinst": {"url": "` + baseURL + `", "token": "` + token + `"}
  }
}`
	configFixture(t, tmpHome, cfg)

	cmd := exec.Command(binaryPath, "--format", "json", "auth", "status")
	cmd.Env = []string{"HOME=" + tmpHome}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	require.NoError(t, cmd.Run(), "stderr: %s", stderr.String())

	var result struct {
		Status string `json:"status"`
	}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "authenticated", result.Status)
}
