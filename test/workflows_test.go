package api_test

import (
	"fmt"
	"testing"
	"time"

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

func TestWorkflow_List(t *testing.T) {
	c := newTestClient(t)

	workflows, err := c.ListWorkflows()
	require.NoError(t, err)
	// Every YouTrack instance has at least one built-in workflow.
	assert.NotEmpty(t, workflows)
}

func TestWorkflow_AddDelete(t *testing.T) {
	c := newTestClient(t)

	// Workflow names must be lowercase alphanumeric + [-_.~] only.
	name := fmt.Sprintf("test-workflow-%d", time.Now().UnixMilli()%100000)
	title := "Test Workflow " + t.Name()

	wf, err := c.AddWorkflow(name, title)
	require.NoError(t, err)
	require.NotEmpty(t, wf.ID)
	t.Cleanup(func() { _ = c.DeleteWorkflow(wf.ID) })

	assert.Equal(t, name, wf.Name)

	// Verify it appears in list
	workflows, err := c.ListWorkflows()
	require.NoError(t, err)
	found := false
	for _, w := range workflows {
		if w.ID == wf.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "newly created workflow should appear in list")
}
