package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmd_BoardList(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	board, err := c.AddBoard("Test Board "+t.Name(), proj.ShortName)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteBoard(board.ID) })

	out := runCmd(t, "board", "list", "--format", "json")
	assert.Contains(t, out, board.Name)
}

func TestCmd_BoardGet(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	board, err := c.AddBoard("Test Board "+t.Name(), proj.ShortName)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteBoard(board.ID) })

	out := runCmd(t, "board", "get", board.ID, "--format", "json")
	assert.Contains(t, out, board.ID)
}
