package api_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
