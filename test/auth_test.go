package api_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
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

	tok, err := api.CreateHubToken(baseURL, login, password, me.ID, "test-cli-token", []string{services[0].ID})
	require.NoError(t, err)
	assert.NotEmpty(t, tok.Token)
	assert.NotEmpty(t, tok.ID)
	t.Cleanup(func() {
		_ = api.DeleteHubToken(baseURL, login, password, me.ID, tok.ID)
	})
}

func TestAuth_Status(t *testing.T) {
	requireE2EEnv(t)

	// Use --format json to get machine-readable output for assertions.
	out := runCmd(t, "--format", "json", "auth", "status")

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

	// No --format flag: default human-readable output.
	cmd := exec.Command(binaryPath, "--base-url", baseURL, "--token", "perm:invalid-token-for-test", "auth", "status")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	require.NoError(t, err, "binary stderr: %s", stderr.String())
	out := stdout.String()

	assert.Contains(t, out, "Token invalid")
	assert.Contains(t, out, baseURL)
}

func TestAuth_Status_TableFormat(t *testing.T) {
	requireE2EEnv(t)

	// auth status --format table falls back to JSON (table rendering not implemented for this command).
	out := runCmd(t, "--format", "table", "auth", "status")

	var result struct {
		Status string `json:"status"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "authenticated", result.Status)
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

func TestAuth_GetTokenSchemeNormalization(t *testing.T) {
	requireE2EEnv(t)

	token := os.Getenv("YOUTRACK_TOKEN")
	baseURL := os.Getenv("YOUTRACK_BASE_URL")
	// Strip the scheme to simulate a user typing the URL without https://.
	// The normalization always prepends https://, so this test only makes sense
	// against an HTTPS endpoint — skip for HTTP-only local servers.
	if !strings.HasPrefix(baseURL, "https://") {
		t.Skip("YOUTRACK_BASE_URL is not https:// — normalization test requires an HTTPS endpoint")
	}
	schemeless := strings.TrimPrefix(baseURL, "https://")

	// The CLI should prepend https:// automatically and still succeed.
	out := runCmdWithInput(t, "3\n"+token+"\n",
		"--base-url", schemeless,
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

	// No --format flag: default human-readable output.
	// Uses option 3 (direct paste) so no new token is created on the server.
	out := runCmdWithInput(t, "3\n"+token+"\n",
		"--base-url", baseURL,
		"auth", "get-token",
		"--save=false",
	)

	assert.Contains(t, out, "Token ready:")
	assert.Contains(t, out, "****")        // token is masked in human output
	assert.NotContains(t, out, token)      // full token must not appear
	assert.NotContains(t, out, `"token":`) // must not be JSON
}

func TestAuth_Logout_Force(t *testing.T) {
	tmpHome := t.TempDir()
	envFile := filepath.Join(tmpHome, ".youtrack-cli-env")
	require.NoError(t, os.WriteFile(envFile,
		[]byte("YOUTRACK_BASE_URL=https://test.example.com\nYOUTRACK_TOKEN=perm-test\n"), 0600))

	out := runCmdWithEnv(t, "", []string{"HOME=" + tmpHome},
		"auth", "logout", "--force")

	assert.Contains(t, out, "logged out from https://test.example.com")
	data, err := os.ReadFile(envFile)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "YOUTRACK_BASE_URL")
	assert.NotContains(t, string(data), "YOUTRACK_TOKEN")
}

func TestAuth_Logout_Confirm(t *testing.T) {
	tmpHome := t.TempDir()
	envFile := filepath.Join(tmpHome, ".youtrack-cli-env")
	require.NoError(t, os.WriteFile(envFile,
		[]byte("YOUTRACK_BASE_URL=https://test.example.com\nYOUTRACK_TOKEN=perm-test\n"), 0600))

	out := runCmdWithEnv(t, "y\n", []string{"HOME=" + tmpHome},
		"auth", "logout")

	assert.Contains(t, out, "logged out from https://test.example.com")
	data, err := os.ReadFile(envFile)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "YOUTRACK_BASE_URL")
	assert.NotContains(t, string(data), "YOUTRACK_TOKEN")
}

func TestAuth_Logout_Cancel(t *testing.T) {
	tmpHome := t.TempDir()
	envFile := filepath.Join(tmpHome, ".youtrack-cli-env")
	original := "YOUTRACK_BASE_URL=https://test.example.com\nYOUTRACK_TOKEN=perm-test\n"
	require.NoError(t, os.WriteFile(envFile, []byte(original), 0600))

	out := runCmdWithEnv(t, "n\n", []string{"HOME=" + tmpHome},
		"auth", "logout")

	assert.Contains(t, out, "Canceled.")
	data, err := os.ReadFile(envFile)
	require.NoError(t, err)
	assert.Equal(t, original, string(data))
}

func TestAuth_Logout_NotConfigured(t *testing.T) {
	tmpHome := t.TempDir()
	// No env file — directory exists but file is absent.

	out := runCmdWithEnv(t, "", []string{"HOME=" + tmpHome},
		"auth", "logout", "--force")

	assert.Contains(t, out, "not logged in")
}

func TestAuth_Logout_JSONFormat(t *testing.T) {
	tmpHome := t.TempDir()
	envFile := filepath.Join(tmpHome, ".youtrack-cli-env")
	require.NoError(t, os.WriteFile(envFile,
		[]byte("YOUTRACK_BASE_URL=https://test.example.com\nYOUTRACK_TOKEN=perm-test\n"), 0600))

	out := runCmdWithEnv(t, "", []string{"HOME=" + tmpHome},
		"--format", "json", "auth", "logout", "--force")

	var result struct {
		URL    string `json:"url"`
		Status string `json:"status"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &result), "output was: %s", out)
	assert.Equal(t, "logged out", result.Status)
	assert.Equal(t, "https://test.example.com", result.URL)
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
		"auth", "refresh-token", "myinst",
	)

	assert.Contains(t, out, "Token ready:")
	assert.Contains(t, out, "****")   // token is masked in human output
	assert.NotContains(t, out, token) // full token must not appear
}

func TestInstanceUpdateToken_ByAlias(t *testing.T) {
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
		"instance", "update-token", "myinst",
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

func TestInstanceUpdateToken_MissingArg(t *testing.T) {
	tmpHome := t.TempDir()
	configFixture(t, tmpHome, twoInstanceConfig)

	cmd := exec.Command(binaryPath, "instance", "update-token")
	cmd.Env = append(os.Environ(), "HOME="+tmpHome)
	out, err := cmd.CombinedOutput()

	assert.Error(t, err)
	assert.Contains(t, string(out), "alias is required")
}
