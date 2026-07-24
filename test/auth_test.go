package api_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

func TestAuth_GetTokenViaBasicAuth(t *testing.T) {
	login := os.Getenv("YOUTRACK_LOGIN")
	password := os.Getenv("YOUTRACK_PASSWORD")
	if login == "" || password == "" {
		t.Skip("YOUTRACK_LOGIN and YOUTRACK_PASSWORD not set — skipping basic auth test")
	}
	baseURL := os.Getenv("YOUTRACK_BASE_URL")
	if baseURL == "" {
		t.Skip("YOUTRACK_BASE_URL not set — skipping basic auth test")
	}

	me, err := api.GetHubMe(baseURL, login, password)
	require.NoError(t, err)
	assert.NotEmpty(t, me.ID)
	assert.NotEmpty(t, me.Login)

	services, err := api.ListHubServices(baseURL, login, password)
	require.NoError(t, err)
	require.NotEmpty(t, services)

	// Build scope: always include YouTrack + YouTrack Administration when available.
	var scopeIDs []string
	for _, svc := range services {
		if svc.Name == "YouTrack" || svc.Name == "YouTrack Administration" {
			scopeIDs = append(scopeIDs, svc.ID)
		}
	}
	if len(scopeIDs) == 0 && len(services) > 0 {
		scopeIDs = []string{services[0].ID}
	}

	tok, err := api.CreateHubToken(baseURL, login, password, me.ID, "test-cli-token", scopeIDs)
	require.NoError(t, err)
	assert.NotEmpty(t, tok.Token)
	assert.NotEmpty(t, tok.ID)
	t.Cleanup(func() {
		_ = api.DeleteHubToken(baseURL, login, password, me.ID, tok.ID)
	})
}

func TestAuth_Status(t *testing.T) {
	requireE2EEnv(t)

	// Use --json to get machine-readable output for assertions.
	out := runCmd(t, "--json", "auth", "status")

	var result struct {
		URL      string `json:"url"`
		FullName string `json:"fullName"`
		Login    string `json:"login"`
		Status   string `json:"status"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "authenticated", result.Status)
	assert.NotEmpty(t, result.FullName)
	assert.NotEmpty(t, result.Login)
	assert.Equal(t, os.Getenv("YOUTRACK_BASE_URL"), result.URL)
}

func TestAuth_HubDiscovery(t *testing.T) {
	baseURL := os.Getenv("YOUTRACK_BASE_URL")
	if baseURL == "" {
		t.Skip("YOUTRACK_BASE_URL not set — skipping hub discovery test")
	}

	resource, err := api.GetOAuthProtectedResource(baseURL)
	require.NoError(t, err)
	assert.NotEmpty(t, resource.AuthorizationServers, "expected at least one authorization server")

	hubURL, _, err := api.DetectHubURL(baseURL)
	require.NoError(t, err)
	// isInternal depends on the YouTrack instance topology (bundled vs. external Hub).
	// We only verify the function returns a non-empty URL without asserting the topology.
	assert.NotEmpty(t, hubURL)
}

func TestAuth_Status_InvalidToken(t *testing.T) {
	baseURL := os.Getenv("YOUTRACK_BASE_URL")
	if baseURL == "" {
		t.Skip("YOUTRACK_BASE_URL not set — skipping invalid token test")
	}

	// --plain forces human-readable output (piped stdout would auto-select JSON).
	cmd := exec.Command(binaryPath, "--plain", "--base-url", baseURL, "--token", "perm:invalid-token-for-test", "auth", "status")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	require.NoError(t, err, "binary stderr: %s", stderr.String())
	out := stdout.String()

	assert.Contains(t, out, "Token invalid")
	assert.Contains(t, out, baseURL)
}

func TestAuth_Status_PipedOutputIsJSON(t *testing.T) {
	requireE2EEnv(t)

	// No format flags: stdout is a pipe (non-TTY), so the output auto-selects JSON.
	out := runCmd(t, "auth", "status")

	var result struct {
		Status string `json:"status"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "authenticated", result.Status)
}

func TestAuth_Status_EnvironmentOverridesSavedInstances(t *testing.T) {
	requireE2EEnv(t)
	tmpHome := t.TempDir()
	configFixture(t, tmpHome, `{
  "currentInstance": "saved",
  "instances": {
    "saved": {"url": "https://invalid.example", "token": "perm:invalid"}
  }
}`)

	out := runCmdWithEnv(t, "", []string{
		"HOME=" + tmpHome,
		"YOUTRACK_BASE_URL=" + os.Getenv("YOUTRACK_BASE_URL"),
		"YOUTRACK_TOKEN=" + os.Getenv("YOUTRACK_TOKEN"),
	}, "--json", "auth", "status")
	var status struct {
		URL    string `json:"url"`
		Status string `json:"status"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &status))
	assert.Equal(t, "authenticated", status.Status)
	assert.Equal(t, os.Getenv("YOUTRACK_BASE_URL"), status.URL)
}

func TestAuth_GetTokenViaBasicAuthCLI(t *testing.T) {
	login := os.Getenv("YOUTRACK_LOGIN")
	password := os.Getenv("YOUTRACK_PASSWORD")
	baseURL := os.Getenv("YOUTRACK_BASE_URL")
	if login == "" || password == "" || baseURL == "" {
		t.Skip("YOUTRACK_LOGIN, YOUTRACK_PASSWORD, and YOUTRACK_BASE_URL not set — skipping CLI basic auth test")
	}

	// Pipe "1\n" to select basic auth; login and password come from flags.
	// Use --format json for machine-readable output.
	out := runCmdWithInput(t, "1\n",
		"--base-url", baseURL,
		"--format", "json",
		"auth", "get-token",
		"--login", login,
		"--password", password,
		"--token-name", "e2e-test-cli-token",
		"--save=false",
	)

	var result struct {
		ID    string `json:"id"`
		Token string `json:"token"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &result), "output was: %s", out)
	t.Cleanup(func() {
		me, err := api.GetHubMe(baseURL, login, password)
		if err == nil {
			_ = api.DeleteHubToken(baseURL, login, password, me.ID, result.ID)
		}
	})
	assert.NotEmpty(t, result.Token)
	assert.NotEmpty(t, result.ID)
}

func TestAuth_GetTokenDirectFlowCLI(t *testing.T) {
	requireE2EEnv(t)

	token := os.Getenv("YOUTRACK_TOKEN")
	baseURL := os.Getenv("YOUTRACK_BASE_URL")

	// Pipe "3\n<token>\n" to select direct token input (option 3).
	// Uses the existing YOUTRACK_TOKEN — no new token is created on the server.
	out := runCmdWithInput(t, "3\n"+token+"\n",
		"--base-url", baseURL,
		"--format", "json",
		"auth", "get-token",
		"--save=false",
	)

	var result struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &result), "output was: %s", out)
	assert.Equal(t, token, result.Token)
}

func TestAuth_GetTokenURLNormalization(t *testing.T) {
	requireE2EEnv(t)

	token := os.Getenv("YOUTRACK_TOKEN")
	baseURL := os.Getenv("YOUTRACK_BASE_URL")

	unmarshalToken := func(t *testing.T, out string) string {
		t.Helper()
		var result struct {
			Token string `json:"token"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &result), "output was: %s", out)
		return result.Token
	}

	// Trailing-slash stripping works for both HTTP and HTTPS.
	t.Run("trailing-slash", func(t *testing.T) {
		out := runCmdWithInput(t, "3\n"+token+"\n",
			"--base-url", strings.TrimRight(baseURL, "/")+"/",
			"--format", "json",
			"auth", "get-token",
			"--save=false",
		)
		assert.Equal(t, token, unmarshalToken(t, out))
	})

	// Schemeless normalization prepends https:// — only testable against an HTTPS server.
	t.Run("schemeless-https", func(t *testing.T) {
		if !strings.HasPrefix(baseURL, "https://") {
			t.Skip("schemeless normalization test requires an HTTPS endpoint")
		}
		schemeless := strings.TrimPrefix(baseURL, "https://")
		out := runCmdWithInput(t, "3\n"+token+"\n",
			"--base-url", schemeless,
			"--format", "json",
			"auth", "get-token",
			"--save=false",
		)
		assert.Equal(t, token, unmarshalToken(t, out))
	})
}

func TestAuth_GetTokenWebFlowCLI(t *testing.T) {
	requireE2EEnv(t)

	token := os.Getenv("YOUTRACK_TOKEN")
	baseURL := os.Getenv("YOUTRACK_BASE_URL")

	// Pipe "2\n<token>\n" to select web browser flow and paste the token.
	// openBrowser may open the browser as a side effect on macOS; that is harmless.
	// In headless environments it fails gracefully (prints the URL), then reads from stdin.
	out := runCmdWithInput(t, "2\n"+token+"\n",
		"--base-url", baseURL,
		"--format", "json",
		"auth", "get-token",
		"--save=false",
	)

	var result struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &result), "output was: %s", out)
	assert.Equal(t, token, result.Token)
}

func TestAuth_GetTokenHumanReadableOutput(t *testing.T) {
	requireE2EEnv(t)

	token := os.Getenv("YOUTRACK_TOKEN")
	baseURL := os.Getenv("YOUTRACK_BASE_URL")

	// --plain forces human-readable output (piped stdout would auto-select JSON).
	// Uses option 3 (direct paste) so no new token is created on the server.
	out := runCmdWithInput(t, "3\n"+token+"\n",
		"--plain",
		"--base-url", baseURL,
		"auth", "get-token",
		"--save=false",
	)

	assert.Contains(t, out, "Token ready:")
	assert.Contains(t, out, "****")        // token is masked in human output
	assert.NotContains(t, out, token)      // full token must not appear
	assert.NotContains(t, out, `"token":`) // must not be JSON
}

func TestAuth_Logout_NotConfigured(t *testing.T) {
	tmpHome := t.TempDir()

	out := runCmdWithEnv(t, "", []string{"HOME=" + tmpHome},
		"--plain", "auth", "logout", "--force")

	assert.Contains(t, out, "not logged in")
}

func TestAuthRefreshToken_UpdatesConfig(t *testing.T) {
	requireE2EEnv(t)
	token := os.Getenv("YOUTRACK_TOKEN")
	baseURL := os.Getenv("YOUTRACK_BASE_URL")
	tmpHome := t.TempDir()

	cfg := `{
  "currentInstance": "myinst",
  "instances": {
    "myinst": {"url": "` + baseURL + `", "token": "` + token + `"}
  }
}`
	cfgPath := configFixture(t, tmpHome, cfg)

	// Option 3 (paste), pipe the existing token so verification passes.
	runCmdWithEnv(t, "3\n"+token+"\n",
		[]string{"HOME=" + tmpHome},
		"auth", "refresh-token", "myinst",
	)

	data, err := os.ReadFile(cfgPath)
	require.NoError(t, err)
	var result struct {
		CurrentInstance string `json:"currentInstance"`
		Instances       map[string]struct {
			URL   string `json:"url"`
			Token string `json:"token"`
		} `json:"instances"`
	}
	require.NoError(t, json.Unmarshal(data, &result))
	assert.Equal(t, "myinst", result.CurrentInstance, "active instance must not change")
	require.Contains(t, result.Instances, "myinst")
	assert.Equal(t, baseURL, result.Instances["myinst"].URL, "URL must not change")
	assert.Equal(t, token, result.Instances["myinst"].Token)
}

func TestAuthRefreshToken_DefaultsToActive(t *testing.T) {
	requireE2EEnv(t)
	token := os.Getenv("YOUTRACK_TOKEN")
	baseURL := os.Getenv("YOUTRACK_BASE_URL")
	tmpHome := t.TempDir()

	cfg := `{
  "currentInstance": "myinst",
  "instances": {
    "myinst": {"url": "` + baseURL + `", "token": "` + token + `"}
  }
}`
	cfgPath := configFixture(t, tmpHome, cfg)

	// No alias arg — should default to currentInstance ("myinst").
	runCmdWithEnv(t, "3\n"+token+"\n",
		[]string{"HOME=" + tmpHome},
		"auth", "refresh-token",
	)

	data, err := os.ReadFile(cfgPath)
	require.NoError(t, err)
	var result struct {
		Instances map[string]struct {
			Token string `json:"token"`
		} `json:"instances"`
	}
	require.NoError(t, json.Unmarshal(data, &result))
	require.Contains(t, result.Instances, "myinst")
	assert.Equal(t, token, result.Instances["myinst"].Token)
}

func TestAuthRefreshToken_UnknownAlias(t *testing.T) {
	tmpHome := t.TempDir()
	configFixture(t, tmpHome, twoInstanceConfig)

	cmd := exec.Command(binaryPath, "auth", "refresh-token", "doesnotexist")
	cmd.Env = append(os.Environ(), "HOME="+tmpHome)
	out, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(out), "doesnotexist")
	assert.Contains(t, string(out), "not found")
}

func TestAuthRefreshToken_NoActiveInstance(t *testing.T) {
	tmpHome := t.TempDir()
	// Config exists but currentInstance is empty.
	configFixture(t, tmpHome, `{"currentInstance":"","instances":{}}`)

	cmd := exec.Command(binaryPath, "auth", "refresh-token")
	cmd.Env = append(os.Environ(), "HOME="+tmpHome)
	out, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(out), "no active instance")
}

func TestAuthRefreshToken_JSONOutput(t *testing.T) {
	requireE2EEnv(t)
	token := os.Getenv("YOUTRACK_TOKEN")
	baseURL := os.Getenv("YOUTRACK_BASE_URL")
	tmpHome := t.TempDir()

	cfg := `{
  "currentInstance": "myinst",
  "instances": {
    "myinst": {"url": "` + baseURL + `", "token": "` + token + `"}
  }
}`
	configFixture(t, tmpHome, cfg)

	out := runCmdWithEnv(t, "3\n"+token+"\n",
		[]string{"HOME=" + tmpHome},
		"--format", "json", "auth", "refresh-token", "myinst",
	)

	var result struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &result), "output was: %s", out)
	assert.Equal(t, token, result.Token)
}

func TestAuthRefreshToken_HumanReadableOutput(t *testing.T) {
	requireE2EEnv(t)
	token := os.Getenv("YOUTRACK_TOKEN")
	baseURL := os.Getenv("YOUTRACK_BASE_URL")
	tmpHome := t.TempDir()

	cfg := `{
  "currentInstance": "myinst",
  "instances": {
    "myinst": {"url": "` + baseURL + `", "token": "` + token + `"}
  }
}`
	configFixture(t, tmpHome, cfg)

	out := runCmdWithEnv(t, "3\n"+token+"\n",
		[]string{"HOME=" + tmpHome},
		"--plain", "auth", "refresh-token", "myinst",
	)

	assert.Contains(t, out, "Token ready:")
	assert.Contains(t, out, "****")   // token is masked in human output
	assert.NotContains(t, out, token) // full token must not appear
}

func TestInstanceRefreshToken_ByAlias(t *testing.T) {
	requireE2EEnv(t)
	token := os.Getenv("YOUTRACK_TOKEN")
	baseURL := os.Getenv("YOUTRACK_BASE_URL")
	tmpHome := t.TempDir()

	cfg := `{
  "currentInstance": "myinst",
  "instances": {
    "myinst": {"url": "` + baseURL + `", "token": "` + token + `"}
  }
}`
	cfgPath := configFixture(t, tmpHome, cfg)

	runCmdWithEnv(t, "3\n"+token+"\n",
		[]string{"HOME=" + tmpHome},
		"instance", "refresh-token", "myinst",
	)

	data, err := os.ReadFile(cfgPath)
	require.NoError(t, err)
	var result struct {
		Instances map[string]struct {
			URL   string `json:"url"`
			Token string `json:"token"`
		} `json:"instances"`
	}
	require.NoError(t, json.Unmarshal(data, &result))
	require.Contains(t, result.Instances, "myinst")
	assert.Equal(t, baseURL, result.Instances["myinst"].URL)
	assert.Equal(t, token, result.Instances["myinst"].Token)
}

func TestInstanceRefreshToken_MissingArg(t *testing.T) {
	tmpHome := t.TempDir()
	configFixture(t, tmpHome, twoInstanceConfig)

	cmd := exec.Command(binaryPath, "instance", "refresh-token")
	cmd.Env = append(os.Environ(), "HOME="+tmpHome)
	out, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(out), "alias is required")
}
