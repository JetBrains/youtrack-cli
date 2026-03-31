package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdmin_GetGlobalSettings(t *testing.T) {
	c := newTestClient(t)

	settings, err := c.GetGlobalSettings()
	require.NoError(t, err)
	// settings.BaseURL may be empty on some instances; just verify no error.
	assert.NotNil(t, settings)
}

func TestAdmin_ListVCSServers(t *testing.T) {
	c := newTestClient(t)

	_, err := c.ListVCSServers()
	require.NoError(t, err)
}

func TestAdmin_ListVCSProcessors(t *testing.T) {
	c := newTestClient(t)

	_, err := c.ListVCSProcessors()
	require.NoError(t, err)
}

func TestAdmin_ListMailboxServers(t *testing.T) {
	c := newTestClient(t)

	_, err := c.ListMailboxServers()
	require.NoError(t, err)
}

func TestAdmin_GetFrontendConfig(t *testing.T) {
	c := newTestClient(t)

	cfg, err := c.GetFrontendConfig()
	require.NoError(t, err)
	assert.NotNil(t, cfg)
}

func TestAdmin_ListPlugins(t *testing.T) {
	c := newTestClient(t)

	_, err := c.ListPlugins()
	require.NoError(t, err)
}
