package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBoard_CRUD(t *testing.T) {
	c := newTestClient(t)

	// Create
	board, err := c.AddBoard("Test Board " + t.Name())
	require.NoError(t, err)
	require.NotEmpty(t, board.ID)
	t.Cleanup(func() { _ = c.DeleteBoard(board.ID) })

	assert.Contains(t, board.Name, "Test Board")

	// Get
	got, err := c.GetBoard(board.ID)
	require.NoError(t, err)
	assert.Equal(t, board.ID, got.ID)
	assert.Equal(t, board.Name, got.Name)

	// List
	boards, err := c.ListBoards()
	require.NoError(t, err)
	found := false
	for _, b := range boards {
		if b.ID == board.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "newly created board should appear in list")

	// Add sprint
	sprint, err := c.AddSprint(board.ID, "Sprint 1", 0, 0, "")
	require.NoError(t, err)
	assert.NotEmpty(t, sprint.ID)
	assert.Equal(t, "Sprint 1", sprint.Name)

	// Delete
	err = c.DeleteBoard(board.ID)
	require.NoError(t, err)
}
