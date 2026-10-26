package api_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tagNames(tags []api.Tag) []string {
	names := make([]string, 0, len(tags))
	for _, t := range tags {
		names = append(names, t.Name)
	}
	return names
}

func createTag(t *testing.T, c *api.Client, name string) *api.Tag {
	t.Helper()
	tag, err := c.AddTag(api.AddTagRequest{Name: name})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteTag(tag.ID) })
	return tag
}

func TestCmd_TagListQueryAndLimit(t *testing.T) {
	c := newTestClient(t)
	marker := fmt.Sprintf("tagq%d", time.Now().UnixNano()%1000000)
	createTag(t, c, marker+" alpha")
	createTag(t, c, marker+" beta")
	createTag(t, c, "other x"+marker)

	matched := runJSON[[]api.Tag](t, "tag", "list", "--query", marker, "--json")
	assert.ElementsMatch(t, []string{marker + " alpha", marker + " beta"}, tagNames(matched))

	narrowed := runJSON[[]api.Tag](t, "tag", "list", "--query", marker+" alpha", "--json")
	assert.Equal(t, []string{marker + " alpha"}, tagNames(narrowed))

	limited := runJSON[[]api.Tag](t, "tag", "list", "--query", marker, "--limit", "1", "--json")
	assert.Len(t, limited, 1)

	assert.JSONEq(t, "[]", runCmd(t, "tag", "list", "--query", marker, "--limit", "0", "--json"))
	assert.Contains(t, runCmdExpectError(t, "tag", "list", "--limit", "-1", "--json"), "--limit must not be negative")
}

func TestCmd_IssueTagByName(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	issue, err := c.AddIssue(api.AddIssueRequest{Summary: "tag by name", Project: &api.Project{ID: proj.ID}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	marker := fmt.Sprintf("tagn%d", time.Now().UnixNano()%1000000)
	target := createTag(t, c, marker+" release blocker")
	createTag(t, c, marker+" release blocker later")

	out := runCmd(t, "issue", "tag", "add", issue.IDReadable, marker+" release blocker")
	assert.Contains(t, out, target.ID)
	onIssue, err := c.ListIssueTags(issue.IDReadable)
	require.NoError(t, err)
	assert.Equal(t, []string{target.Name}, tagNames(onIssue))

	t.Run("unknown name is not created", func(t *testing.T) {
		missing := marker + " does not exist"
		stderr := runCmdExpectError(t, "issue", "tag", "add", issue.IDReadable, missing, "--json")
		assert.Contains(t, stderr, "no tag named")
		found, err := c.ListTags(marker, 100)
		require.NoError(t, err)
		assert.NotContains(t, tagNames(found), missing)
	})

	t.Run("name differing in case is rejected", func(t *testing.T) {
		stderr := runCmdExpectError(t, "issue", "tag", "add", issue.IDReadable, marker+" RELEASE BLOCKER", "--json")
		assert.Contains(t, stderr, "differing only in case")
	})

	t.Run("remove resolves among the issue's tags", func(t *testing.T) {
		stderr := runCmdExpectError(t, "issue", "tag", "remove", issue.IDReadable, marker+" release blocker later", "--json")
		assert.Contains(t, stderr, "on issue "+issue.IDReadable)

		out := runCmd(t, "issue", "tag", "remove", issue.IDReadable, marker+" release blocker")
		assert.Contains(t, out, target.ID)
		after, err := c.ListIssueTags(issue.IDReadable)
		require.NoError(t, err)
		assert.Empty(t, after)
	})
}
