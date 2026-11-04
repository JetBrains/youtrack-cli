package api_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	// Team permissions propagate asynchronously through Hub.
	reader := api.NewClient(c.BaseURL, token)
	require.Eventually(t, func() bool {
		_, err := reader.GetIssue(f.issue)
		return err == nil
	}, 30*time.Second, 250*time.Millisecond, "reader must gain access to the parent issue")
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
			assert.NotEmpty(t, uploaded.URL, "visibility-aware uploads retain the attachment URL")
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
	assert.NotEmpty(t, public.URL, "explicitly public uploads retain the attachment URL")
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

func TestCmd_AttachmentVisibilityDryRun(t *testing.T) {
	c, issueID := createWriteOptionsIssue(t)
	me, err := c.GetCurrentUser()
	require.NoError(t, err)
	file := filepath.Join(t.TempDir(), "private.txt")
	content := "private content must not appear in a dry-run preview"
	require.NoError(t, os.WriteFile(file, []byte(content), 0o600))
	for _, flags := range [][]string{{"--visibility", "user:" + me.ID}, {"--public"}} {
		t.Run(flags[0], func(t *testing.T) {
			args := append([]string{"issue", "attachment", "add", issueID, "--file", file, "--dry-run", "--json"}, flags...)
			out := runCmd(t, args...)
			var preview struct {
				DryRun bool   `json:"dryRun"`
				Method string `json:"method"`
				Body   struct {
					File       string         `json:"file"`
					FileName   string         `json:"fileName"`
					Visibility api.Visibility `json:"visibility"`
				} `json:"body"`
			}
			require.NoError(t, json.Unmarshal([]byte(out), &preview))
			assert.True(t, preview.DryRun)
			assert.Equal(t, "POST", preview.Method)
			assert.Equal(t, file, preview.Body.File)
			assert.Equal(t, filepath.Base(file), preview.Body.FileName)
			if flags[0] == "--public" {
				assert.Equal(t, "UnlimitedVisibility", preview.Body.Visibility.Type)
			} else {
				assert.Equal(t, "LimitedVisibility", preview.Body.Visibility.Type)
				require.Len(t, preview.Body.Visibility.PermittedUsers, 1)
				assert.Equal(t, me.ID, preview.Body.Visibility.PermittedUsers[0].ID)
			}
			assert.NotContains(t, out, "base64Content")
			assert.NotContains(t, out, content)
			assert.NotContains(t, out, base64.StdEncoding.EncodeToString([]byte(content)))
		})
	}
	attachments, err := c.GetIssueAttachments(issueID)
	require.NoError(t, err)
	assert.Empty(t, attachments, "dry-run must not upload any attachment")
	missing := runCmdExpectError(t, "issue", "attachment", "add", issueID, "--file", file+".missing", "--public", "--dry-run")
	assert.Contains(t, missing, "no such file")
}
