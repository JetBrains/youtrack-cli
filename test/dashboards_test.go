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
