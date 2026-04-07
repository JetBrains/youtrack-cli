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
