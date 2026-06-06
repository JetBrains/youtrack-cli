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

// TestCmd_IssueViewAlias verifies that "issue view" routes to the same handler as "issue get".
func TestCmd_IssueViewAlias(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "view alias test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	out := runCmd(t, "issue", "view", issue.IDReadable, "--format", "json")
	assert.Contains(t, out, issue.IDReadable)
	assert.Contains(t, out, "view alias test")
}

// TestCmd_IssueCreateAlias verifies that "issue create" routes to the same handler as "issue add".
func TestCmd_IssueCreateAlias(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	out := runCmd(t, "issue", "create",
		"--project", proj.ShortName,
		"--summary", "create alias test",
		"--format", "json",
	)
	assert.Contains(t, out, proj.ShortName)
	assert.Contains(t, out, "create alias test")
}

// TestCmd_UserMeAlias verifies that "user me" routes to the same handler as "user current".
func TestCmd_UserMeAlias(t *testing.T) {
	out := runCmd(t, "user", "me", "--format", "json")
	assert.Contains(t, out, "login")
}

// TestCmd_UserViewAlias verifies that "user view" routes to the same handler as "user get".
func TestCmd_UserViewAlias(t *testing.T) {
	c := newTestClient(t)
	me, err := c.GetCurrentUser()
	require.NoError(t, err)

	out := runCmd(t, "user", "view", me.Login, "--format", "json")
	assert.Contains(t, out, me.Login)
}

// TestCmd_UserCreateAlias verifies that "user create" routes to the same handler as "user add".
func TestCmd_UserCreateAlias(t *testing.T) {
	c := newTestClient(t)
	successor, err := c.GetCurrentUser()
	require.NoError(t, err)

	login := fmt.Sprintf("alias%d", time.Now().UnixMilli()%100000)
	t.Cleanup(func() { _ = c.DeleteUser(login, successor.Login) })

	out := runCmd(t, "user", "create",
		"--login", login,
		"--fullName", "Alias Test User",
		"--email", login+"@example.com",
		"--password", "Test1234!",
		"--format", "json",
	)
	assert.Contains(t, out, login)
}

// TestCmd_ProjectViewAlias verifies that "project view" routes to the same handler as "project get".
func TestCmd_ProjectViewAlias(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	out := runCmd(t, "project", "view", proj.ShortName, "--format", "json")
	assert.Contains(t, out, proj.ShortName)
}

// TestCmd_ProjectCreateAlias verifies that "project create" routes to the same handler as "project add".
func TestCmd_ProjectCreateAlias(t *testing.T) {
	c := newTestClient(t)
	me, err := c.GetCurrentUser()
	require.NoError(t, err)

	shortName := uniqueProjectShortName("ALS")
	out := runCmd(t, "project", "create",
		"--shortName", shortName,
		"--name", "Alias Test Project",
		"--leader", me.Login,
		"--format", "json",
	)
	assert.Contains(t, out, shortName)
}

// TestCmd_BoardViewAlias verifies that "board view" routes to the same handler as "board get".
func TestCmd_BoardViewAlias(t *testing.T) {
	c := newTestClient(t)
	board, err := c.AddBoard("View Alias Board " + t.Name())
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteBoard(board.ID) })

	out := runCmd(t, "board", "view", board.ID, "--format", "json")
	assert.Contains(t, out, board.ID)
}

// TestCmd_BoardCreateAlias verifies that "board create" routes to the same handler as "board add".
func TestCmd_BoardCreateAlias(t *testing.T) {
	c := newTestClient(t)
	boardName := "Create Alias Board " + t.Name()

	out := runCmd(t, "board", "create", "--name", boardName, "--format", "json")
	assert.Contains(t, out, boardName)

	var created api.Board
	require.NoError(t, json.Unmarshal([]byte(out), &created))
	require.NotEmpty(t, created.ID)
	t.Cleanup(func() { _ = c.DeleteBoard(created.ID) })
}

// TestCmd_DashboardCreateAlias verifies that "dashboard create" routes to the same handler as "dashboard add".
func TestCmd_DashboardCreateAlias(t *testing.T) {
	c := newTestClient(t)
	name := "Create Alias Dashboard " + t.Name()

	out := runCmd(t, "dashboard", "create", "--name", name, "--format", "json")
	assert.Contains(t, out, name)

	dashboards, err := c.ListDashboards()
	require.NoError(t, err)
	for _, d := range dashboards {
		if d.Name == name {
			t.Cleanup(func() { _ = c.DeleteDashboard(d.ID) })
			break
		}
	}
}

// TestCmd_GroupViewAlias verifies that "group view" routes to the same handler as "group get".
func TestCmd_GroupViewAlias(t *testing.T) {
	c := newTestClient(t)
	groupName := fmt.Sprintf("aliasgp-view-%d", time.Now().UnixMilli()%100000)
	group, err := c.AddGroup(groupName)
	require.NoError(t, err)
	successor := groupDeleteSuccessor(t, c, group.ID)
	t.Cleanup(func() { _ = c.DeleteGroup(group.ID, successor.ID) })

	out := runCmd(t, "group", "view", group.Name, "--format", "json")
	assert.Contains(t, out, group.Name)
}

// TestCmd_GroupCreateAlias verifies that "group create" routes to the same handler as "group add".
func TestCmd_GroupCreateAlias(t *testing.T) {
	c := newTestClient(t)
	groupName := fmt.Sprintf("aliasgp-create-%d", time.Now().UnixMilli()%100000)

	out := runCmd(t, "group", "create", "--name", groupName, "--format", "json")
	assert.Contains(t, out, groupName)

	groups, err := c.ListGroups(groupName)
	require.NoError(t, err)
	for _, g := range groups {
		if g.Name == groupName {
			successor := groupDeleteSuccessor(t, c, g.ID)
			t.Cleanup(func() { _ = c.DeleteGroup(g.ID, successor.ID) })
			break
		}
	}
}

// TestCmd_AuthLoginAliasRegistered verifies that "auth login" is registered as a subcommand.
func TestCmd_AuthLoginAliasRegistered(t *testing.T) {
	out := runCmd(t, "auth", "--help")
	assert.Contains(t, out, "login")
}
