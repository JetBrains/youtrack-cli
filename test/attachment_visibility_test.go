package api_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmd_AttachmentVisibilityOnUpload(t *testing.T) {
	c := newTestClient(t)
	f := newVisibilityFixture(t, c)
	project := strings.Split(f.issue, "-")[0]
	_, err := c.AddUserToProjectTeam(project, f.user.ID)
	require.NoError(t, err)
	token := visibilityUserToken(t, c, f.user)
	// Establish that this reader can see the parent, so denial proves the attachment restriction.
	parent := runJSON[api.Issue](t, "--token", token, "issue", "view", f.issue, "--json")
	assert.Equal(t, f.issue, parent.IDReadable)
	me, err := c.GetCurrentUser()
	require.NoError(t, err)
	file := filepath.Join(t.TempDir(), "private.txt")
	require.NoError(t, os.WriteFile(file, []byte("private upload content"), 0o600))
	for _, audience := range [][]string{
		{"group:" + f.group.ID}, {"user:" + me.ID}, {"group:" + f.group.ID, "user:" + me.ID},
	} {
		t.Run(strings.Join(audience, "+"), func(t *testing.T) {
			args := []string{"issue", "attachment", "add", f.issue, "--file", file, "--json"}
			for _, principal := range audience {
				args = append(args, "--visibility", principal)
			}
			uploaded := runJSON[api.IssueAttachment](t, args...)
			require.NotEmpty(t, uploaded.ID)
			require.NotNil(t, uploaded.Visibility)
			assert.Equal(t, "LimitedVisibility", uploaded.Visibility.Type)
			stored := readVisibility(t, c, fmt.Sprintf("/api/issues/%s/attachments/%s", f.issue, uploaded.ID))
			groups, users := permittedIDs(stored)
			if strings.HasPrefix(audience[0], "group:") {
				assert.Equal(t, []string{f.group.ID}, groups)
			} else {
				assert.Empty(t, groups)
			}
			if strings.HasPrefix(audience[len(audience)-1], "user:") {
				assert.Equal(t, []string{me.ID}, users)
			} else {
				assert.Empty(t, users)
			}
			denied := runCmdExpectError(t, "--token", token, "raw", fmt.Sprintf("/api/issues/%s/attachments/%s", f.issue, uploaded.ID), "--fields", "id,url")
			assert.NotContains(t, denied, "sign=")
			listed := runJSON[[]api.IssueAttachment](t, "--token", token, "issue", "attachment", "list", f.issue, "--json")
			assert.Empty(t, listed, "an excluded reader must not discover signed URLs")
		})
	}
	public := runJSON[api.IssueAttachment](t, "issue", "attachment", "add", f.issue, "--file", file, "--public", "--json")
	require.NotNil(t, public.Visibility)
	assert.Equal(t, "UnlimitedVisibility", public.Visibility.Type)
	listed := runJSON[[]api.IssueAttachment](t, "--token", token, "issue", "attachment", "list", f.issue, "--json")
	require.Len(t, listed, 1)
	assert.Equal(t, public.ID, listed[0].ID, "an excluded reader may discover an explicitly public attachment")
	before, err := c.GetIssueAttachments(f.issue)
	require.NoError(t, err)
	runCmdExpectError(t, "issue", "attachment", "add", f.issue, "--file", file, "--visibility", "group:4-999999999")
	after, err := c.GetIssueAttachments(f.issue)
	require.NoError(t, err)
	assert.Len(t, after, len(before), "invalid audiences must not leave an unrestricted upload")
}
