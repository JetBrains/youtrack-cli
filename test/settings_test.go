package api_test

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCmd_SettingsSystemGet verifies that "settings system get" returns a JSON object
// that includes the instance base URL and the max upload file size.
func TestCmd_SettingsSystemGet(t *testing.T) {
	out := runCmd(t, "settings", "system", "get", "--format", "json")
	assert.Contains(t, out, `"baseUrl"`)
	assert.Contains(t, out, `"maxUploadFileSize"`)
}

// TestCmd_SettingsFrontendGet verifies that "settings frontend" returns a JSON object
// with the FrontendConfig type and a version field.
func TestCmd_SettingsFrontendGet(t *testing.T) {
	out := runCmd(t, "settings", "frontend", "--format", "json")
	assert.Contains(t, out, `"FrontendConfig"`)
	assert.Contains(t, out, `"version"`)
}

// TestCmd_SettingsBannerSetDisable sets a global banner and then disables it,
// verifying both "settings banner set" and "settings banner disable".
func TestCmd_SettingsBannerSetDisable(t *testing.T) {
	const bannerMsg = "e2e-test-banner"

	// Clean up: always disable the banner when the test finishes.
	t.Cleanup(func() {
		_ = runCmdBestEffort(t, "settings", "banner", "disable")
	})

	// Set the banner — the command exits 0 and produces no output on success.
	runCmd(t, "settings", "banner", "set", "--message", bannerMsg)

	// Disable the banner — again exits 0 with no output.
	runCmd(t, "settings", "banner", "disable")
}

// TestCmd_SettingsAIJetBrains verifies the "settings ai jetbrains" toggle round-trips
// without error and restores the enabled state afterwards.
func TestCmd_SettingsAIJetBrains(t *testing.T) {
	t.Cleanup(func() {
		// Best-effort restore: re-enable JetBrains AI.
		_ = runCmdBestEffort(t, "settings", "ai", "jetbrains", "--enabled", "true")
	})

	// Disable then re-enable — both should exit 0.
	runCmd(t, "settings", "ai", "jetbrains", "--enabled", "false")
	runCmd(t, "settings", "ai", "jetbrains", "--enabled", "true")
}

// TestCmd_SettingsSystemGetBaseURL checks that the system settings JSON contains the
// configured base URL of the test instance.
func TestCmd_SettingsSystemGetBaseURL(t *testing.T) {
	baseURL := os.Getenv("YOUTRACK_BASE_URL")
	require.NotEmpty(t, baseURL, "YOUTRACK_BASE_URL must be set")

	out := runCmd(t, "settings", "system", "get", "--format", "json")
	assert.True(t,
		strings.Contains(out, baseURL),
		"expected baseUrl %q in system settings output:\n%s", baseURL, out,
	)
}

// runCmdBestEffort executes the CLI binary but does not fail the test on error.
// Useful for cleanup steps where failure is acceptable.
func runCmdBestEffort(t *testing.T, args ...string) error {
	t.Helper()
	requireE2EEnv(t)

	cmd := exec.Command(binaryPath, args...)
	cmd.Env = append(os.Environ(), "HOME="+t.TempDir())
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		t.Logf("runCmdBestEffort %v: %v (stderr: %s)", args, err, stderr.String())
	}
	return err
}
