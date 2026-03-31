package api_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProject_CRUD(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	got, err := c.GetProject(proj.ShortName)
	require.NoError(t, err)
	assert.Equal(t, proj.ShortName, got.ShortName)

	projects, err := c.ListProjects()
	require.NoError(t, err)
	found := false
	for _, p := range projects {
		if p.ShortName == proj.ShortName {
			found = true
			break
		}
	}
	assert.True(t, found, "created project should appear in list")
}

func TestProject_GetNotFound(t *testing.T) {
	c := newTestClient(t)
	_, err := c.GetProject("DEFINITELYNOTEXIST99")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DEFINITELYNOTEXIST99")
}

func TestProject_Archive(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	err := c.ArchiveProject(proj.ShortName)
	require.NoError(t, err)
}

func TestProject_TimeTracking(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	err := c.SetProjectTimeTracking(proj.ShortName, true)
	require.NoError(t, err)

	err = c.SetProjectTimeTracking(proj.ShortName, false)
	require.NoError(t, err)
}

func TestProject_Templates(t *testing.T) {
	c := newTestClient(t)
	templates, err := c.GetProjectTemplates()
	require.NoError(t, err)
	assert.NotNil(t, templates)
}

func TestProject_CustomFields(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	fields, err := c.GetProjectCustomFields(proj.ShortName)
	require.NoError(t, err)
	assert.NotNil(t, fields)
}

func TestCmd_ProjectList(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	out := runCmd(t, "project", "list", "--format", "json")
	assert.Contains(t, out, proj.ShortName)
}

func TestCmd_ProjectGet(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	out := runCmd(t, "project", "get", proj.ShortName, "--format", "json")
	assert.Contains(t, out, proj.ShortName)
}

func TestCmd_ProjectAdd(t *testing.T) {
	c := newTestClient(t)
	me, err := c.GetCurrentUser()
	require.NoError(t, err)

	shortName := fmt.Sprintf("CMDP%d", time.Now().UnixMilli()%100000)
	t.Cleanup(func() { _ = c.DeleteProject(shortName) })

	out := runCmd(t, "project", "add",
		"--shortName", shortName,
		"--name", "Cmd Test Project",
		"--leader", me.Login,
		"--format", "json",
	)
	assert.Contains(t, out, shortName)
}
