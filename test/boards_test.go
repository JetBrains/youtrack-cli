package api_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

func TestCmd_BoardList(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	board, err := c.AddBoard("Test Board "+t.Name(), proj.ShortName)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteBoard(board.ID) })

	boards := runJSON[[]api.Board](t, "board", "list", "--json")
	require.True(t, containsBoard(boards, board.ID))
}

func TestCmd_BoardView(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	board, err := c.AddBoard("Test Board "+t.Name(), proj.ShortName)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteBoard(board.ID) })

	result := runJSON[api.Board](t, "board", "view", board.ID, "--json")
	require.Equal(t, board.ID, result.ID)
}

func TestCmd_BoardCreate(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	name := "e2e-board-add-" + t.Name()

	result := runJSON[api.Board](t, "board", "create", "--name", name, "--project", proj.ShortName, "--json")
	require.Equal(t, name, result.Name)

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

	out := runCmd(t, "board", "delete", board.ID, "--force")
	assert.Contains(t, out, "deleted board "+board.ID)
}

func TestCmd_BoardUpdate(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	board, err := c.AddBoard("e2e-board-upd-"+t.Name(), proj.ShortName)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteBoard(board.ID) })

	newName := "e2e-board-upd-renamed-" + t.Name()
	result := runJSON[api.Board](t, "board", "update", board.ID, "--name", newName, "--json")
	require.Equal(t, newName, result.Name)
}

func TestCmd_BoardSprintView(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	board, err := c.AddBoard("e2e-sprint-get-"+t.Name(), proj.ShortName)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteBoard(board.ID) })

	sprint, err := c.AddSprint(board.ID, "Sprint 1", 0, 0, "")
	require.NoError(t, err)

	result := runJSON[api.Sprint](t, "board", "sprint", "view", sprint.ID, "--board", board.ID, "--json")
	require.Equal(t, sprint.ID, result.ID)
}

func TestCmd_BoardSprintCreate(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	board, err := c.AddBoard("e2e-sprint-add-"+t.Name(), proj.ShortName)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteBoard(board.ID) })

	sprintName := "e2e-sprint-" + t.Name()
	result := runJSON[api.Sprint](t, "board", "sprint", "create",
		"--board", board.ID,
		"--name", sprintName,
		"--json",
	)
	require.Equal(t, sprintName, result.Name)
}

func TestCmd_BoardSprintUpdateAndDelete(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	board, err := c.AddBoard("e2e-sprint-lifecycle-"+t.Name(), proj.ShortName)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteBoard(board.ID) })
	sprint, err := c.AddSprint(board.ID, "Before", 0, 0, "old goal")
	require.NoError(t, err)

	updated := runJSON[api.Sprint](t, "board", "sprint", "update", sprint.ID,
		"--board", board.ID, "--name", "After", "--goal", "new goal", "--json")
	require.Equal(t, "After", updated.Name)
	require.Equal(t, "new goal", updated.Goal)

	out := runCmd(t, "board", "sprint", "delete", sprint.ID, "--board", board.ID, "--force")
	assert.Contains(t, out, "deleted sprint "+sprint.ID)
	require.Eventually(t, func() bool {
		sprints, listErr := c.ListSprints(board.ID)
		return listErr == nil && !containsSprint(sprints, sprint.ID)
	}, 5*time.Second, 100*time.Millisecond)
}

func containsSprint(sprints []api.Sprint, id string) bool {
	for _, sprint := range sprints {
		if sprint.ID == id {
			return true
		}
	}
	return false
}

func containsBoard(boards []api.Board, id string) bool {
	for _, board := range boards {
		if board.ID == id {
			return true
		}
	}
	return false
}
