package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	shortName := uniqueProjectShortName("CMDP")

	out := runCmd(t, "project", "add",
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
