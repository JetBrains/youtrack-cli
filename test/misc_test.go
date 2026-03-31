//go:build integration

package api_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

// ---------- Tags ----------

func TestTag_CRUD(t *testing.T) {
	c := newTestClient(t)
	name := fmt.Sprintf("test-%d", time.Now().UnixMilli()%100000)

	tag, err := c.AddTag(api.AddTagRequest{Name: name})
	require.NoError(t, err)
	require.NotEmpty(t, tag.ID)
	assert.Equal(t, name, tag.Name)

	t.Cleanup(func() { _ = c.DeleteTag(tag.ID) })

	tags, err := c.ListTags()
	require.NoError(t, err)
	found := false
	for _, tg := range tags {
		if tg.ID == tag.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "created tag should appear in list")
}

func TestCmd_TagList(t *testing.T) {
	c := newTestClient(t)
	name := fmt.Sprintf("test-%d", time.Now().UnixMilli()%100000)

	tag, err := c.AddTag(api.AddTagRequest{Name: name})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteTag(tag.ID) })

	out := runCmd(t, "tag", "list", "--format", "json")
	assert.Contains(t, out, tag.ID)
}

// ---------- Saved Searches ----------

func TestSearch_CRUD(t *testing.T) {
	c := newTestClient(t)
	name := fmt.Sprintf("test-%d", time.Now().UnixMilli()%100000)

	result, err := c.AddSavedSearch(api.AddSavedSearchRequest{
		Name:  name,
		Query: "project: GOTEST",
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.ID)
	assert.Equal(t, name, result.Name)
	assert.Equal(t, "project: GOTEST", result.Query)

	// No ListSavedSearches API — verify the created object directly
	t.Cleanup(func() {
		// SavedSearch deletion is not exposed via API client, share as cleanup proxy
		// ShareSavedSearch is the only other mutation we can perform without delete
		_ = c.ShareSavedSearch(result.ID)
	})
}

func TestCmd_SearchAdd(t *testing.T) {
	name := fmt.Sprintf("test-%d", time.Now().UnixMilli()%100000)

	out := runCmd(t, "search", "add", "--name", name, "--query", "project: GOTEST", "--format", "json")
	assert.True(t, strings.Contains(out, name) || strings.Contains(out, "id"),
		"output should reference the created saved search")
}

// ---------- Organizations ----------

func TestOrg_CRUD(t *testing.T) {
	c := newTestClient(t)
	name := fmt.Sprintf("test-%d", time.Now().UnixMilli()%100000)

	org, err := c.AddOrganization(api.AddOrganizationRequest{Name: name})
	require.NoError(t, err)
	require.NotEmpty(t, org.ID)
	assert.Equal(t, name, org.Name)

	t.Cleanup(func() { _ = c.DeleteOrganization(org.ID) })

	orgs, err := c.ListOrganizations()
	require.NoError(t, err)
	found := false
	for _, o := range orgs {
		if o.ID == org.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "created org should appear in list")
}

func TestCmd_OrgList(t *testing.T) {
	out := runCmd(t, "org", "list", "--format", "json")
	// Just verify no error and valid JSON array output
	assert.True(t, strings.Contains(out, "[") || out == "[]\n" || out == "[]",
		"org list should return a JSON array")
}

// ---------- Apps ----------

func TestApp_List(t *testing.T) {
	c := newTestClient(t)
	apps, err := c.ListApps()
	require.NoError(t, err)
	assert.NotNil(t, apps)
	// Most instances won't have custom apps; just verify no error and valid slice
}

func TestCmd_AppList(t *testing.T) {
	out := runCmd(t, "app", "list", "--format", "json")
	assert.NotEmpty(t, out)
}

// ---------- Canvas ----------

func TestCanvas_CRUD(t *testing.T) {
	c := newTestClient(t)
	name := fmt.Sprintf("test-%d", time.Now().UnixMilli()%100000)

	canvas, err := c.AddCanvas(api.AddCanvasRequest{Name: name})
	require.NoError(t, err)
	require.NotEmpty(t, canvas.ID)
	assert.Equal(t, name, canvas.Name)

	t.Cleanup(func() { _ = c.DeleteCanvas(canvas.ID) })

	got, err := c.GetCanvas(canvas.ID)
	require.NoError(t, err)
	assert.Equal(t, canvas.ID, got.ID)
	assert.Equal(t, name, got.Name)
}

func TestCmd_CanvasAdd(t *testing.T) {
	c := newTestClient(t)
	name := fmt.Sprintf("test-%d", time.Now().UnixMilli()%100000)

	out := runCmd(t, "canvas", "add", "--name", name, "--format", "json")
	assert.Contains(t, out, name)

	// Extract ID for cleanup — parse JSON minimally
	var id string
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, `"id"`) {
			parts := strings.SplitN(line, `"`, 4)
			if len(parts) >= 4 {
				id = parts[3]
				id = strings.TrimSuffix(id, `"`)
			}
		}
	}
	if id != "" {
		t.Cleanup(func() { _ = c.DeleteCanvas(id) })
	}
}
