package api_test

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

// binaryPath is the path to the compiled CLI binary, built once by TestMain.
var (
	binaryPath    string
	binaryVersion string
	binaryCommit  string
	binaryDate    string
)

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "youtrack-cli-test-*")
	if err != nil {
		log.Fatalf("failed to create temp dir: %v", err)
	}

	binaryPath = filepath.Join(tmpDir, "youtrack")
	binaryVersion = envOrDefault("VERSION", "dev")
	binaryCommit = gitShortCommit("..")
	binaryDate = time.Now().UTC().Format("2006-01-02")

	// Build the binary through the Makefile so tests exercise the same recipe as production builds.
	build := exec.Command(
		"make",
		"build",
		"BINARY="+binaryPath,
		"VERSION="+binaryVersion,
		"COMMIT="+binaryCommit,
		"DATE="+binaryDate,
	)
	build.Dir = ".." // repo root (tests run from test/)
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		os.RemoveAll(tmpDir)
		log.Fatalf("failed to build binary: %v", err)
	}

	exitCode := m.Run()
	os.RemoveAll(tmpDir)
	os.Exit(exitCode)
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func gitShortCommit(repoRoot string) string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = repoRoot

	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	commit := strings.TrimSpace(string(out))
	if commit == "" {
		return "unknown"
	}
	return commit
}

func requireE2EEnv(t *testing.T) {
	t.Helper()

	if os.Getenv("YOUTRACK_BASE_URL") == "" || os.Getenv("YOUTRACK_TOKEN") == "" {
		t.Skip("skipping YouTrack E2E test: set YOUTRACK_BASE_URL and YOUTRACK_TOKEN")
	}
}

// newTestClient creates an API client from env vars.
func newTestClient(t *testing.T) *api.Client {
	t.Helper()
	requireE2EEnv(t)

	return api.NewClient(
		os.Getenv("YOUTRACK_BASE_URL"),
		os.Getenv("YOUTRACK_TOKEN"),
	)
}

// createTempProject creates a fresh project and registers t.Cleanup to delete it.
// The project is owned by the current user.
func createTempProject(t *testing.T, c *api.Client) *api.Project {
	t.Helper()
	me, err := c.GetCurrentUser()
	require.NoError(t, err)

	// short name: max 10 chars, letters+digits, unique per millisecond
	shortName := fmt.Sprintf("T%07d", time.Now().UnixMilli()%10000000)
	proj, err := c.AddProject(api.AddProjectRequest{
		ShortName: shortName,
		Name:      "Temp " + shortName,
		Leader:    &api.User{Login: me.Login},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteProject(proj.ShortName) })
	return proj
}

// runCmd executes the compiled CLI binary with given args, returns stdout.
// Env vars YOUTRACK_BASE_URL and YOUTRACK_TOKEN must be set.
func runCmd(t *testing.T, args ...string) string {
	t.Helper()
	requireE2EEnv(t)

	cmd := exec.Command(binaryPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	require.NoError(t, err, "binary stdout: %s\nbinary stderr: %s", stdout.String(), stderr.String())
	return stdout.String()
}

// runCmdWithInput executes the compiled CLI binary with the given args and provides
// stdinInput as the command's stdin. Returns stdout as a string.
func runCmdWithInput(t *testing.T, stdinInput string, args ...string) string {
	t.Helper()
	requireE2EEnv(t)

	cmd := exec.Command(binaryPath, args...)
	// The subprocess inherits the full test-process environment, including YOUTRACK_TOKEN.
	// Callers that need a specific token should pass it via --token flag.
	cmd.Stdin = strings.NewReader(stdinInput)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	require.NoError(t, err, "binary stdout: %s\nbinary stderr: %s", stdout.String(), stderr.String())
	return stdout.String()
}

// runCmdWithEnv executes the compiled CLI binary with the given args, piped stdin,
// and extra environment variables appended to the current process env.
// Unlike runCmd/runCmdWithInput, it does NOT call requireE2EEnv — callers manage
// their own skip conditions.
func runCmdWithEnv(t *testing.T, stdinInput string, extraEnv []string, args ...string) string {
	t.Helper()

	cmd := exec.Command(binaryPath, args...)
	cmd.Stdin = strings.NewReader(stdinInput)
	// The subprocess inherits the full test-process environment; extraEnv overrides specific keys.
	cmd.Env = append(os.Environ(), extraEnv...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	require.NoError(t, err, "binary stdout: %s\nbinary stderr: %s", stdout.String(), stderr.String())
	return stdout.String()
}
