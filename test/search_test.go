package api_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

func newTempSearch(t *testing.T, c *api.Client, prefix string) *api.SavedSearch {
	t.Helper()
	name := fmt.Sprintf("%s-%d", prefix, time.Now().UnixMilli()%100000)
	search, err := c.AddSavedSearch(api.AddSavedSearchRequest{
		Name:  name,
		Query: "project: {}",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteSavedSearch(search.ID) })
	return search
}

func TestCmd_SearchList(t *testing.T) {
	c := newTestClient(t)
	search := newTempSearch(t, c, "e2e-list")

	out := runCmd(t, "search", "list", "--format", "json")
	assert.Contains(t, out, search.Name)
}

func TestCmd_SearchView(t *testing.T) {
	c := newTestClient(t)
	search := newTempSearch(t, c, "e2e-get")

	out := runCmd(t, "search", "view", search.ID, "--format", "json")
	assert.Contains(t, out, search.ID)
}

func TestCmd_SearchDelete(t *testing.T) {
	c := newTestClient(t)
	name := fmt.Sprintf("e2e-del-%d", time.Now().UnixMilli()%100000)
	search, err := c.AddSavedSearch(api.AddSavedSearchRequest{
		Name:  name,
		Query: "project: {}",
	})
	require.NoError(t, err)

	out := runCmd(t, "search", "delete", search.ID, "--force")
	assert.Contains(t, out, "deleted saved search "+search.ID)

	_, err = c.GetSavedSearch(search.ID)
	require.Error(t, err)
}

func TestCmd_SearchShare(t *testing.T) {
	c := newTestClient(t)
	search := newTempSearch(t, c, "e2e-share")

	// share returns no output on success; just verify it exits 0
	runCmd(t, "search", "share", search.ID)

	// verify sharing actually propagated: All Users group must appear in permittedGroups
	updated, err := c.GetSavedSearch(search.ID)
	require.NoError(t, err)
	require.NotNil(t, updated.ReadSharingSettings, "readSharingSettings should be set after share")
	assert.NotEmpty(t, updated.ReadSharingSettings.PermittedGroups, "All Users group should appear in permittedGroups after share")
}
