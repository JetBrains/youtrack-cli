package api_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkill_List(t *testing.T) {
	dir := t.TempDir()
	stdout, _, err := runCmdInDir(t, dir, "skill", "list")
	require.NoError(t, err)
	assert.Contains(t, stdout, "youtrack-cli")
	assert.Contains(t, stdout, "2.0.0")
}

func TestSkill_InstallUpdateRemove(t *testing.T) {
	dir := t.TempDir()

	// Install into project
	stdout, _, err := runCmdInDir(t, dir, "skill", "install", "--agent", "claude-code", "--project")
	require.NoError(t, err)
	assert.Contains(t, stdout, "installed youtrack-cli")
	assert.Contains(t, stdout, "claude-code")

	// Verify SKILL.md written to claude-code project-level dir
	skillMD := filepath.Join(dir, ".claude", "skills", "youtrack-cli", "SKILL.md")
	_, err = os.Stat(skillMD)
	require.NoError(t, err, "SKILL.md should exist after install")

	// Update: already at latest version, should report up to date
	stdout, _, err = runCmdInDir(t, dir, "skill", "update", "--agent", "claude-code", "--project")
	require.NoError(t, err)
	assert.Contains(t, stdout, "up to date")

	// Remove
	stdout, _, err = runCmdInDir(t, dir, "skill", "remove", "--agent", "claude-code", "--project")
	require.NoError(t, err)
	assert.Contains(t, stdout, "removed youtrack-cli")

	// SKILL.md is gone
	_, err = os.Stat(skillMD)
	assert.True(t, os.IsNotExist(err), "SKILL.md should no longer exist after remove")

	// Remove again: skill gone, warning goes to stderr, exit 0
	stdout, stderr, err := runCmdInDir(t, dir, "skill", "remove", "--agent", "claude-code", "--project")
	require.NoError(t, err)
	assert.Empty(t, stdout)
	assert.Contains(t, stderr, "not installed")
}
