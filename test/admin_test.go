package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCmd_SettingsGet(t *testing.T) {
	out := runCmd(t, "settings", "get", "--format", "json")
	assert.NotEmpty(t, out)
}

func TestCmd_SettingsFrontend(t *testing.T) {
	out := runCmd(t, "settings", "frontend", "--format", "json")
	assert.NotEmpty(t, out)
}
