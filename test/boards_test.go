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

func TestCmd_BoardAdd(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	name := "e2e-board-add-" + t.Name()

	out := runCmd(t, "board", "add", "--name", name, "--project", proj.ShortName, "--format", "json")
	assert.Contains(t, out, name)

	boards, err := c.ListBoards()
	require.NoError(t, err)
	for _, b := range boards {
		if b.Name == name {
			b := b
			t.Cleanup(func() { _ = c.DeleteBoard(b.ID) })
		}
	}
}

func TestCmd_BoardDelete(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	board, err := c.AddBoard("e2e-board-del-"+t.Name(), proj.ShortName)
	require.NoError(t, err)

	out := runCmd(t, "board", "delete", board.ID)
	assert.Contains(t, out, "deleted board "+board.ID)
}

func TestCmd_BoardUpdate(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	board, err := c.AddBoard("e2e-board-upd-"+t.Name(), proj.ShortName)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteBoard(board.ID) })

	newName := "e2e-board-upd-renamed-" + t.Name()
	out := runCmd(t, "board", "update", board.ID, "--name", newName, "--format", "json")
	assert.Contains(t, out, newName)
}

func TestCmd_BoardSprintGet(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	board, err := c.AddBoard("e2e-sprint-get-"+t.Name(), proj.ShortName)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteBoard(board.ID) })

	sprint, err := c.AddSprint(board.ID, "Sprint 1", 0, 0, "")
	require.NoError(t, err)

	out := runCmd(t, "board", "sprint", "get", sprint.ID, "--board", board.ID, "--format", "json")
	assert.Contains(t, out, sprint.ID)
}

func TestCmd_BoardSprintAdd(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	board, err := c.AddBoard("e2e-sprint-add-"+t.Name(), proj.ShortName)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteBoard(board.ID) })

	sprintName := "e2e-sprint-" + t.Name()
	out := runCmd(t, "board", "sprint", "add",
		"--board", board.ID,
		"--name", sprintName,
		"--format", "json",
	)
	assert.Contains(t, out, sprintName)
}
