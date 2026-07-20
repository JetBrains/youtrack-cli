package api_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
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

		out := runCmd(t, "gantt", "delete", chart.ID, "--force")
		assert.Contains(t, out, "deleted")
	})
}

func TestCmd_GanttUpdate(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	chart, err := c.AddGanttChart("e2e-gantt-upd-"+t.Name(), []string{proj.ID}, time.Now().UnixMilli())
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteGanttChart(chart.ID) })

	newName := "e2e-gantt-renamed-" + t.Name()
	out := runCmd(t, "gantt", "update", chart.ID, "--name", newName, "--format", "json")
	assert.Contains(t, out, newName)
}

func TestCmd_GanttMemberList(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	chart, err := c.AddGanttChart("e2e-gantt-mbr-"+t.Name(), []string{proj.ID}, time.Now().UnixMilli())
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteGanttChart(chart.ID) })

	if _, err := c.ListGanttResources(chart.ID); err != nil {
		t.Skipf("gantt member list not supported on this instance: %v", err)
	}

	out := runCmd(t, "gantt", "member", "list", chart.ID, "--format", "json")
	assert.Contains(t, out, "[")
}

func TestCmd_GanttMemberAdd(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	chart, err := c.AddGanttChart("e2e-gantt-mbr-add-"+t.Name(), []string{proj.ID}, time.Now().UnixMilli())
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteGanttChart(chart.ID) })

	if _, err := c.ListGanttResources(chart.ID); err != nil {
		t.Skipf("gantt member add not supported on this instance: %v", err)
	}

	issue, err := c.AddIssue(api.AddIssueRequest{Summary: "e2e-gantt-member-issue", Project: &api.Project{ID: proj.ID}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.ID) })

	out := runCmd(t, "gantt", "member", "add", chart.ID, "--issue", issue.ID, "--format", "json")
	assert.Contains(t, out, issue.ID)
}

func TestCmd_GanttSettingsUpdate(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	chart, err := c.AddGanttChart("e2e-gantt-settings-"+t.Name(), []string{proj.ID}, time.Now().UnixMilli())
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteGanttChart(chart.ID) })

	if _, err := c.UpdateGanttSettings(chart.ID, map[string]any{"showWeekends": false}); err != nil {
		t.Skipf("gantt settings update not supported on this instance: %v", err)
	}

	out := runCmd(t, "gantt", "settings", "update", chart.ID, "--show-weekends", "--format", "json")
	assert.Contains(t, out, "showWeekends")
}
