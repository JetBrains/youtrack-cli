package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

func TestCmd_IssueDeleteCancellationFormats(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	issue, err := c.AddIssue(api.AddIssueRequest{Summary: "cancellation output", Project: &api.Project{ID: proj.ID}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.ID) })
	for _, tc := range []struct {
		name  string
		flags []string
		json  bool
	}{
		{"explicit JSON", []string{"--json"}, true},
		{"piped default", nil, true},
		{"quiet JSON", []string{"--json", "--quiet"}, true},
		{"legacy JSON", []string{"--format", "json"}, true},
		{"plain", []string{"--plain"}, false},
		{"table", []string{"--format", "table"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"issue", "delete", issue.IDReadable}, tc.flags...)
			out := runCmdWithInput(t, "n\n", args...)
			if tc.json {
				assert.JSONEq(t, `{"status":"canceled","message":"Canceled."}`, out)
			} else {
				assert.Equal(t, "Canceled.\n", out)
			}
			_, err := c.GetIssue(issue.ID)
			require.NoError(t, err, "declining deletion must preserve the issue")
		})
	}
}
