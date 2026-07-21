package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

func TestCmd_ProjectList(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	out := runCmd(t, "project", "list", "--format", "json")
	assert.Contains(t, out, proj.ShortName)
}

func TestCmd_ProjectView(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	out := runCmd(t, "project", "view", proj.ShortName, "--format", "json")
	assert.Contains(t, out, proj.ShortName)
}

func TestCmd_ProjectCreate(t *testing.T) {
	c := newTestClient(t)
	me, err := c.GetCurrentUser()
	require.NoError(t, err)

	shortName := uniqueProjectShortName("CMDP")

	out := runCmd(t, "project", "create",
		"--shortName", shortName,
		"--name", "Cmd Test Project",
		"--leader", me.Login,
		"--format", "json",
	)
	assert.Contains(t, out, shortName)
}

func TestCmd_ProjectArchive(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	out := runCmd(t, "project", "archive", proj.ShortName)
	assert.Contains(t, out, "archived project "+proj.ShortName)

	var result struct {
		Archived bool `json:"archived"`
	}
	err := c.Get("/api/admin/projects/"+url.PathEscape(proj.ID)+"?fields=archived", &result)
	require.NoError(t, err)
	assert.True(t, result.Archived)
}

func TestCmd_ProjectTeamAddUser(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	me, err := c.GetCurrentUser()
	require.NoError(t, err)

	user, err := c.GetUser(me.Login)
	require.NoError(t, err)

	// Probe: some instances do not support POST /team/users (returns 405).
	if _, probeErr := c.AddUserToProjectTeam(proj.ShortName, user.ID); probeErr != nil {
		t.Skipf("project team add-user not supported on this instance: %v", probeErr)
	}

	// Use a fresh project so the probe did not already add the user.
	proj2 := createTempProject(t, c)
	out := runCmd(t, "project", "team", "add-user", proj2.ShortName, user.ID)
	assert.Contains(t, out, "added")
}

func TestCmd_ProjectTeamAddGroup(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	groupName := "TestTeamGroup-" + proj.ShortName
	group, err := c.AddGroup(groupName)
	require.NoError(t, err)

	successor := groupDeleteSuccessor(t, c, group.ID)
	t.Cleanup(func() { _ = c.DeleteGroup(group.ID, successor.ID) })

	out := runCmd(t, "project", "team", "add-group", proj.ShortName, group.ID)
	assert.Contains(t, out, "added")
}

func TestCmd_ProjectFieldList(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	out := runCmd(t, "project", "field", "list", proj.ShortName, "--format", "json")
	assert.Contains(t, out, "[")
}

func TestCmd_ProjectFieldListWithFieldsProjection(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	out := runCmd(t, "project", "field", "list", proj.ShortName, "--fields", "id,field(id,name)", "--format", "json")
	assert.NotContains(t, out, "not found")
}

func TestCmd_ProjectFieldAdd(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	fields, err := c.ListCustomFields()
	require.NoError(t, err)
	if len(fields) == 0 {
		t.Skip("no global custom fields available on this instance")
	}

	// Pick a field not already attached to the project.
	existing, err := c.GetProjectCustomFields(proj.ShortName)
	require.NoError(t, err)
	existingIDs := make(map[string]bool, len(existing))
	for _, pf := range existing {
		if pf.Field != nil {
			existingIDs[pf.Field.ID] = true
		}
	}

	var target *api.CustomField
	for i := range fields {
		if !existingIDs[fields[i].ID] {
			target = &fields[i]
			break
		}
	}
	if target == nil {
		t.Skip("all global custom fields are already attached to the project")
	}

	runCmd(t, "project", "field", "add", proj.ShortName, "--field", target.ID)
	// Verify the field is now listed for the project.
	out := runCmd(t, "project", "field", "list", proj.ShortName, "--format", "json")
	assert.Contains(t, out, target.Name)
}

func TestCmd_ProjectTimetrack(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	out := runCmd(t, "project", "timetrack", "set", proj.ShortName, "--enabled")
	assert.Contains(t, out, "updated time tracking for project "+proj.ShortName)
}
