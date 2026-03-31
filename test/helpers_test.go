package api_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/JetBrains/youtrack-cli/cmd"
)

// TestMain guards against running without credentials.
func TestMain(m *testing.M) {
	if os.Getenv("YOUTRACK_BASE_URL") == "" || os.Getenv("YOUTRACK_TOKEN") == "" {
		fmt.Fprintln(os.Stderr, "YOUTRACK_BASE_URL and YOUTRACK_TOKEN must be set to run tests")
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// newTestClient creates an API client from env vars.
func newTestClient(t *testing.T) *api.Client {
	t.Helper()
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

// runCmd executes the CLI root command with given args, returns stdout.
// Env vars YOUTRACK_BASE_URL and YOUTRACK_TOKEN must be set.
func runCmd(t *testing.T, args ...string) string {
	t.Helper()
	root := cmd.NewRootCmd()
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs(args)
	err := root.Execute()
	require.NoError(t, err, "cmd stderr: %s", stderr.String())
	return stdout.String()
}
