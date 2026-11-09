package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

func TestCmd_IssueUpdateExplicitFields(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	for _, tc := range []struct {
		name, summary, description string
		flags                      []string
	}{
		{"clear description", "original", "", []string{"--description", ""}},
		{"clear and rename", "renamed", "", []string{"--summary", "renamed", "--description", ""}},
		{"preserve omitted description", "renamed", "original description", []string{"--summary", "renamed"}},
		{"preserve omitted summary", "original", "new description", []string{"--description", "new description"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			issue, err := c.AddIssue(api.AddIssueRequest{
				Summary: "original", Description: "original description", Project: &api.Project{ID: proj.ID},
			})
			require.NoError(t, err)
			t.Cleanup(func() { _ = c.DeleteIssue(issue.ID) })

			args := append([]string{"issue", "update", issue.IDReadable, "--json"}, tc.flags...)
			updated := runJSON[api.Issue](t, args...)
			assert.Equal(t, tc.summary, updated.Summary)
			assert.Equal(t, tc.description, updated.Description)

			stored, err := c.GetIssue(issue.ID)
			require.NoError(t, err)
			assert.Equal(t, tc.summary, stored.Summary)
			assert.Equal(t, tc.description, stored.Description)
		})
	}

	// Flag presence is required even when an empty description is a valid update.
	out := runCmdExpectError(t, "issue", "update", "unused", "--json")
	assert.Contains(t, out, "at least one of --summary, --description or --description-file must be provided")
}
