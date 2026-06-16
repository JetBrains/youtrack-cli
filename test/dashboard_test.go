package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmd_DashboardAdd(t *testing.T) {
	c := newTestClient(t)
	name := "e2e-dash-add-" + t.Name()

	out := runCmd(t, "dashboard", "add", "--name", name, "--format", "json")
	assert.Contains(t, out, name)

	dashboards, err := c.ListDashboards()
	require.NoError(t, err)
	for _, d := range dashboards {
		if d.Name == name {
			d := d
			t.Cleanup(func() { _ = c.DeleteDashboard(d.ID) })
		}
	}
}

func TestCmd_DashboardDelete(t *testing.T) {
	c := newTestClient(t)
	dash, err := c.AddDashboard("e2e-dash-del-" + t.Name())
	require.NoError(t, err)

	out := runCmd(t, "dashboard", "delete", dash.ID)
	assert.Contains(t, out, "deleted dashboard "+dash.ID)

	dashboards, err := c.ListDashboards()
	require.NoError(t, err)
	for _, d := range dashboards {
		assert.NotEqual(t, dash.ID, d.ID)
	}
}

func TestCmd_DashboardUpdate(t *testing.T) {
	c := newTestClient(t)
	dash, err := c.AddDashboard("e2e-dash-upd-" + t.Name())
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteDashboard(dash.ID) })

	newName := "e2e-dash-upd-renamed-" + t.Name()
	out := runCmd(t, "dashboard", "update", dash.ID, "--name", newName, "--format", "json")
	assert.Contains(t, out, newName)
}
