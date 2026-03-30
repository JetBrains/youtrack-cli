//go:build integration

package api_test

import (
	"os"
	"testing"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// integProjectKey must exist in the YouTrack instance before running these tests.
const integProjectKey = "GOTEST"

var (
	integClient    *api.Client
	createdIssueID string
)

func TestMain(m *testing.M) {
	baseURL := os.Getenv("YOUTRACK_BASE_URL")
	token := os.Getenv("YOUTRACK_TOKEN")
	if baseURL == "" || token == "" {
		os.Exit(0) // skip silently when env not configured
	}
	integClient = api.NewClient(baseURL, token)
	code := m.Run()
	integCleanup()
	os.Exit(code)
}

func integCleanup() {
	if createdIssueID != "" {
		_ = integClient.DeleteIssue(createdIssueID)
	}
}

func TestIntegration_GetCurrentUser(t *testing.T) {
	user, err := integClient.GetCurrentUser()
	require.NoError(t, err)
	assert.NotEmpty(t, user.Login)
}

func TestIntegration_ListProjects(t *testing.T) {
	projects, err := integClient.ListProjects()
	require.NoError(t, err)
	assert.NotEmpty(t, projects)
}

func TestIntegration_AddGetDeleteIssue(t *testing.T) {
	proj, err := integClient.GetProject(integProjectKey)
	require.NoError(t, err, "project %s must exist in YouTrack", integProjectKey)

	issue, err := integClient.AddIssue(api.AddIssueRequest{
		Summary: "Integration test — safe to delete",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, issue.IDReadable)
	createdIssueID = issue.IDReadable

	fetched, err := integClient.GetIssue(createdIssueID)
	require.NoError(t, err)
	assert.Equal(t, "Integration test — safe to delete", fetched.Summary)
}
