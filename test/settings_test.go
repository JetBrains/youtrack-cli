package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCmd_SettingsFrontendGet verifies that "settings frontend" returns a JSON object
// with the FrontendConfig type and a version field.
func TestCmd_SettingsFrontendGet(t *testing.T) {
	out := runCmd(t, "settings", "frontend", "--format", "json")
	assert.Contains(t, out, `"FrontendConfig"`)
	assert.Contains(t, out, `"version"`)
}
