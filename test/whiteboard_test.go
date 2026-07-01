package api_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

// createTestWhiteboard creates a whiteboard owned by the current user and registers cleanup.
func createTestWhiteboard(t *testing.T, c *api.Client) string {
	t.Helper()
	me, err := c.GetCurrentUser()
	require.NoError(t, err)
	name := fmt.Sprintf("test-whiteboard-%d", time.Now().UnixMilli()%100000)

	out := runCmd(t, "whiteboard", "add", "--name", name, "--owner", me.Login, "--format", "json")
	var wb struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &wb))
	require.NotEmpty(t, wb.ID)
	t.Cleanup(func() { _ = c.DeleteCanvas(wb.ID) })
	return wb.ID
}

// addTestWhiteboardNode adds a card node and returns its ID.
func addTestWhiteboardNode(t *testing.T, whiteboardID, label string, x, y int) string {
	t.Helper()
	out := runCmd(t, "whiteboard", "node", "add", whiteboardID,
		"--x", fmt.Sprintf("%d", x),
		"--y", fmt.Sprintf("%d", y),
		"--label", label,
		"--format", "json",
	)
	var node struct {
		ID    string `json:"id"`
		Label string `json:"label"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &node))
	require.NotEmpty(t, node.ID, "node %s has no ID", label)
	assert.Equal(t, label, node.Label)
	return node.ID
}

func TestCmd_WhiteboardDeletePrintsSuccess(t *testing.T) {
	c := newTestClient(t)
	me, err := c.GetCurrentUser()
	require.NoError(t, err)

	wb, err := c.AddCanvas(api.AddCanvasRequest{Name: "delete-test", Owner: &api.User{ID: me.ID}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteCanvas(wb.ID) })

	out := runCmd(t, "whiteboard", "delete", wb.ID)
	assert.Contains(t, out, wb.ID)
}

func TestCmd_WhiteboardAdd(t *testing.T) {
	c := newTestClient(t)
	wbID := createTestWhiteboard(t, c)
	assert.NotEmpty(t, wbID)
}

func TestCmd_WhiteboardNodeAdd(t *testing.T) {
	c := newTestClient(t)
	wbID := createTestWhiteboard(t, c)
	nodeID := addTestWhiteboardNode(t, wbID, "Test Node", 50, 50)
	assert.NotEmpty(t, nodeID)
}

func TestCmd_WhiteboardList(t *testing.T) {
	c := newTestClient(t)
	wbID := createTestWhiteboard(t, c)
	_ = wbID

	out := runCmd(t, "whiteboard", "list", "--format", "json")
	assert.Contains(t, out, "[")
}

func TestCmd_WhiteboardGet(t *testing.T) {
	c := newTestClient(t)
	wbID := createTestWhiteboard(t, c)

	out := runCmd(t, "whiteboard", "get", wbID, "--format", "json")
	assert.Contains(t, out, wbID)
}

func TestCmd_WhiteboardNodeList(t *testing.T) {
	c := newTestClient(t)
	wbID := createTestWhiteboard(t, c)
	addTestWhiteboardNode(t, wbID, "node-list-node", 10, 10)

	out := runCmd(t, "whiteboard", "node", "list", wbID, "--format", "json")
	assert.Contains(t, out, "node-list-node")
}

func TestCmd_WhiteboardNodeGet(t *testing.T) {
	c := newTestClient(t)
	wbID := createTestWhiteboard(t, c)
	nodeID := addTestWhiteboardNode(t, wbID, "node-get-node", 20, 20)

	out := runCmd(t, "whiteboard", "node", "get", wbID, nodeID, "--format", "json")
	assert.Contains(t, out, nodeID)
}

func TestCmd_WhiteboardReload(t *testing.T) {
	c := newTestClient(t)
	wbID := createTestWhiteboard(t, c)

	if err := c.ReloadCanvas(wbID); err != nil {
		t.Skipf("whiteboard reload not supported on this instance: %v", err)
	}
	runCmd(t, "whiteboard", "reload", wbID)
}

func TestCmd_WhiteboardLinkTypes(t *testing.T) {
	c := newTestClient(t)
	wbID := createTestWhiteboard(t, c)

	// Resolve a link prototype ID for the PROTOTYPED case.
	linkTypes, err := c.GetIssueLinkTypes()
	require.NoError(t, err)
	require.NotEmpty(t, linkTypes, "need at least one issue link type for PROTOTYPED test")
	protoID := linkTypes[0].ID

	tests := []struct {
		kind    string
		protoID string // non-empty only for PROTOTYPED
	}{
		{kind: "ONE_WAY"},
		{kind: "TWO_WAY"},
		{kind: "PROTOTYPED", protoID: protoID},
	}

	for i, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			srcID := addTestWhiteboardNode(t, wbID, fmt.Sprintf("src-%d", i), i*200, 0)
			tgtID := addTestWhiteboardNode(t, wbID, fmt.Sprintf("tgt-%d", i), i*200, 200)

			args := []string{"whiteboard", "link", wbID, "--source", srcID, "--target", tgtID, "--link-type", tt.kind}
			if tt.protoID != "" {
				args = append(args, "--link-prototype", tt.protoID)
			}
			runCmd(t, args...)

			// Verify the link appears in the whiteboard link list.
			linksOut := runCmd(t, "whiteboard", "links", wbID, "--format", "json")
			var links []struct {
				SourceCard struct {
					ID string `json:"id"`
				} `json:"sourceCard"`
				TargetCard struct {
					ID string `json:"id"`
				} `json:"targetCard"`
			}
			require.NoError(t, json.Unmarshal([]byte(linksOut), &links))

			found := false
			for _, l := range links {
				if l.SourceCard.ID == srcID && l.TargetCard.ID == tgtID {
					found = true
					break
				}
			}
			assert.True(t, found, "link %s→%s not found in whiteboard links after %s creation", srcID, tgtID, tt.kind)
		})
	}
}
