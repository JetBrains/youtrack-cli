package api_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

// ---------- Tags ----------

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

// TestCmd_SearchAdd verifies that the saved search is created and the response echoes the name.
// No cleanup is registered because the YouTrack API has no delete endpoint for saved searches.
func TestCmd_SearchAdd(t *testing.T) {
	name := fmt.Sprintf("test-%d", time.Now().UnixMilli()%100000)

	out := runCmd(t, "search", "add", "--name", name, "--query", "project: GOTEST", "--format", "json")
	assert.Contains(t, out, name)
}

// ---------- Organizations ----------

func TestCmd_OrgList(t *testing.T) {
	out := runCmd(t, "org", "list", "--format", "json")
	var items []json.RawMessage
	err := json.Unmarshal([]byte(strings.TrimSpace(out)), &items)
	require.NoError(t, err, "org list should return a JSON array")
}

func TestCmd_OrgAddDelete(t *testing.T) {
	c := newTestClient(t)
	name := fmt.Sprintf("cmd-org-%d", time.Now().UnixMilli()%100000)

	out := runCmd(t, "org", "add", "--name", name, "--format", "json")
	assert.Contains(t, out, name)

	var org struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &org))
	require.NotEmpty(t, org.ID)
	createdID := org.ID
	t.Cleanup(func() { _ = c.DeleteOrganization(createdID) })

	out = runCmd(t, "org", "delete", createdID)
	assert.Empty(t, strings.TrimSpace(out))

	orgs, err := c.ListOrganizations()
	require.NoError(t, err)
	for _, item := range orgs {
		assert.NotEqual(t, createdID, item.ID)
	}
}

// ---------- Apps ----------

func TestCmd_AppList(t *testing.T) {
	out := runCmd(t, "app", "list", "--format", "json")
	assert.NotEmpty(t, out)
}

// ---------- Credential precedence ----------

func TestCmd_EnvVarPrecedenceOverConfig(t *testing.T) {
	requireE2EEnv(t)

	// Write a config file that points at a non-existent server.
	home := t.TempDir()
	fakeCfg := `{"instances":{"fake":{"url":"http://127.0.0.1:1","token":"bad"}},"currentInstance":"fake"}`
	err := os.WriteFile(filepath.Join(home, ".youtrack-cli-config.json"), []byte(fakeCfg), 0600)
	require.NoError(t, err)

	// The process env already has real YOUTRACK_BASE_URL + YOUTRACK_TOKEN.
	// With new precedence (env > config), the command should use them and succeed.
	// On macOS/Linux the last HOME= entry wins; the fake config dir overrides the real one.
	out := runCmdWithEnv(t, "", []string{"HOME=" + home}, "user", "me", "--format", "json")
	assert.Contains(t, out, `"login"`)
}

// ---------- Whiteboard ----------
