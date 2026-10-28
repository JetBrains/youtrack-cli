package api_test

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

func TestCmd_RawFieldsProjection(t *testing.T) {
	requireE2EEnv(t)

	out := runCmd(t,
		"--base-url", os.Getenv("YOUTRACK_BASE_URL"),
		"--token", os.Getenv("YOUTRACK_TOKEN"),
		"raw", "/api/users/me", "--fields", "login,fullName",
	)

	// with $fields= the server ignores the projection and returns only {$type,id}
	var result struct {
		Login string `json:"login"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.NotEmpty(t, result.Login)
}

func TestCmd_RawPaginate(t *testing.T) {
	requireE2EEnv(t)

	out := runCmd(t, "raw", "/api/users", "--fields", "id,login", "--paginate", "--page-size", "1")
	var users []map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &users))
	require.NotEmpty(t, users)
	for _, user := range users {
		assert.NotEmpty(t, user["id"])
	}
}

func TestCmd_RawUnexpectedResponse(t *testing.T) {
	requireE2EEnv(t)

	// The JSON error envelope is rendered on stderr; stdout carries only data.
	out := runCmdExpectError(t,
		"--base-url", os.Getenv("YOUTRACK_BASE_URL"),
		"--token", os.Getenv("YOUTRACK_TOKEN"),
		"raw", "/",
	)

	var result struct {
		Error struct {
			Code       string `json:"code"`
			Message    string `json:"message"`
			Suggestion string `json:"suggestion"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Equal(t, "unexpected_response", result.Error.Code)
	assert.Contains(t, result.Error.Message, "unexpected YouTrack API response")
	assert.Contains(t, result.Error.Suggestion, "youtrack auth status")
}

func TestCmd_RawBodyFile(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	issue, err := c.AddIssue(api.AddIssueRequest{Summary: "raw body file", Project: &api.Project{ID: proj.ID}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	// Longer than the 32,767-character Windows command line, with quotes that Windows would double.
	description := strings.Repeat(`line with "quotes" and \backslashes\ `, 1000)
	body, err := json.Marshal(map[string]string{"description": description})
	require.NoError(t, err)
	bodyPath := filepath.Join(t.TempDir(), "body.json")
	require.NoError(t, os.WriteFile(bodyPath, body, 0o600))

	path := "/api/issues/" + issue.IDReadable
	runCmd(t, "raw", path, "--method", "POST", "--body-file", bodyPath, "--fields", "id", "--json")
	got, err := c.GetIssue(issue.IDReadable)
	require.NoError(t, err)
	assert.Equal(t, description, got.Description)

	runCmdWithInput(t, `{"summary":"from stdin"}`, "raw", path, "--method", "POST", "--body-file", "-", "--fields", "id", "--json")
	got, err = c.GetIssue(issue.IDReadable)
	require.NoError(t, err)
	assert.Equal(t, "from stdin", got.Summary)

	stderr := runCmdExpectError(t, "raw", path, "--method", "POST", "--body", "{}", "--body-file", bodyPath)
	assert.Contains(t, stderr, "mutually exclusive")
}

func TestCmd_RawOutputBinary(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	issue, err := c.AddIssue(api.AddIssueRequest{Summary: "raw output", Project: &api.Project{ID: proj.ID}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for i := range img.Pix {
		img.Pix[i] = byte(i * 7)
	}
	var want bytes.Buffer
	require.NoError(t, png.Encode(&want, img))
	pngPath := filepath.Join(t.TempDir(), "pixel.png")
	require.NoError(t, os.WriteFile(pngPath, want.Bytes(), 0o600))
	attach, err := c.AddIssueAttachment(issue.IDReadable, pngPath)
	require.NoError(t, err)
	require.NotEmpty(t, attach.URL)

	outPath := filepath.Join(t.TempDir(), "out.png")
	require.NoError(t, os.WriteFile(outPath, []byte("previous"), 0o600))
	runCmd(t, "raw", attach.URL, "--output", outPath)
	got, err := os.ReadFile(outPath)
	require.NoError(t, err)
	assert.Equal(t, want.Bytes(), got)

	assert.Equal(t, want.Bytes(), []byte(runCmd(t, "raw", attach.URL, "--output", "-")))

	missing := filepath.Join(t.TempDir(), "missing.json")
	runCmdExpectError(t, "raw", "/api/issues/"+proj.ShortName+"-999999", "--output", missing)
	assert.NoFileExists(t, missing, "a failed request must not leave an output file")
	require.NoError(t, os.WriteFile(missing, []byte("keep existing data"), 0o600))
	runCmdExpectError(t, "raw", "/api/issues/"+proj.ShortName+"-999999", "--output", missing)
	kept, err := os.ReadFile(missing)
	require.NoError(t, err)
	assert.Equal(t, "keep existing data", string(kept))
}

func TestCmd_RawStreamReadOnly(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	issue, err := c.AddIssue(api.AddIssueRequest{Summary: "stream read-only", Project: &api.Project{ID: proj.ID}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })
	t.Setenv("YOUTRACK_RO", "1")
	for _, method := range []string{"POST", "PUT", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "response.json")
			require.NoError(t, os.WriteFile(path, []byte("previous"), 0o600))
			stderr := runCmdExpectError(t, "raw", "/api/issues/"+issue.IDReadable, "--method", method,
				"--body", `{"summary":"must not change"}`, "--output", path, "--json")
			assert.Contains(t, stderr, "read-only mode")
			stored, err := c.GetIssue(issue.IDReadable)
			require.NoError(t, err)
			assert.Equal(t, "stream read-only", stored.Summary)
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, "previous", string(data))
			files, err := os.ReadDir(filepath.Dir(path))
			require.NoError(t, err)
			assert.Len(t, files, 1)
		})
	}
}
