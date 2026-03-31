package api_test

import (
	"os"
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
