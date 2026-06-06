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

// createTestCanvas creates a canvas owned by the current user and registers cleanup.
func createTestCanvas(t *testing.T, c *api.Client) string {
	t.Helper()
	me, err := c.GetCurrentUser()
	require.NoError(t, err)
	name := fmt.Sprintf("test-canvas-%d", time.Now().UnixMilli()%100000)

	out := runCmd(t, "canvas", "add", "--name", name, "--owner", me.Login, "--format", "json")
	var canvas struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &canvas))
	require.NotEmpty(t, canvas.ID)
	t.Cleanup(func() { _ = c.DeleteCanvas(canvas.ID) })
	return canvas.ID
}

// addTestCanvasNode adds a card node and returns its ID.
func addTestCanvasNode(t *testing.T, canvasID, label string, x, y int) string {
	t.Helper()
	out := runCmd(t, "canvas", "node", "add", canvasID,
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

func TestCmd_CanvasDeletePrintsSuccess(t *testing.T) {
	c := newTestClient(t)
	me, err := c.GetCurrentUser()
	require.NoError(t, err)

	canvas, err := c.AddCanvas(api.AddCanvasRequest{Name: "delete-test", Owner: &api.User{ID: me.ID}})
	require.NoError(t, err)

	out := runCmd(t, "canvas", "delete", canvas.ID)
	assert.Contains(t, out, canvas.ID)
}

func TestCmd_CanvasAdd(t *testing.T) {
	c := newTestClient(t)
	canvasID := createTestCanvas(t, c)
	assert.NotEmpty(t, canvasID)
}

func TestCmd_CanvasNodeAdd(t *testing.T) {
	c := newTestClient(t)
	canvasID := createTestCanvas(t, c)
	nodeID := addTestCanvasNode(t, canvasID, "Test Node", 50, 50)
	assert.NotEmpty(t, nodeID)
}

func TestCmd_CanvasLinkTypes(t *testing.T) {
	c := newTestClient(t)
	canvasID := createTestCanvas(t, c)

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
			srcID := addTestCanvasNode(t, canvasID, fmt.Sprintf("src-%d", i), i*200, 0)
			tgtID := addTestCanvasNode(t, canvasID, fmt.Sprintf("tgt-%d", i), i*200, 200)

			args := []string{"canvas", "link", canvasID, "--source", srcID, "--target", tgtID, "--link-type", tt.kind}
			if tt.protoID != "" {
				args = append(args, "--link-prototype", tt.protoID)
			}
			runCmd(t, args...)

			// Verify the link appears in the canvas link list.
			linksOut := runCmd(t, "canvas", "links", canvasID, "--format", "json")
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
			assert.True(t, found, "link %s→%s not found in canvas links after %s creation", srcID, tgtID, tt.kind)
		})
	}
}
