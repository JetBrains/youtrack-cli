package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmd_DashboardList(t *testing.T) {
	c := newTestClient(t)
	dashboard, err := c.AddDashboard("Test Dashboard " + t.Name())
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteDashboard(dashboard.ID) })

	out := runCmd(t, "dashboard", "list", "--format", "json")
	assert.Contains(t, out, dashboard.Name)
}

func TestDashboard_CRUD(t *testing.T) {
	c := newTestClient(t)

	// Create
	dashboard, err := c.AddDashboard("Test Dashboard " + t.Name())
	require.NoError(t, err)
	require.NotEmpty(t, dashboard.ID)
	t.Cleanup(func() { _ = c.DeleteDashboard(dashboard.ID) })

	assert.Contains(t, dashboard.Name, "Test Dashboard")

	// List
	dashboards, err := c.ListDashboards()
	require.NoError(t, err)
	found := false
	for _, d := range dashboards {
		if d.ID == dashboard.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "newly created dashboard should appear in list")

	// Update
	updated, err := c.UpdateDashboard(dashboard.ID, "Renamed Dashboard")
	require.NoError(t, err)
	assert.Equal(t, "Renamed Dashboard", updated.Name)
}
