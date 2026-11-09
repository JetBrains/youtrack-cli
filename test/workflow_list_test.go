package api_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmd_WorkflowListReturnsEveryWorkflow(t *testing.T) {
	c := newTestClient(t)
	marker := fmt.Sprintf("wf-pages-%d", time.Now().UnixNano())
	// At least 501 matching workflows force a second page even on an empty instance.
	ids := make([]string, 0, 501)
	t.Cleanup(func() {
		for _, id := range ids {
			assert.NoError(t, c.DeleteWorkflow(id))
		}
	})
	for i := 0; i < 501; i++ {
		wf, err := c.AddWorkflow(fmt.Sprintf("%s-%03d", marker, i), "Pagination fixture")
		require.NoError(t, err)
		ids = append(ids, wf.ID)
	}
	listed := runJSON[[]api.Workflow](t, "workflow", "list", "--name", marker, "--json")
	got := make([]string, 0, len(listed))
	for _, wf := range listed {
		got = append(got, wf.ID)
	}
	assert.ElementsMatch(t, ids, got)

}

func TestCmd_WorkflowListFilterByName(t *testing.T) {
	c := newTestClient(t)
	marker := fmt.Sprintf("wf-filter-%d", time.Now().UnixNano()%1000000)
	first, err := c.AddWorkflow(marker+"-first", "First")
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteWorkflow(first.ID) })
	second, err := c.AddWorkflow(marker+"-second", "Second")
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteWorkflow(second.ID) })

	names := func(wfs []api.Workflow) []string {
		out := make([]string, 0, len(wfs))
		for _, wf := range wfs {
			out = append(out, wf.Name)
		}
		return out
	}

	both := runJSON[[]api.Workflow](t, "workflow", "list", "--name", strings.ToUpper(marker), "--json")
	assert.ElementsMatch(t, []string{marker + "-first", marker + "-second"}, names(both))

	one := runJSON[[]api.Workflow](t, "workflow", "list", "--name", marker+"-sec", "--json")
	assert.Equal(t, []string{marker + "-second"}, names(one))

	assert.JSONEq(t, "[]", runCmd(t, "workflow", "list", "--name", marker+"-none", "--json"))
}
