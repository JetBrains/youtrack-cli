package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// configFile returns the config path the binary uses when HOME is tmpHome and XDG_CONFIG_HOME is unset.
func configFile(tmpHome string) string {
	return filepath.Join(tmpHome, ".config", "youtrack", "config.json")
}

// configFixture writes a pre-built config file into tmpHome.
func configFixture(t *testing.T, tmpHome, content string) string {
	t.Helper()
	path := configFile(tmpHome)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0700))
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
	var items []struct {
		Alias  string `json:"alias"`
		URL    string `json:"url"`
		Active bool   `json:"active"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &items))
	require.Len(t, items, 2)
	assert.Equal(t, "personal", items[0].Alias)
	assert.Equal(t, "https://personal.example.com", items[0].URL)
	assert.False(t, items[0].Active)
	assert.Equal(t, "work", items[1].Alias)
	assert.Equal(t, "https://work.example.com", items[1].URL)
	assert.True(t, items[1].Active)
}

func TestInstance_List_MigratesLegacyConfig(t *testing.T) {
	tmpHome := t.TempDir()
	legacy := filepath.Join(tmpHome, ".youtrack-cli-config.json")
	require.NoError(t, os.WriteFile(legacy, []byte(twoInstanceConfig), 0600))

	out := runCmdWithEnv(t, "", []string{"HOME=" + tmpHome}, "instance", "list")
	var items []struct {
		Alias string `json:"alias"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &items))
	require.Len(t, items, 2)

	assert.NoFileExists(t, legacy)
	data, err := os.ReadFile(configFile(tmpHome))
	require.NoError(t, err)
	assert.JSONEq(t, twoInstanceConfig, string(data))
}

func TestInstance_List_XDGConfigHome(t *testing.T) {
	tmpHome := t.TempDir()
	xdg := t.TempDir()
	path := filepath.Join(xdg, "youtrack", "config.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0700))
	require.NoError(t, os.WriteFile(path, []byte(twoInstanceConfig), 0600))

	out := runCmdWithEnv(t, "", []string{"HOME=" + tmpHome, "XDG_CONFIG_HOME=" + xdg}, "instance", "list")
	var items []struct {
		Alias string `json:"alias"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &items))
	require.Len(t, items, 2)
	assert.NoDirExists(t, filepath.Join(tmpHome, ".config"))
}

func TestInstance_List_Empty(t *testing.T) {
	tmpHome := t.TempDir()

	out := runCmdWithEnv(t, "", []string{"HOME=" + tmpHome}, "instance", "list")
	var items []any
	require.NoError(t, json.Unmarshal([]byte(out), &items))
	assert.Empty(t, items)
}

func TestInstance_Use(t *testing.T) {
	tmpHome := t.TempDir()
	cfgPath := configFixture(t, tmpHome, twoInstanceConfig)

	out := runCmdWithEnv(t, "", []string{"HOME=" + tmpHome}, "instance", "use", "personal")
	var success struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &success))
	assert.Equal(t, "ok", success.Status)
	assert.Contains(t, success.Message, "personal")

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
	var success struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &success))
	assert.Equal(t, "ok", success.Status)
	assert.Contains(t, success.Message, "personal")
	assert.Contains(t, success.Message, "deleted")

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
	var success struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &success))
	assert.Equal(t, "ok", success.Status)
	assert.Contains(t, success.Message, "work")
	assert.Contains(t, success.Message, "deleted")
	assert.Contains(t, success.Message, "youtrack instance use")

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

	data, err := os.ReadFile(configFile(tmpHome))
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

// credentialSink is a local HTTP server standing in for an attacker-controlled host; it records every Authorization header it receives.
func credentialSink(t *testing.T) (*httptest.Server, *[]string) {
	t.Helper()
	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Header.Get("Authorization"))
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"1-1","login":"attacker"}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &seen
}

// runIsolated runs the binary with only the given env (no inherited YOUTRACK_* vars) and returns stderr and the exit error.
func runIsolated(t *testing.T, env []string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Env = append([]string{"PATH=" + os.Getenv("PATH"), "YOUTRACK_DISABLE_KEYRING=1"}, env...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stderr.String(), err
}

func TestGetClient_SavedTokenNotSentToOverriddenURL(t *testing.T) {
	cases := []struct {
		name string
		env  func(sinkURL string) []string
		args func(sinkURL string) []string
	}{
		{
			name: "--base-url flag with active instance token",
			args: func(sinkURL string) []string { return []string{"--base-url", sinkURL} },
		},
		{
			name: "YOUTRACK_BASE_URL env with active instance token",
			env:  func(sinkURL string) []string { return []string{"YOUTRACK_BASE_URL=" + sinkURL} },
		},
		{
			name: "--base-url flag with --server profile token",
			args: func(sinkURL string) []string { return []string{"--base-url", sinkURL, "--server", "personal"} },
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sink, seen := credentialSink(t)
			tmpHome := t.TempDir()
			configFixture(t, tmpHome, twoInstanceConfig)

			env := []string{"HOME=" + tmpHome}
			if c.env != nil {
				env = append(env, c.env(sink.URL)...)
			}
			var args []string
			if c.args != nil {
				args = c.args(sink.URL)
			}
			args = append(args, "--json", "user", "current")

			stderr, err := runIsolated(t, env, args...)
			require.Error(t, err, "expected the CLI to refuse; stderr: %s", stderr)
			assert.Contains(t, stderr, "refusing to send the saved token")
			assert.NotContains(t, stderr, "perm:aaa")
			assert.NotContains(t, stderr, "perm:bbb")
			assert.Empty(t, *seen, "saved token must not reach the overridden host")
		})
	}
}

func TestGetClient_OriginRefusalHasNoReauthTip(t *testing.T) {
	sink, _ := credentialSink(t)
	tmpHome := t.TempDir()
	configFixture(t, tmpHome, twoInstanceConfig)

	// Piped output defaults to JSON, whose errors carry no tip; force text mode, where a misleading re-auth tip would appear.
	stderr, err := runIsolated(t, []string{"HOME=" + tmpHome}, "--format", "table", "--base-url", sink.URL, "user", "current")
	require.Error(t, err)
	assert.Contains(t, stderr, "refusing to send the saved token")
	assert.NotContains(t, stderr, "Tip:", "the error already says how to proceed")
}

func TestGetClient_ExplicitTokenStillAllowedForOverriddenURL(t *testing.T) {
	sink, seen := credentialSink(t)
	tmpHome := t.TempDir()
	configFixture(t, tmpHome, twoInstanceConfig)

	// A caller-supplied token is the caller's own choice; only saved tokens are origin-bound.
	stderr, err := runIsolated(t, []string{"HOME=" + tmpHome},
		"--base-url", sink.URL, "--token", "perm:explicit", "--json", "user", "current")
	require.NoError(t, err, "stderr: %s", stderr)
	assert.Equal(t, []string{"Bearer perm:explicit"}, *seen)
}

func TestGetClient_SavedTokenUsedForSameOriginOverride(t *testing.T) {
	requireE2EEnv(t)
	baseURL := os.Getenv("YOUTRACK_BASE_URL")
	token := os.Getenv("YOUTRACK_TOKEN")
	tmpHome := t.TempDir()
	configFixture(t, tmpHome, `{
  "currentInstance": "myinst",
  "instances": {
    "myinst": {"url": "`+baseURL+`", "token": "`+token+`"}
  }
}`)

	// Same origin, different spelling (trailing slash) — the saved token is still usable.
	for _, env := range [][]string{
		{"HOME=" + tmpHome, "YOUTRACK_BASE_URL=" + baseURL + "/"},
		{"HOME=" + tmpHome},
	} {
		args := []string{"--format", "json", "auth", "status"}
		if len(env) == 1 {
			args = append([]string{"--base-url", baseURL + "/"}, args...)
		}
		cmd := exec.Command(binaryPath, args...)
		cmd.Env = append([]string{"YOUTRACK_DISABLE_KEYRING=1"}, env...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		require.NoError(t, cmd.Run(), "stderr: %s", stderr.String())

		var result struct {
			Status string `json:"status"`
		}
		require.NoError(t, json.Unmarshal(stdout.Bytes(), &result), "output was: %s", stdout.String())
		assert.Equal(t, "authenticated", result.Status)
	}
}

func TestGetClient_WarnsOnRemotePlainHTTP(t *testing.T) {
	// The warning is printed before the request, so an unresolvable host is enough.
	stderr, err := runIsolated(t, []string{"HOME=" + t.TempDir()},
		"--base-url", "http://youtrack.invalid", "--token", "perm:x", "user", "current")
	require.Error(t, err)
	assert.Contains(t, stderr, "uses plain http")

	sink, _ := credentialSink(t) // loopback http, as in the local harness: no warning
	stderr, err = runIsolated(t, []string{"HOME=" + t.TempDir()},
		"--base-url", sink.URL, "--token", "perm:x", "--json", "user", "current")
	require.NoError(t, err, "stderr: %s", stderr)
	assert.NotContains(t, stderr, "uses plain http")
}
