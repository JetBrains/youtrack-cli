package api_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmd_WorkflowList(t *testing.T) {
	c := newTestClient(t)
	name := fmt.Sprintf("test-wf-cmd-%d", time.Now().UnixMilli()%100000)
	wf, err := c.AddWorkflow(name, "Test Workflow "+t.Name())
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteWorkflow(wf.ID) })

	out := runCmd(t, "workflow", "list", "--format", "json")
	assert.Contains(t, out, wf.Name)
}

func workflowRuleScript(title string) string {
	return "const entities = require('@jetbrains/youtrack-scripting-api/entities');\r\n" +
		"exports.rule = entities.Issue.onChange({\n" +
		"\ttitle: '" + title + "',\n" +
		"\tguard: () => false,\n" +
		"\taction: () => {},\n" +
		"\trequirements: {}\n" +
		"});\n"
}

func TestCmd_WorkflowRuleViewUpdate(t *testing.T) {
	c := newTestClient(t)
	wf, err := c.AddWorkflow(fmt.Sprintf("test-wf-rule-%d", time.Now().UnixNano()%1000000), "Rule script round trip")
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteWorkflow(wf.ID) })

	original := workflowRuleScript("Original")
	scriptPath := filepath.Join(t.TempDir(), "rule.js")
	require.NoError(t, os.WriteFile(scriptPath, []byte(original), 0o600))

	created := runJSON[api.WorkflowRule](t, "workflow", "rule", "create", wf.ID, "--name", "round-trip", "--script-file", scriptPath, "--json")
	require.NotEmpty(t, created.ID)
	assert.Equal(t, "round-trip", created.Name)
	assert.Equal(t, original, created.Script)

	viewed := runJSON[api.WorkflowRule](t, "workflow", "rule", "view", wf.ID, created.ID, "--json")
	assert.Equal(t, created.ID, viewed.ID)
	assert.Equal(t, "StatelessRule", viewed.Type)
	assert.Equal(t, original, viewed.Script, "view must return the script byte-for-byte")

	table := runCmd(t, "workflow", "rule", "view", wf.ID, created.ID, "--format", "table")
	assert.Contains(t, table, "round-trip")
	assert.Contains(t, table, "title: 'Original'")

	replacement := workflowRuleScript("Replaced")
	out := runCmdWithInput(t, replacement, "workflow", "rule", "update", wf.ID, created.ID, "--script-file", "-", "--json")
	assert.Contains(t, out, "verified")
	stored, err := c.GetWorkflowRule(wf.ID, created.ID)
	require.NoError(t, err)
	assert.Equal(t, replacement, stored.Script)

	t.Run("rejected script leaves the rule unchanged", func(t *testing.T) {
		broken := filepath.Join(t.TempDir(), "broken.js")
		require.NoError(t, os.WriteFile(broken, []byte("this is not javascript {"), 0o600))
		stderr := runCmdExpectError(t, "workflow", "rule", "update", wf.ID, created.ID, "--script-file", broken, "--json")
		assert.Contains(t, stderr, "SyntaxError")
		after, err := c.GetWorkflowRule(wf.ID, created.ID)
		require.NoError(t, err)
		assert.Equal(t, replacement, after.Script)
	})

	t.Run("empty script is rejected before the request", func(t *testing.T) {
		stderr := runCmdExpectError(t, "workflow", "rule", "update", wf.ID, created.ID, "--script-file", os.DevNull, "--json")
		assert.Contains(t, stderr, "script is empty")
	})

	t.Run("unknown rule fails", func(t *testing.T) {
		stderr := runCmdExpectError(t, "workflow", "rule", "update", wf.ID, "0-0", "--script-file", scriptPath, "--json")
		assert.True(t, strings.Contains(stderr, "not found") || strings.Contains(stderr, "404"), stderr)
		runCmdExpectError(t, "workflow", "rule", "view", wf.ID, "0-0")
	})
}
