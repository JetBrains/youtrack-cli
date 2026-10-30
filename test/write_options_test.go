package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

// runCmdStreams runs the binary like runCmd and returns stdout and stderr separately.
func runCmdStreams(t *testing.T, args ...string) (stdout, stderr string) {
	t.Helper()
	requireE2EEnv(t)

	cmd := exec.Command(binaryPath, args...)
	cmd.Env = append(os.Environ(), "HOME="+t.TempDir())
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	require.NoError(t, cmd.Run(), "binary stdout: %s\nbinary stderr: %s", out.String(), errOut.String())
	return out.String(), errOut.String()
}

func createWriteOptionsIssue(t *testing.T) (*api.Client, string) {
	t.Helper()
	c := newTestClient(t)
	proj := createTempProject(t, c)
	issue, err := c.AddIssue(api.AddIssueRequest{Summary: "write options issue", Project: &api.Project{ID: proj.ID}})
	require.NoError(t, err)
	return c, issue.IDReadable
}

func TestCmd_DryRunPrintsWriteWithoutSending(t *testing.T) {
	c, issueID := createWriteOptionsIssue(t)

	var req struct {
		DryRun bool           `json:"dryRun"`
		Method string         `json:"method"`
		URL    string         `json:"url"`
		Body   map[string]any `json:"body"`
	}
	out := runCmd(t, "issue", "comment", "create", issueID, "--text", "dry run comment", "--dry-run", "--json")
	require.NoError(t, json.Unmarshal([]byte(out), &req), out)
	assert.True(t, req.DryRun)
	assert.Equal(t, "POST", req.Method)
	assert.Contains(t, req.URL, "/api/issues/"+issueID+"/comments")
	assert.Equal(t, "dry run comment", req.Body["text"])

	comments, err := c.ListComments(issueID)
	require.NoError(t, err)
	assert.Empty(t, comments, "dry run must not create the comment")

	plain, _ := runCmdStreams(t, "issue", "delete", issueID, "--dry-run", "--plain")
	assert.Contains(t, plain, "DELETE ")
	assert.Contains(t, plain, "/api/issues/"+issueID)
	_, err = c.GetIssue(issueID)
	require.NoError(t, err, "dry run must not delete the issue")

	missing := runCmdExpectError(t, "issue", "attachment", "add", issueID, "--file", t.TempDir()+"/missing.txt", "--dry-run")
	assert.Contains(t, missing, "no such file")

	stderr := runCmdExpectError(t, "instance", "delete", "missing", "--dry-run")
	assert.Contains(t, stderr, "does not support it")
}

func TestCmd_MuteNotificationsAddsParam(t *testing.T) {
	c, issueID := createWriteOptionsIssue(t)

	_, stderr := runCmdStreams(t, "issue", "comment", "create", issueID, "--text", "quiet comment", "--mute-notifications", "--verbose", "--json")
	assert.Contains(t, stderr, "/api/issues/"+issueID+"/comments?")
	assert.Contains(t, stderr, "muteUpdateNotifications=true")
	assert.NotContains(t, stderr, "does not accept muteUpdateNotifications")

	comments, err := c.ListComments(issueID)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "quiet comment", comments[0].Text)

	var req struct {
		URL  string         `json:"url"`
		Body map[string]any `json:"body"`
	}
	out := runCmd(t, "issue", "command", issueID, "tag Star", "--mute-notifications", "--dry-run", "--json")
	require.NoError(t, json.Unmarshal([]byte(out), &req), out)
	assert.Contains(t, req.URL, "/api/commands?muteUpdateNotifications=true")
	assert.Equal(t, true, req.Body["silent"])
}

func TestCmd_MuteNotificationsIncludesDeletes(t *testing.T) {
	_, issueID := createWriteOptionsIssue(t)
	out, stderr := runCmdStreams(t, "issue", "tag", "remove", issueID, "1-1", "--mute-notifications", "--dry-run")
	assert.Contains(t, out, "muteUpdateNotifications=true")
	assert.NotContains(t, stderr, "warning:")
}

func TestCmd_DryRunRedactsSecrets(t *testing.T) {
	requireE2EEnv(t)
	for _, format := range []string{"--plain", "--json"} {
		out := runCmd(t, "user", "create", "--login", "dry-secret", "--fullName", "Dry Secret", "--email", "dry@example.com", "--password", "S3cretNeverPrint", "--dry-run", format)
		assert.NotContains(t, out, "S3cretNeverPrint")
		assert.Contains(t, out, "***")
		nested := runCmd(t, "raw", "/api/issues", "--method", "POST", "--body", `{"password":"nested-password","items":[{"access_token":"nested-token","client-secret":"nested-secret","summary":"visible summary"}]}`, "--dry-run", format)
		for _, secret := range []string{"nested-password", "nested-token", "nested-secret"} {
			assert.NotContains(t, nested, secret)
		}
		assert.Contains(t, nested, "visible summary")
	}
}

// A regular project team member may write, but cannot silently write.
func TestCmd_MuteNotificationsRefusesUnauthorizedWrite(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	login := fmt.Sprintf("mute%d", time.Now().UnixNano())
	const password = "MuteTest1234!"
	user, err := c.AddUser(&api.AddUserRequest{Login: login, FullName: "Mute permission test", Email: login + "@example.com", Password: password})
	require.NoError(t, err)
	successor, err := c.GetCurrentUser()
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteUser(user.ID, successor.ID) })
	_, err = c.AddUserToProjectTeam(proj.ShortName, user.ID)
	require.NoError(t, err)
	issue, err := c.AddIssue(api.AddIssueRequest{Summary: "mute permission probe", Project: &api.Project{ID: proj.ID}})
	require.NoError(t, err)
	baseURL := os.Getenv("YOUTRACK_BASE_URL")
	me, err := api.GetHubMeContext(t.Context(), baseURL, login, password)
	require.NoError(t, err)
	services, err := api.ListHubServicesContext(t.Context(), baseURL, login, password)
	require.NoError(t, err)
	var scopes []string
	for _, service := range services {
		if service.Name == "YouTrack" {
			scopes = append(scopes, service.ID)
		}
	}
	require.NotEmpty(t, scopes)
	token, err := api.CreateHubTokenContext(t.Context(), baseURL, login, password, me.ID, "mute-permission-test", scopes)
	require.NoError(t, err)
	t.Cleanup(func() { _ = api.DeleteHubTokenContext(context.Background(), baseURL, login, password, me.ID, token.ID) })
	// Verify ordinary writes work, so the 403 below is specifically the mute permission.
	runCmd(t, "issue", "comment", "create", issue.IDReadable, "--text", "normal", "--token", token.Token)
	denied := runCmdExpectError(t, "issue", "comment", "create", issue.IDReadable, "--text", "muted", "--token", token.Token, "--mute-notifications")
	assert.Contains(t, strings.ToLower(denied), "apply commands silently")
	assert.Contains(t, denied, "notifications are acceptable")
	// Tags and deletes must use the same mute policy as comments.
	userClient := api.NewClient(baseURL, token.Token)
	tag, err := userClient.AddTag(api.AddTagRequest{Name: "mute-" + login})
	require.NoError(t, err)
	t.Cleanup(func() { _ = userClient.DeleteTag(tag.ID) })
	runCmd(t, "issue", "tag", "add", issue.IDReadable, tag.ID, "--token", token.Token)
	beforeTags, beforeErr := userClient.ListIssueTags(issue.IDReadable)
	require.NoError(t, beforeErr)
	require.Len(t, beforeTags, 1, "fixture tag should be attached before the muted delete")
	denied = runCmdExpectError(t, "issue", "tag", "remove", issue.IDReadable, tag.ID, "--token", token.Token, "--mute-notifications")
	assert.Contains(t, strings.ToLower(denied), "apply commands silently")
	tags, err := userClient.ListIssueTags(issue.IDReadable)
	require.NoError(t, err)
	require.Len(t, tags, 1)
	assert.Equal(t, tag.ID, tags[0].ID)

	// One project grants silent-command access; another deliberately does not.
	privileged := createTempProject(t, c)
	_, err = c.AddUserToProjectTeam(privileged.ShortName, user.ID)
	require.NoError(t, err)
	roles, err := c.ListRoles()
	require.NoError(t, err)
	var projectAdminRole string
	for _, role := range roles {
		if role.Name == "Project Admin" {
			projectAdminRole = role.ID
		}
	}
	require.NotEmpty(t, projectAdminRole)
	require.NoError(t, c.GrantRoleToUser(user.ID, projectAdminRole, privileged.ID, ""))
	other, err := c.AddIssue(api.AddIssueRequest{Summary: "other mute probe", Project: &api.Project{ID: privileged.ID}})
	require.NoError(t, err)
	// Team membership and role changes can take time to reach the user's permissions.
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		_, err := userClient.GetIssue(other.IDReadable)
		assert.NoError(collect, err)
	}, 10*time.Second, 20*time.Millisecond, "privileged issue must be readable before the bulk command")
	// Prove this project's silent permission works before testing the mixed batch.
	runCmd(t, "issue", "command", other.IDReadable, "tag "+tag.Name, "--token", token.Token, "--mute-notifications")
	runCmd(t, "issue", "tag", "remove", other.IDReadable, tag.ID, "--token", token.Token)
	denied = runCmdExpectError(t, "issue", "command", other.IDReadable+","+issue.IDReadable, "tag "+tag.Name, "--token", token.Token, "--mute-notifications")
	assert.Contains(t, strings.ToLower(denied), "apply commands silently")
	// The privileged issue must remain untagged even though it is first in the batch.
	otherTags, err := userClient.ListIssueTags(other.IDReadable)
	require.NoError(t, err)
	assert.Empty(t, otherTags)
	unchangedTags, err := userClient.ListIssueTags(issue.IDReadable)
	require.NoError(t, err)
	require.Len(t, unchangedTags, 1)
	assert.Equal(t, tag.ID, unchangedTags[0].ID)

	comments, err := c.ListComments(issue.IDReadable)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "normal", comments[0].Text)
}

func TestCmd_RawStreamDryRun(t *testing.T) {
	c, issueID := createWriteOptionsIssue(t)
	for _, method := range []string{"POST", "PUT", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "response.json")
			require.NoError(t, os.WriteFile(path, []byte("previous"), 0o600))
			out := runCmd(t, "raw", "/api/issues/"+issueID, "--method", method,
				"--body", `{"summary":"must not change"}`, "--output", path, "--dry-run", "--mute-notifications", "--json")
			var preview struct {
				DryRun bool   `json:"dryRun"`
				Method string `json:"method"`
				URL    string `json:"url"`
			}
			require.NoError(t, json.Unmarshal([]byte(out), &preview))
			assert.True(t, preview.DryRun)
			assert.Equal(t, method, preview.Method)
			assert.Contains(t, preview.URL, "muteUpdateNotifications=true")
			got, err := c.GetIssue(issueID)
			require.NoError(t, err)
			assert.Equal(t, "write options issue", got.Summary)
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, "previous", string(data))
			files, err := os.ReadDir(filepath.Dir(path))
			require.NoError(t, err)
			assert.Len(t, files, 1)
		})
	}
}
