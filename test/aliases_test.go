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

// TestCmd_IssueViewAlias verifies that "issue get" still works as an alias of "issue view".
func TestCmd_IssueViewAlias(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "view alias test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	out := runCmd(t, "issue", "get", issue.IDReadable, "--format", "json")
	assert.Contains(t, out, issue.IDReadable)
	assert.Contains(t, out, "view alias test")
}

// TestCmd_IssueCreateAlias verifies that "issue add" still works as an alias of "issue create".
func TestCmd_IssueCreateAlias(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	out := runCmd(t, "issue", "add",
		"--project", proj.ShortName,
		"--summary", "create alias test",
		"--format", "json",
	)
	assert.Contains(t, out, proj.ShortName)
	assert.Contains(t, out, "create alias test")
}

// TestCmd_UserMeAlias verifies that "user me" still works as an alias of "user current".
func TestCmd_UserMeAlias(t *testing.T) {
	out := runCmd(t, "user", "me", "--format", "json")
	assert.Contains(t, out, "login")
}

// TestCmd_UserViewAlias verifies that "user get" still works as an alias of "user view".
func TestCmd_UserViewAlias(t *testing.T) {
	c := newTestClient(t)
	me, err := c.GetCurrentUser()
	require.NoError(t, err)

	out := runCmd(t, "user", "get", me.Login, "--format", "json")
	assert.Contains(t, out, me.Login)
}

// TestCmd_UserCreateAlias verifies that "user add" still works as an alias of "user create".
func TestCmd_UserCreateAlias(t *testing.T) {
	c := newTestClient(t)
	successor, err := c.GetCurrentUser()
	require.NoError(t, err)

	login := fmt.Sprintf("alias%d", time.Now().UnixMilli()%100000)
	t.Cleanup(func() { _ = c.DeleteUser(login, successor.Login) })

	out := runCmd(t, "user", "add",
		"--login", login,
		"--fullName", "Alias Test User",
		"--email", login+"@example.com",
		"--password", "Test1234!",
		"--format", "json",
	)
	assert.Contains(t, out, login)
}

// TestCmd_ProjectViewAlias verifies that "project get" still works as an alias of "project view".
func TestCmd_ProjectViewAlias(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	out := runCmd(t, "project", "get", proj.ShortName, "--format", "json")
	assert.Contains(t, out, proj.ShortName)
}

// TestCmd_ProjectCreateAlias verifies that "project add" still works as an alias of "project create".
func TestCmd_ProjectCreateAlias(t *testing.T) {
	c := newTestClient(t)
	me, err := c.GetCurrentUser()
	require.NoError(t, err)

	shortName := uniqueProjectShortName("ALS")
	out := runCmd(t, "project", "add",
		"--shortName", shortName,
		"--name", "Alias Test Project",
		"--leader", me.Login,
		"--format", "json",
	)
	assert.Contains(t, out, shortName)
}

// TestCmd_BoardViewAlias verifies that "board get" still works as an alias of "board view".
func TestCmd_BoardViewAlias(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	board, err := c.AddBoard("View Alias Board "+t.Name(), proj.ShortName)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteBoard(board.ID) })

	out := runCmd(t, "board", "get", board.ID, "--format", "json")
	assert.Contains(t, out, board.ID)
}

// TestCmd_BoardCreateAlias verifies that "board add" still works as an alias of "board create".
func TestCmd_BoardCreateAlias(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	boardName := "Create Alias Board " + t.Name()

	out := runCmd(t, "board", "add", "--name", boardName, "--project", proj.ShortName, "--format", "json")
	assert.Contains(t, out, boardName)

	var created api.Board
	require.NoError(t, json.Unmarshal([]byte(out), &created))
	require.NotEmpty(t, created.ID)
	t.Cleanup(func() { _ = c.DeleteBoard(created.ID) })
}

// TestCmd_DashboardCreateAlias verifies that "dashboard add" still works as an alias of "dashboard create".
func TestCmd_DashboardCreateAlias(t *testing.T) {
	c := newTestClient(t)
	name := "Create Alias Dashboard " + t.Name()

	out := runCmd(t, "dashboard", "add", "--name", name, "--format", "json")
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

// TestCmd_GroupViewAlias verifies that "group get" still works as an alias of "group view".
func TestCmd_GroupViewAlias(t *testing.T) {
	c := newTestClient(t)
	groupName := fmt.Sprintf("aliasgp-view-%d", time.Now().UnixMilli()%100000)
	group, err := c.AddGroup(groupName)
	require.NoError(t, err)
	successor := groupDeleteSuccessor(t, c, group.ID)
	t.Cleanup(func() { _ = c.DeleteGroup(group.ID, successor.ID) })

	out := runCmd(t, "group", "get", group.Name, "--format", "json")
	assert.Contains(t, out, group.Name)
}

// TestCmd_GroupCreateAlias verifies that "group add" still works as an alias of "group create".
func TestCmd_GroupCreateAlias(t *testing.T) {
	c := newTestClient(t)
	groupName := fmt.Sprintf("aliasgp-create-%d", time.Now().UnixMilli()%100000)

	out := runCmd(t, "group", "add", "--name", groupName, "--format", "json")
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

// TestCmd_AuthLoginAliasRegistered verifies that "auth login" is registered as a subcommand
// (canonical name; "get-token" is its alias).
func TestCmd_AuthLoginAliasRegistered(t *testing.T) {
	out := runCmd(t, "auth", "--help")
	assert.Contains(t, out, "login")
}
