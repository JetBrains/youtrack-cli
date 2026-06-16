package api_test

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
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
	nameCounter   uint64
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

// createTempProject creates a fresh project.
// The project is owned by the current user.
//
// Project deletion in YouTrack is asynchronous and can briefly poison the
// global /api/admin/projects list with "Project[...] was removed" errors.
// The Docker E2E harness resets state between CI runs, so fixture projects are
// intentionally left in place instead of being deleted during the test run.
func createTempProject(t *testing.T, c *api.Client) *api.Project {
	t.Helper()
	me, err := c.GetCurrentUser()
	require.NoError(t, err)

	shortName := uniqueProjectShortName("T")
	proj, err := c.AddProject(api.AddProjectRequest{
		ShortName: shortName,
		Name:      "Temp " + shortName,
		Leader:    &api.User{Login: me.Login},
	}, "")
	require.NoError(t, err)
	return proj
}

func uniqueProjectShortName(prefix string) string {
	const maxProjectShortNameLength = 10

	seq := atomic.AddUint64(&nameCounter, 1)
	suffix := strconv.FormatInt(time.Now().UnixMilli(), 36) + strconv.FormatUint(seq%36, 36)
	if len(prefix)+len(suffix) > maxProjectShortNameLength {
		prefix = prefix[:maxProjectShortNameLength-len(suffix)]
	}
	return strings.ToUpper(prefix + suffix)
}

// runCmd executes the compiled CLI binary with given args, returns stdout.
// Env vars YOUTRACK_BASE_URL and YOUTRACK_TOKEN must be set.
// HOME is overridden to an empty temp dir so the binary does not pick up the
// developer's ~/.youtrack-cli-config.json and instead falls back to the E2E env vars.
func runCmd(t *testing.T, args ...string) string {
	t.Helper()
	requireE2EEnv(t)

	cmd := exec.Command(binaryPath, args...)
	cmd.Env = append(os.Environ(), "HOME="+t.TempDir())
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	require.NoError(t, err, "binary stdout: %s\nbinary stderr: %s", stdout.String(), stderr.String())
	return stdout.String()
}

func runCmdExpectError(t *testing.T, args ...string) string {
	t.Helper()
	requireE2EEnv(t)

	cmd := exec.Command(binaryPath, args...)
	cmd.Env = append(os.Environ(), "HOME="+t.TempDir())
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	require.Error(t, err, "binary stdout: %s\nbinary stderr: %s", stdout.String(), stderr.String())
	return stdout.String()
}

// runCmdWithInput executes the compiled CLI binary with the given args and provides
// stdinInput as the command's stdin. Returns stdout as a string.
// HOME is overridden to an empty temp dir so the binary does not pick up the
// developer's ~/.youtrack-cli-config.json and instead falls back to the E2E env vars.
func runCmdWithInput(t *testing.T, stdinInput string, args ...string) string {
	t.Helper()
	requireE2EEnv(t)

	cmd := exec.Command(binaryPath, args...)
	cmd.Env = append(os.Environ(), "HOME="+t.TempDir())
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
// runCmdInDir runs the compiled binary in a specific working directory.
// Does NOT require YOUTRACK_BASE_URL or YOUTRACK_TOKEN — skill commands are filesystem-only.
func runCmdInDir(t *testing.T, dir string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = dir
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

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
