package api_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"testing"

	"github.com/JetBrains/youtrack-cli/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmd_ListRejectsNegativePagination(t *testing.T) {
	c := newTestClient(t)
	tag, err := c.AddTag(api.AddTagRequest{Name: "pagination-" + uniqueProjectShortName("P")})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteTag(tag.ID) })
	for _, tc := range []struct {
		name string
		args []string
		flag string
	}{
		{"issue limit", []string{"issue", "list", "--query", "#Unresolved", "--limit", "-1"}, "--limit"},
		{"issue offset", []string{"issue", "list", "--query", "#Unresolved", "--offset", "-1"}, "--offset"},
		{"article limit", []string{"article", "list", "--limit", "-1"}, "--limit"},
		{"user limit", []string{"user", "list", "--limit", "-1"}, "--limit"},
		{"tag issue limit", []string{"tag", "issue", "list", tag.ID, "--limit", "-1"}, "--limit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(binaryPath, append(tc.args, "--json")...)
			cmd.Env = append(os.Environ(), "HOME="+t.TempDir())
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			require.Error(t, cmd.Run(), "negative pagination must fail")
			assert.Empty(t, stdout.String(), "invalid pagination must not return data")
			var result struct {
				Error struct {
					Message string `json:"message"`
				} `json:"error"`
			}
			require.NoError(t, json.Unmarshal(stderr.Bytes(), &result))
			assert.Contains(t, result.Error.Message, tc.flag+" must not be negative")
		})
	}
}

func TestCmd_ListZeroLimit(t *testing.T) {
	c := newTestClient(t)
	tag, err := c.AddTag(api.AddTagRequest{Name: "pagination-zero-" + uniqueProjectShortName("P")})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteTag(tag.ID) })
	for _, args := range [][]string{
		{"issue", "list", "--query", "#Unresolved", "--limit", "0", "--offset", "0", "--json"},
		{"article", "list", "--limit", "0", "--json"},
		{"user", "list", "--limit", "0", "--json"},
		{"tag", "issue", "list", tag.ID, "--limit", "0", "--json"},
	} {
		t.Run(args[0], func(t *testing.T) {
			assert.JSONEq(t, "[]", runCmd(t, args...))
		})
	}
}
