package api_test

import (
	"encoding/json"
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

	// Note: YouTrack API has no delete endpoint for saved searches.
	// This test resource will persist on the instance after the test run.
}

// TestCmd_SearchAdd verifies that the saved search is created and the response echoes the name.
// No cleanup is registered because the YouTrack API has no delete endpoint for saved searches.
func TestCmd_SearchAdd(t *testing.T) {
	name := fmt.Sprintf("test-%d", time.Now().UnixMilli()%100000)

	out := runCmd(t, "search", "add", "--name", name, "--query", "project: GOTEST", "--format", "json")
	assert.Contains(t, out, name)
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
	var items []json.RawMessage
	err := json.Unmarshal([]byte(strings.TrimSpace(out)), &items)
	assert.NoError(t, err, "org list should return a JSON array")
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

	// Extract ID for cleanup — unmarshal JSON response
	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(out), &result); err == nil && result.ID != "" {
		t.Cleanup(func() { _ = c.DeleteCanvas(result.ID) })
	}
}
