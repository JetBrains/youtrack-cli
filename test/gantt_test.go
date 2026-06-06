package api_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCmd_GanttCRUD runs list and delete against the same project so that the
// project is already visible to the planning system by the time it is reused.
func TestCmd_GanttCRUD(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	t.Run("list", func(t *testing.T) {
		chart, err := c.AddGanttChart("List Gantt "+t.Name(), []string{proj.ID}, time.Now().UnixMilli())
		require.NoError(t, err)
		t.Cleanup(func() { _ = c.DeleteGanttChart(chart.ID) })

		out := runCmd(t, "gantt", "list", "--format", "json")
		assert.Contains(t, out, chart.Name)
	})

	t.Run("delete", func(t *testing.T) {
		chart, err := c.AddGanttChart("Delete Gantt "+t.Name(), []string{proj.ID}, time.Now().UnixMilli())
		require.NoError(t, err)

		out := runCmd(t, "gantt", "delete", chart.ID)
		assert.Contains(t, out, "deleted")
	})
}
