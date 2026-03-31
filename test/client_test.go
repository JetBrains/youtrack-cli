//go:build integration

package api_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

func TestClient_GetReturnsData(t *testing.T) {
	c := newTestClient(t)
	projects, err := c.ListProjects()
	require.NoError(t, err)
	assert.NotNil(t, projects)
}

func TestClient_AuthFailure(t *testing.T) {
	c := api.NewClient(os.Getenv("YOUTRACK_BASE_URL"), "invalid-token-xyz")
	_, err := c.ListProjects()
	require.Error(t, err)
}
