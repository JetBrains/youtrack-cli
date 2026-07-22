package api_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// updateStatus mirrors internal.UpdateStatus's JSON shape.
type updateStatus struct {
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion"`
	UpdateAvailable bool   `json:"updateAvailable"`
	UpgradeCommand  string `json:"upgradeCommand,omitempty"`
	ReleaseURL      string `json:"releaseUrl"`
}

// runUpdate runs "youtrack update" with the given extra args, returning
// (stdout, stderr, err) without asserting on the outcome. "update" needs no
// YouTrack instance, so it does not go through requireE2EEnv/runCmd.
func runUpdate(t *testing.T, env []string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := exec.Command(binaryPath, append([]string{"update"}, args...)...)
	cmd.Env = append(os.Environ(), env...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

// updateCheckEnvLimitMarkers are substrings that identify an update-check failure
// as an environment limitation (no internet, no/unauthenticated gh) rather than a
// real bug in the CLI, so the two don't get lumped into one blanket skip-on-error.
var updateCheckEnvLimitMarkers = []string{
	"gh CLI",                    // our own message when gh is missing
	"cli.github.com",            // our own message when gh is missing
	"gh api failed",             // gh present but failed (auth, rate limit, ...)
	"executable file not found", // gh missing from PATH (alternate OS phrasing)
	"GitHub API returned",       // non-200 from the public API (rate limit, etc.)
	"no such host", "connection refused", "i/o timeout",
	"network is unreachable", "dial tcp", "TLS handshake", "context deadline exceeded",
}

// skipIfUpdateCheckUnreachable skips the test when the update check failed for a
// reason outside the CLI's control (no internet, no/unauthenticated gh CLI) — the
// expected state in a network-restricted container — but still fails the test on
// any other error, so a real regression in the update command isn't silently
// swallowed as "environment limitation".
func skipIfUpdateCheckUnreachable(t *testing.T, stdout, stderr string, err error) {
	t.Helper()
	if err == nil {
		return
	}
	for _, marker := range updateCheckEnvLimitMarkers {
		if strings.Contains(stderr, marker) {
			t.Skipf("skipping: update check unreachable in this environment (stderr: %s)", stderr)
		}
	}
	t.Fatalf("update check failed unexpectedly (stdout: %s, stderr: %s): %v", stdout, stderr, err)
}

func TestCmd_Update_JSON(t *testing.T) {
	stdout, stderr, err := runUpdate(t, nil, "--json")
	skipIfUpdateCheckUnreachable(t, stdout, stderr, err)

	var status updateStatus
	require.NoError(t, json.Unmarshal([]byte(stdout), &status), "stdout: %s", stdout)

	require.Equal(t, binaryVersion, status.CurrentVersion)
	require.NotEmpty(t, status.LatestVersion)
	require.Contains(t, status.ReleaseURL, "github.com/JetBrains/youtrack-cli/releases/tag/")
	// binaryVersion is "dev" (or a low test version), so a real release always outranks it.
	require.True(t, status.UpdateAvailable)
	require.NotEmpty(t, status.UpgradeCommand)

	// --json must never attempt the auto-upgrade side effect.
	require.Empty(t, stderr)
}

// TestCmd_Update_NonTTY_ReportsWithoutAttempting covers a redirected/logged
// invocation of "update --format table" — piping through exec.Command always
// gives a non-TTY stdout, exactly the case that must never trigger a real
// auto-upgrade attempt, even though --format table resolves the same as an
// interactive run. It should still report the available update and the manual
// upgrade command, just without attempting to run anything or suggesting the
// gh-based alternative (that's only surfaced as part of the interactive
// fallback flow).
func TestCmd_Update_NonTTY_ReportsWithoutAttempting(t *testing.T) {
	stdout, stderr, err := runUpdate(t, nil, "--format", "table")
	skipIfUpdateCheckUnreachable(t, stdout, stderr, err)

	require.Contains(t, stderr, "Checking for updates...")
	require.Contains(t, stdout, "YouTrack CLI")
	require.Contains(t, stdout, "is available (you have dev)")
	require.Contains(t, stdout, "Upgrade:")
	require.Contains(t, stdout, "https://github.com/JetBrains/youtrack-cli/releases/latest")
	require.NotContains(t, stdout, "Alternative (via gh):")
}

func TestCmd_Update_AlreadyUpToDate(t *testing.T) {
	versionedBinary := buildVersionedBinary(t, "999.0.0")

	cmd := exec.Command(versionedBinary, "update", "--format", "table")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	skipIfUpdateCheckUnreachable(t, stdout.String(), stderr.String(), err)

	require.Contains(t, stdout.String(), "YouTrack CLI v999.0.0 — up to date")
	require.NotContains(t, stdout.String(), "Upgrade:")
}

// TestCmd_Update_NoGH_ClearError checks that when the public API can't resolve a
// release (as it can't while this repo is internal) and gh isn't on PATH, the
// command fails with an error naming the gh CLI as the way to check while the
// repo is private — rather than a vague "missing tag_name"-style message.
// If the public API succeeds (e.g. once the repo is public), that's a valid
// outcome too, and this test verifies the JSON output stays well-formed.
func TestCmd_Update_NoGH_ClearError(t *testing.T) {
	pathWithoutGH := trimFromPath(t, "gh")

	stdout, stderr, err := runUpdate(t, []string{"PATH=" + pathWithoutGH}, "--json")
	if err == nil {
		var status updateStatus
		require.NoError(t, json.Unmarshal([]byte(stdout), &status), "stdout: %s", stdout)
		return
	}
	require.Contains(t, stderr, "gh CLI")
	require.Contains(t, stderr, "cli.github.com")
}

// trimFromPath returns the current PATH with every directory containing name removed.
func trimFromPath(t *testing.T, name string) string {
	t.Helper()
	dirs := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	kept := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			continue
		}
		kept = append(kept, dir)
	}
	return strings.Join(kept, string(os.PathListSeparator))
}

// buildVersionedBinary compiles a second copy of the CLI with a specific
// version baked in, for tests that need to control the "current version"
// side of a comparison (e.g. asserting the "already up to date" case).
func buildVersionedBinary(t *testing.T, version string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "youtrack")
	build := exec.Command("make", "build",
		"BINARY="+path, "VERSION="+version, "COMMIT="+binaryCommit, "DATE="+binaryDate)
	build.Dir = ".."
	var out bytes.Buffer
	build.Stdout = &out
	build.Stderr = &out
	require.NoError(t, build.Run(), "build failed: %s", out.String())
	return path
}
