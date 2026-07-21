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

func TestInstance_Version(t *testing.T) {
	out := runCmd(t, "instance", "version", "--format", "json")
	assert.Contains(t, out, `"version"`)
	assert.Contains(t, out, `"build"`)
}

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

	// Use raw exec.Command here — runCmdWithEnv calls require.NoError on exit,
	// which would panic instead of letting us assert the expected error.
	cmd := exec.Command(binaryPath, "instance", "use", "doesnotexist")
	cmd.Env = append(os.Environ(), "HOME="+tmpHome)
	out, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(out), "doesnotexist")
	assert.Contains(t, string(out), "not found")
}

func TestInstance_Delete_NonActive(t *testing.T) {
	tmpHome := t.TempDir()
	cfgPath := configFixture(t, tmpHome, twoInstanceConfig)

	out := runCmdWithEnv(t, "y\n", []string{"HOME=" + tmpHome}, "instance", "delete", "personal")

	assert.Contains(t, out, `"personal"`)
	assert.Contains(t, out, "deleted")

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

func TestInstance_Delete_Active(t *testing.T) {
	tmpHome := t.TempDir()
	cfgPath := configFixture(t, tmpHome, twoInstanceConfig)

	out := runCmdWithEnv(t, "y\n", []string{"HOME=" + tmpHome}, "instance", "delete", "work")

	assert.Contains(t, out, `"work"`)
	assert.Contains(t, out, "deleted")
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

func TestInstance_Delete_Force(t *testing.T) {
	tmpHome := t.TempDir()
	cfgPath := configFixture(t, tmpHome, twoInstanceConfig)

	out := runCmdWithEnv(t, "", []string{"HOME=" + tmpHome}, "instance", "delete", "--force", "personal")

	assert.Contains(t, out, "deleted")

	data, err := os.ReadFile(cfgPath)
	require.NoError(t, err)
	var result struct {
		Instances map[string]json.RawMessage `json:"instances"`
	}
	require.NoError(t, json.Unmarshal(data, &result))
	assert.NotContains(t, result.Instances, "personal")
}

// TestInstance_Remove_Cancel invokes the "remove" spelling on purpose — it doubles
// as coverage for the alias of "instance delete".
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
	cmd.Env = []string{"HOME=" + tmpHome, "YOUTRACK_DISABLE_KEYRING=1"}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	require.NoError(t, cmd.Run(), "stderr: %s", stderr.String())

	var results []struct {
		Status string `json:"status"`
		Active bool   `json:"active"`
	}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &results), "output was: %s", stdout.String())
	require.Len(t, results, 1)
	assert.Equal(t, "authenticated", results[0].Status)
	assert.True(t, results[0].Active)
}

func TestAuthGetToken_SavesToConfig(t *testing.T) {
	requireE2EEnv(t)
	token := os.Getenv("YOUTRACK_TOKEN")
	baseURL := os.Getenv("YOUTRACK_BASE_URL")
	tmpHome := t.TempDir()

	// Option 3 (direct paste), --alias mywork, default --save=true.
	runCmdWithEnv(t, "3\n"+token+"\n",
		[]string{"HOME=" + tmpHome},
		"--base-url", baseURL, "auth", "get-token",
		"--alias", "mywork",
	)

	data, err := os.ReadFile(filepath.Join(tmpHome, ".youtrack-cli-config.json"))
	require.NoError(t, err)
	var result struct {
		CurrentInstance string `json:"currentInstance"`
		Instances       map[string]struct {
			URL   string `json:"url"`
			Token string `json:"token"`
		} `json:"instances"`
	}
	require.NoError(t, json.Unmarshal(data, &result))
	assert.Equal(t, "mywork", result.CurrentInstance)
	require.Contains(t, result.Instances, "mywork")
	assert.Equal(t, baseURL, result.Instances["mywork"].URL)
	assert.Equal(t, token, result.Instances["mywork"].Token)
}

func TestAuthLogout_ConfigInstance(t *testing.T) {
	tmpHome := t.TempDir()
	cfgPath := configFixture(t, tmpHome, twoInstanceConfig)

	// Log out the active instance ("work") by default.
	// --plain forces human-readable output (piped stdout would auto-select JSON).
	out := runCmdWithEnv(t, "y\n", []string{"HOME=" + tmpHome}, "--plain", "auth", "logout")

	assert.Contains(t, out, "logged out from https://work.example.com")

	data, err := os.ReadFile(cfgPath)
	require.NoError(t, err)
	var result struct {
		CurrentInstance string                     `json:"currentInstance"`
		Instances       map[string]json.RawMessage `json:"instances"`
	}
	require.NoError(t, json.Unmarshal(data, &result))
	assert.NotContains(t, result.Instances, "work")
	assert.Equal(t, "", result.CurrentInstance)
}

func TestAuthLogout_ConfigInstance_WithFlag(t *testing.T) {
	tmpHome := t.TempDir()
	cfgPath := configFixture(t, tmpHome, twoInstanceConfig)

	// Log out "personal" (non-active) via --instance flag.
	out := runCmdWithEnv(t, "y\n", []string{"HOME=" + tmpHome},
		"--plain", "auth", "logout", "--instance", "personal")

	assert.Contains(t, out, "logged out from https://personal.example.com")

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

func TestAuthStatus_MultiInstance(t *testing.T) {
	requireE2EEnv(t)
	baseURL := os.Getenv("YOUTRACK_BASE_URL")
	token := os.Getenv("YOUTRACK_TOKEN")
	tmpHome := t.TempDir()

	cfg := `{
  "currentInstance": "myinst",
  "instances": {
    "myinst": {"url": "` + baseURL + `", "token": "` + token + `"},
    "broken": {"url": "https://invalid.example.com", "token": "perm:bad"}
  }
}`
	configFixture(t, tmpHome, cfg)

	// --plain forces human-readable output (piped stdout would auto-select JSON).
	cmd := exec.Command(binaryPath, "--plain", "auth", "status")
	cmd.Env = []string{"HOME=" + tmpHome, "YOUTRACK_DISABLE_KEYRING=1"}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	require.NoError(t, cmd.Run(), "stderr: %s", stderr.String())
	out := stdout.String()

	assert.Contains(t, out, "myinst")
	assert.Contains(t, out, "[active]")
	assert.Contains(t, out, "broken")
}
