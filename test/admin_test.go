package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

func TestCmd_SettingsView(t *testing.T) {
	settings := runJSON[api.GlobalSettings](t, "settings", "view", "--format", "json")
	require.NotNil(t, settings.SystemSettings)
	// baseUrl and maxUploadFileSize live on systemSettings, not on GlobalSettings itself.
	assert.NotEmpty(t, settings.SystemSettings.BaseURL)
	assert.Positive(t, settings.SystemSettings.MaxUploadFileSize)
}
