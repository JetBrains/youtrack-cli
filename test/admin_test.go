package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCmd_SettingsView(t *testing.T) {
	out := runCmd(t, "settings", "view", "--format", "json")
	assert.NotEmpty(t, out)
}

func TestCmd_SettingsFrontend(t *testing.T) {
	out := runCmd(t, "settings", "frontend", "--format", "json")
	assert.NotEmpty(t, out)
}
