package api_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JetBrains/youtrack-cli/api"
)

func TestCmd_IssueFieldsProjection(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	me, err := c.GetCurrentUser()
	require.NoError(t, err)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary:     "fields projection test",
		Description: "test description content",
		Project:     &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	_, err = c.AddComment(issue.IDReadable, api.AddCommentRequest{Text: "test comment"})
	require.NoError(t, err)

	t.Run("scalar fields render as plain values without map[] or scientific notation", func(t *testing.T) {
		out := runCmd(t, "issue", "list",
			"--query", "project: "+proj.ShortName,
			"--fields", "idReadable,summary,created,updated,numberInProject,commentsCount,votes,isDraft",
			"--format", "plain",
		)
		assert.Contains(t, out, issue.IDReadable)
		assert.Contains(t, out, "fields projection test")
		assert.Contains(t, out, "2026-") // created rendered as date, not epoch
		assert.NotContains(t, out, "map[")
		assert.NotContains(t, out, "e+12") // no scientific notation for timestamps
	})

	t.Run("reporter auto-expands and displays login", func(t *testing.T) {
		out := runCmd(t, "issue", "list",
			"--query", "project: "+proj.ShortName,
			"--fields", "idReadable,reporter",
			"--format", "plain",
		)
		assert.Contains(t, out, me.Login)
		assert.NotContains(t, out, "map[")
	})

	t.Run("project auto-expands and displays shortName", func(t *testing.T) {
		out := runCmd(t, "issue", "list",
			"--query", "project: "+proj.ShortName,
			"--fields", "idReadable,project",
			"--format", "plain",
		)
		assert.Contains(t, out, proj.ShortName)
		assert.NotContains(t, out, "map[")
	})

	t.Run("comments array displays as count not raw map", func(t *testing.T) {
		out := runCmd(t, "issue", "list",
			"--query", "project: "+proj.ShortName,
			"--fields", "idReadable,commentsCount,comments",
			"--format", "plain",
		)
		assert.Contains(t, out, issue.IDReadable)
		assert.NotContains(t, out, "map[")
		// commentsCount is a scalar int — 1 after adding the comment above
		assert.Contains(t, out, "1")
	})

	t.Run("id and idReadable together use KEY header to avoid duplicate ID columns", func(t *testing.T) {
		out := runCmd(t, "issue", "list",
			"--query", "project: "+proj.ShortName,
			"--fields", "id,idReadable",
			"--format", "table",
		)
		lines := strings.Split(strings.TrimSpace(out), "\n")
		require.NotEmpty(t, lines)
		header := lines[0]
		assert.Contains(t, header, "KEY")
		assert.Equal(t, 1, strings.Count(header, "ID"), "expected exactly one ID column, got: %q", header)
	})

	t.Run("description renders on single line in table format", func(t *testing.T) {
		out := runCmd(t, "issue", "list",
			"--query", "project: "+proj.ShortName,
			"--fields", "idReadable,description",
			"--format", "table",
		)
		assert.Contains(t, out, "test description content")
		assert.NotContains(t, out, "map[")
	})
}

// issue list --fields mixes issue attributes and custom fields by name, like the visible columns of the issue list UI.
func TestCmd_IssueListCustomFieldColumns(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	priority := createProjectEnumField(t, c, proj, api.FieldTypeEnumSingle, "Critical")
	attachDefaultProjectFields(t, c, proj, "State", "Assignee")

	me, err := c.GetCurrentUser()
	require.NoError(t, err)

	suffix := strconv.FormatInt(time.Now().UnixMilli(), 36)
	notesName := "Notes " + suffix
	dueName := "Due " + suffix
	notes, err := c.AddCustomField(notesName, api.FieldTypeText)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteCustomField(notes.ID) })
	due, err := c.AddCustomField(dueName, api.FieldTypeDate)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteCustomField(due.ID) })
	require.NoError(t, c.AddProjectCustomField(proj.ShortName, notes.ID))
	require.NoError(t, c.AddProjectCustomField(proj.ShortName, due.ID))

	issue, err := c.AddIssue(api.AddIssueRequest{Summary: "custom column test", Project: &api.Project{ID: proj.ID}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })
	awaitIssueFields(t, c, issue.IDReadable, priority.Name, "State", "Assignee", notesName, dueName)
	require.NoError(t, c.SetCustomField(issue.IDReadable, priority.Name, "Critical", ""))
	runCmd(t, "issue", "assign", issue.IDReadable, me.Login)
	// 2026-10-13 12:00 UTC — YouTrack stores date-only values at noon UTC.
	require.NoError(t, c.Post("/api/issues/"+issue.IDReadable+"?fields=id", map[string]any{
		"customFields": []map[string]any{
			{"name": notesName, "$type": "TextIssueCustomField", "value": map[string]any{"text": "first note line\nsecond"}},
			{"name": dueName, "$type": "DateIssueCustomField", "value": int64(1791892800000)},
		},
	}, nil))

	stateField, err := c.GetCustomFieldValue(issue.IDReadable, "State")
	require.NoError(t, err)
	stateValue, _ := stateField.Value.(map[string]any)
	state, _ := stateValue["name"].(string)
	require.NotEmpty(t, state, "default project has no State value")

	query := "project: " + proj.ShortName

	t.Run("table shows attribute and custom field columns in requested order", func(t *testing.T) {
		out := runCmd(t, "--format", "table", "issue", "list", "--query", query,
			"--fields", "idReadable,summary,"+priority.Name+",state,Assignee,"+notesName+","+dueName)
		lines := strings.Split(strings.TrimSpace(out), "\n")
		require.Len(t, lines, 2, "output: %s", out)
		assert.Regexp(t, `^ID\s+SUMMARY\s+`+regexp.QuoteMeta(strings.ToUpper(priority.Name))+`\s+STATE\s+ASSIGNEE\s+`+regexp.QuoteMeta(strings.ToUpper(notesName))+`\s+`+regexp.QuoteMeta(strings.ToUpper(dueName))+`$`, lines[0])
		assert.Regexp(t, `^`+issue.IDReadable+`\s+custom column test\s+Critical\s+`+state+`\s+`+regexp.QuoteMeta(me.Login)+`\s+first note line\s+2026-10-13$`, lines[1])
	})

	t.Run("plain rows carry custom field values", func(t *testing.T) {
		out := runCmd(t, "issue", "list", "--query", query, "--fields", "idReadable,"+priority.Name+",State", "--plain")
		assert.Equal(t, issue.IDReadable+"\tCritical\t"+state+"\n", out)
	})

	t.Run("attribute and custom field names are case-insensitive", func(t *testing.T) {
		out := runCmd(t, "issue", "list", "--query", query, "--fields", "IdReadable,Summary,"+strings.ToUpper(priority.Name), "--plain")
		assert.Equal(t, issue.IDReadable+"\tcustom column test\tCritical\n", out)
	})

	t.Run("json returns only the requested custom fields", func(t *testing.T) {
		issues := runJSON[[]api.Issue](t, "issue", "list", "--query", query, "--fields", "idReadable,"+strings.ToLower(priority.Name)+",State", "--json")
		require.Len(t, issues, 1)
		assert.Equal(t, issue.IDReadable, issues[0].IDReadable)
		assert.Equal(t, "custom column test", issues[0].Summary, "default issue fields stay in JSON output")
		names := make([]string, 0, len(issues[0].CustomFields))
		for _, f := range issues[0].CustomFields {
			names = append(names, f.Name)
		}
		assert.ElementsMatch(t, []string{priority.Name, "State"}, names)
	})

	t.Run("non-custom names the server returns still resolve as attributes", func(t *testing.T) {
		stdout, stderr, err := runCmdInDir(t, t.TempDir(), "issue", "list", "--query", query, "--fields", "idReadable,$type", "--plain")
		require.NoError(t, err, "stderr: %s", stderr)
		assert.Equal(t, issue.IDReadable+"\tIssue\n", stdout)
		assert.NotContains(t, stderr, "no custom field")
	})

	t.Run("unknown custom field warns on stderr and renders an empty column", func(t *testing.T) {
		stdout, stderr, err := runCmdInDir(t, t.TempDir(), "issue", "list", "--query", query,
			"--fields", "idReadable,NoSuchField"+suffix, "--plain")
		require.NoError(t, err, "stderr: %s", stderr)
		assert.Equal(t, issue.IDReadable+"\t\n", stdout)
		assert.Contains(t, stderr, `no custom field named "NoSuchField`+suffix+`"`)

		_, stderr, err = runCmdInDir(t, t.TempDir(), "--quiet", "issue", "list", "--query", query,
			"--fields", "idReadable,NoSuchField"+suffix, "--plain")
		require.NoError(t, err)
		assert.Empty(t, stderr)
	})
}

func TestCmd_IssueListOffset(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	for _, summary := range []string{"offset page first", "offset page second"} {
		issue, err := c.AddIssue(api.AddIssueRequest{Summary: summary, Project: &api.Project{ID: proj.ID}})
		require.NoError(t, err)
		t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })
	}

	firstRaw := runCmd(t, "issue", "list", "--query", "project: "+proj.ShortName,
		"--limit", "1", "--offset", "0", "--json")
	secondRaw := runCmd(t, "issue", "list", "--query", "project: "+proj.ShortName,
		"--limit", "1", "--offset", "1", "--json")
	var first, second []api.Issue
	require.NoError(t, json.Unmarshal([]byte(firstRaw), &first))
	require.NoError(t, json.Unmarshal([]byte(secondRaw), &second))
	require.Len(t, first, 1)
	require.Len(t, second, 1)
	assert.NotEqual(t, first[0].IDReadable, second[0].IDReadable)
}

// Cmd-layer tests

func TestCmd_IssueList(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "cmd list test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	out := runCmd(t, "issue", "list", "--query", "project: "+proj.ShortName, "--format", "json")
	assert.Contains(t, out, "cmd list test")
}

func TestCmd_IssueCreate(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	out := runCmd(t, "issue", "create",
		"--project", proj.ShortName,
		"--summary", "cmd add test",
		"--format", "json",
	)
	assert.Contains(t, out, "cmd add test")

	issues, err := c.ListIssues("project: "+proj.ShortName, 5)
	require.NoError(t, err)
	for _, iss := range issues {
		if strings.Contains(iss.Summary, "cmd add test") {
			t.Cleanup(func() { _ = c.DeleteIssue(iss.IDReadable) })
		}
	}
}

func TestCmd_IssueCreateWithFieldsProjection(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	out := runCmd(t, "issue", "create",
		"--project", proj.ShortName,
		"--summary", "cmd add fields projection test",
		"--fields", "id,idReadable,summary,project(shortName)",
		"--format", "json",
	)
	assert.Contains(t, out, "cmd add fields projection test")

	issues, err := c.ListIssues("project: "+proj.ShortName, 5)
	require.NoError(t, err)
	for _, iss := range issues {
		if strings.Contains(iss.Summary, "cmd add fields projection test") {
			t.Cleanup(func() { _ = c.DeleteIssue(iss.IDReadable) })
		}
	}
}

// An unknown field is rejected before anything is created, so no issue number is burned.
func TestCmd_IssueCreateRejectsUnknownFieldBeforeCreating(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	out, err := runCmdFail(t, "issue", "create",
		"--project", proj.ShortName,
		"--summary", "unknown field",
		"--custom-fields", `{"NonExistentField999":"Bug"}`,
	)
	require.Error(t, err)
	assert.Contains(t, out, "has no field")
	assert.Contains(t, out, "NonExistentField999")

	first := runJSON[api.Issue](t, "issue", "create", "--project", proj.ShortName, "--summary", "first", "--json")
	t.Cleanup(func() { _ = c.DeleteIssue(first.ID) })
	assert.Equal(t, proj.ShortName+"-1", first.IDReadable, "the rejected create must not have used an issue number")
}

func TestCmd_IssueCreateRejectsMalformedCustomFields(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	for _, tc := range []struct {
		name, input string
	}{
		{"empty input", ""},
		{"whitespace", " \n\t"},
		{"unclosed empty object", `{`},
		{"missing closing brace", `{"Priority":"Normal"`},
		{"trailing comma", `{"Priority":"Normal",}`},
		{"trailing garbage", `{} garbage`},
		{"multiple objects", `{} {"Priority":"Critical"}`},
		{"array", `[]`},
		{"null object", `null`},
		{"null value", `{"Priority":null}`},
		{"numeric value", `{"Priority":1}`},
		{"nested value", `{"Priority":{"name":"Normal"}}`},
	} {
		for _, fromFile := range []bool{false, true} {
			source := "inline"
			if fromFile {
				source = "file"
			}
			t.Run(tc.name+"/"+source, func(t *testing.T) {
				args := []string{"issue", "create", "--project", proj.ShortName, "--summary", "invalid custom fields", "--json"}
				if fromFile {
					path := filepath.Join(t.TempDir(), "fields.json")
					require.NoError(t, os.WriteFile(path, []byte(tc.input), 0o600))
					args = append(args, "--custom-fields-file", path)
				} else {
					args = append(args, "--custom-fields", tc.input)
				}
				out, err := runCmdFail(t, args...)
				// Clean up even if a regression accepts the input and creates an issue.
				var created api.Issue
				if json.Unmarshal([]byte(out), &created) == nil && created.ID != "" {
					t.Cleanup(func() { _ = c.DeleteIssue(created.ID) })
				}
				assert.Error(t, err, "malformed input must not create an issue: %s", out)
				assert.Contains(t, out, "parse custom-fields", "reject input before contacting the server")
			})
		}
	}
}

func TestCmd_IssueCreateWithValidCustomFields(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	for _, input := range []string{"{}\n", `{"Priority":"Critical","Type":"Bug"}`} {
		t.Run(input, func(t *testing.T) {
			issue := runJSON[api.Issue](t, "issue", "create", "--project", proj.ShortName,
				"--summary", "valid custom fields", "--custom-fields", input, "--json")
			t.Cleanup(func() { _ = c.DeleteIssue(issue.ID) })
			require.NotEmpty(t, issue.IDReadable)
			if strings.Contains(input, "Priority") {
				priority, err := c.GetCustomFieldValue(issue.IDReadable, "Priority")
				require.NoError(t, err)
				assert.Contains(t, fmt.Sprint(priority.Value), "Critical")
			}
		})
	}
}

func TestCmd_IssueView(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "cmd get test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	out := runCmd(t, "issue", "view", issue.IDReadable, "--format", "json")
	assert.Contains(t, out, issue.IDReadable)
	assert.Contains(t, out, "cmd get test")
}

// TestCmd_IssueView_StripsTerminalControls verifies that escape sequences stored in issue text never reach human-readable output.
func TestCmd_IssueView_StripsTerminalControls(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	const hostile = "\x1b]8;;https://evil.example\x1b\\click\x1b]8;;\x1b\\ \x1b[2J\x9b31m\u009b31m\rspoof\x07"
	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary:     "ctl summary " + hostile,
		Description: "ctl description\nline two " + hostile,
		Project:     &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	for _, args := range [][]string{
		{"--format", "table", "issue", "view", issue.IDReadable},
		{"--plain", "issue", "view", issue.IDReadable},
		{"--format", "table", "issue", "list", "--query", "project: " + proj.ShortName},
		{"--format", "table", "--fields", "idReadable,summary,description", "issue", "view", issue.IDReadable},
	} {
		out := runCmd(t, args...)
		assert.Contains(t, out, "ctl summary", "args: %v", args)
		assert.NotRegexp(t, "[\\x00-\\x08\\x0b-\\x1f\\x7f\\x{80}-\\x{9f}]", out, "args: %v output: %q", args, out)
	}

	// JSON output keeps the stored text intact (escaped by the encoder) for machine consumers.
	out := runCmd(t, "--json", "issue", "view", issue.IDReadable)
	assert.Contains(t, strings.ToLower(out), `\u001b`)
}

func TestCmd_IssueDelete(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "cmd delete test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)

	runCmd(t, "issue", "delete", issue.IDReadable, "--force")

	_, err = c.GetIssue(issue.IDReadable)
	assert.Error(t, err)
}

func TestCmd_IssueUpdate(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "cmd update original",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	out := runCmd(t, "issue", "update", issue.IDReadable,
		"--summary", "cmd update revised",
		"--description", "updated through cli",
		"--format", "json",
	)
	assert.Contains(t, out, "cmd update revised")

	got, err := c.GetIssue(issue.IDReadable)
	require.NoError(t, err)
	assert.Equal(t, "cmd update revised", got.Summary)
	assert.Equal(t, "updated through cli", got.Description)
}

func TestCmd_IssueCommentList_MoreThanDefaultPage(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "comment paging test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	// Without $top the server stops at 42 entries.
	const total = 45
	for i := range total {
		_, err := c.AddComment(issue.IDReadable, api.AddCommentRequest{Text: "comment " + strconv.Itoa(i)})
		require.NoError(t, err)
	}

	comments := runJSON[[]api.Comment](t, "issue", "comment", "list", issue.IDReadable, "--format", "json")
	assert.Len(t, comments, total)
}

func TestCmd_IssueCommentCRUD(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "cmd comment test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	out := runCmd(t, "issue", "comment", "create", issue.IDReadable,
		"--text", "hello from cli",
		"--format", "json",
	)
	assert.Contains(t, out, "hello from cli")

	var comment struct {
		ID      string `json:"id"`
		Text    string `json:"text"`
		Deleted *bool  `json:"deleted"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &comment))
	require.NotEmpty(t, comment.ID)
	require.NotNil(t, comment.Deleted, "comment JSON must include deleted=false")
	assert.False(t, *comment.Deleted)

	out = runCmd(t, "issue", "comment", "list", issue.IDReadable, "--format", "json")
	assert.Contains(t, out, comment.ID)
	assert.Contains(t, out, "hello from cli")

	out = runCmd(t, "issue", "comment", "update", issue.IDReadable, comment.ID,
		"--text", "updated from cli",
		"--format", "json",
	)
	assert.Contains(t, out, "updated from cli")

	comments, err := c.ListComments(issue.IDReadable)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "updated from cli", comments[0].Text)

	out = runCmd(t, "issue", "comment", "delete", issue.IDReadable, comment.ID, "--force")
	assert.Contains(t, out, "deleted comment "+comment.ID)

	// UI-style deletion retains a recoverable comment. JSON must make its state
	// visible even when the server still returns the original comment text.
	hidden := runJSON[[]map[string]any](t, "issue", "comment", "list", issue.IDReadable, "--json")
	assert.Empty(t, hidden, "comment list leaves deleted comments out by default")

	deleted := runJSON[[]map[string]any](t, "issue", "comment", "list", issue.IDReadable, "--include-deleted", "--json")
	require.Len(t, deleted, 1)
	assert.Equal(t, comment.ID, deleted[0]["id"])
	assert.Equal(t, true, deleted[0]["deleted"])
	assert.Equal(t, "IssueComment", deleted[0]["$type"], "preserve API fields beyond the typed comment model")

	viewed := runJSON[api.Comment](t, "issue", "comment", "view", issue.IDReadable, comment.ID, "--json")
	assert.True(t, viewed.Deleted)

	for _, format := range []string{"table", "plain"} {
		for _, action := range []string{"list", "view"} {
			t.Run("deleted comment "+action+" "+format, func(t *testing.T) {
				args := []string{"issue", "comment", action, issue.IDReadable}
				if action == "view" {
					args = append(args, comment.ID)
				} else {
					args = append(args, "--include-deleted")
				}
				args = append(args, "--format", format)
				out := runCmd(t, args...)
				assert.Contains(t, out, "[deleted]")
				assert.NotContains(t, out, "updated from cli")

				out = runCmd(t, append(args, "--fields", "id,deleted,text")...)
				assert.Contains(t, out, "[deleted]")
				assert.NotContains(t, out, "updated from cli")
			})
		}
	}

	activeText := "active comment stays visible"
	active, err := c.AddComment(issue.IDReadable, api.AddCommentRequest{Text: activeText})
	require.NoError(t, err)
	// Include deleted comments in pinnedComments so the narrow pinned projection
	// has the same deletion authority requirement as the comments collection.
	require.NoError(t, c.PinComment(issue.IDReadable, comment.ID, true))

	const commentProjection = "comments(id,deleted,text,textPreview)"
	var server struct {
		Comments json.RawMessage `json:"comments"`
	}
	require.NoError(t, c.Get("/api/issues/"+issue.IDReadable+"?fields="+commentProjection, &server))
	fromCLI := runJSON[struct {
		Comments json.RawMessage `json:"comments"`
	}](t, "issue", "view", issue.IDReadable, "--fields", commentProjection, "--json")
	assert.JSONEq(t, string(server.Comments), string(fromCLI.Comments), "JSON must preserve the server's actual comment data")

	for _, format := range []string{"table", "plain"} {
		for _, action := range []string{"view", "list"} {
			for _, projection := range []string{commentProjection, "comments(text)", "pinnedComments(text)"} {
				t.Run("nested deleted comment "+action+" "+format+" "+projection, func(t *testing.T) {
					args := []string{"issue", action}
					if action == "view" {
						args = append(args, issue.IDReadable)
					} else {
						args = append(args, "--query", "project: "+proj.ShortName)
					}
					out := runCmd(t, append(args, "--fields", projection, "--format", format)...)
					assert.Contains(t, out, "[deleted]")
					assert.NotContains(t, out, "updated from cli")
					if projection == commentProjection {
						assert.Contains(t, out, comment.ID)
						assert.Contains(t, out, active.ID)
						assert.Contains(t, out, activeText)
					}
				})
			}
		}
	}
}

func TestCmd_IssueWorkitemCreate(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	out := runCmd(t, "project", "timetrack", "set", proj.ShortName, "--enabled")
	assert.Contains(t, out, "updated time tracking for project "+proj.ShortName)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "cmd workitem test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	out = runCmd(t, "issue", "workitem", "create", issue.IDReadable,
		"--minutes", "90",
		"--text", "design work",
		"--format", "json",
	)
	assert.Contains(t, out, "design work")

	var item struct {
		ID       string `json:"id"`
		Text     string `json:"text"`
		Duration struct {
			Minutes int `json:"minutes"`
		} `json:"duration"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &item))
	assert.NotEmpty(t, item.ID)
	assert.Equal(t, "design work", item.Text)
	assert.Equal(t, 90, item.Duration.Minutes)
}

func TestCmd_IssueDelete_Confirm(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "confirm delete test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)

	out := runCmdWithInput(t, "y\n", "issue", "delete", issue.IDReadable)
	assert.Contains(t, out, "deleted issue "+issue.IDReadable)

	_, err = c.GetIssue(issue.IDReadable)
	assert.Error(t, err, "issue should be deleted after confirming with y")
}

func TestCmd_IssueDelete_Cancel(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "cancel delete test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	out := runCmdWithInput(t, "n\n", "issue", "delete", issue.IDReadable)
	assert.Contains(t, out, "Canceled.")

	got, err := c.GetIssue(issue.IDReadable)
	require.NoError(t, err, "issue should still exist after canceling")
	assert.Equal(t, issue.IDReadable, got.IDReadable)
}

func TestCmd_IssueLinkAdd(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	src, err := c.AddIssue(api.AddIssueRequest{Summary: "link source", Project: &api.Project{ID: proj.ID}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(src.IDReadable) })

	dst, err := c.AddIssue(api.AddIssueRequest{Summary: "link target", Project: &api.Project{ID: proj.ID}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(dst.IDReadable) })

	types, err := c.GetIssueLinkTypes()
	require.NoError(t, err)
	require.NotEmpty(t, types, "YouTrack instance must have at least one link type")

	runCmd(t, "issue", "link", "add", src.IDReadable,
		"--target", dst.IDReadable,
		"--type", types[0].ID,
	)

	links, err := c.GetLinkedIssues(src.IDReadable)
	require.NoError(t, err)
	assert.NotEmpty(t, links, "expected at least one link after issue link add")
}

func TestCmd_IssueCountNonNegative(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{Summary: "count test", Project: &api.Project{ID: proj.ID}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	out := runCmd(t, "issue", "count", "--query", "project: "+proj.ShortName)
	count, err := strconv.Atoi(strings.TrimSpace(out))
	require.NoError(t, err, "output should be a plain integer, got: %q", out)
	// Assert >= 0, not >= 1: this test verifies the -1 sentinel is never returned,
	// not that the just-created issue has been indexed yet (indexing may lag).
	assert.GreaterOrEqual(t, count, 0, "count must never be -1 after the retry loop")
}

func TestCmd_IssueLinkList_ShowsID(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	src, err := c.AddIssue(api.AddIssueRequest{Summary: "link src", Project: &api.Project{ID: proj.ID}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(src.IDReadable) })

	dst, err := c.AddIssue(api.AddIssueRequest{Summary: "link dst", Project: &api.Project{ID: proj.ID}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(dst.IDReadable) })

	types, err := c.GetIssueLinkTypes()
	require.NoError(t, err)
	require.NotEmpty(t, types)

	err = c.LinkIssues(src.IDReadable, dst.IDReadable, types[0].ID)
	require.NoError(t, err)

	out := runCmd(t, "issue", "link", "list", src.IDReadable, "--format", "json")
	assert.Contains(t, out, `"id"`)
}

func TestCmd_UserDeleteSelfRejected(t *testing.T) {
	// F-17 regression: deleting the authenticated user must be rejected
	c := newTestClient(t)
	me, err := c.GetCurrentUser()
	require.NoError(t, err)

	out := runCmdExpectError(t, "user", "delete", me.Login, "--successor", me.Login, "--force")
	_ = out // runCmdExpectError asserts non-zero exit; errors render on stderr
}

func TestCmd_IssueCommentView(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "comment get test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	comment, err := c.AddComment(issue.IDReadable, api.AddCommentRequest{Text: "get me"})
	require.NoError(t, err)
	require.NotEmpty(t, comment.ID)

	out := runCmd(t, "issue", "comment", "view", issue.IDReadable, comment.ID, "--format", "json")
	assert.Contains(t, out, comment.ID)
	assert.Contains(t, out, "get me")
}

func TestCmd_IssueCommentPin(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "comment pin test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	comment, err := c.AddComment(issue.IDReadable, api.AddCommentRequest{Text: "pin me"})
	require.NoError(t, err)
	require.NotEmpty(t, comment.ID)

	out := runCmd(t, "issue", "comment", "pin", issue.IDReadable, comment.ID)
	assert.Contains(t, out, "pinned")

	other, err := c.AddComment(issue.IDReadable, api.AddCommentRequest{Text: "not pinned"})
	require.NoError(t, err)
	listed := runJSON[[]api.Comment](t, "issue", "comment", "list", issue.IDReadable, "--json")
	pinned := map[string]bool{}
	for _, cm := range listed {
		pinned[cm.ID] = cm.Pinned
	}
	assert.Equal(t, map[string]bool{comment.ID: true, other.ID: false}, pinned)
	raw := runJSON[[]map[string]any](t, "issue", "comment", "list", issue.IDReadable, "--json")
	for _, cm := range raw {
		assert.Contains(t, cm, "pinned", "comment JSON must include pinned=false too")
	}

	out = runCmd(t, "issue", "comment", "list", issue.IDReadable, "--fields", "id,pinned", "--plain")
	assert.Contains(t, out, comment.ID+"\ttrue")
	assert.Contains(t, out, other.ID+"\tfalse")
}

func TestCmd_IssueCommentVisibility(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "comment visibility test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	comment, err := c.AddComment(issue.IDReadable, api.AddCommentRequest{Text: "visibility me"})
	require.NoError(t, err)
	require.NotEmpty(t, comment.ID)

	// Call with no groups/users — sets an empty LimitedVisibility which is valid
	out := runCmd(t, "issue", "comment", "visibility", issue.IDReadable, comment.ID)
	assert.Contains(t, out, "updated visibility for comment "+comment.ID)

	me, err := c.GetCurrentUser()
	require.NoError(t, err)
	runCmd(t, "issue", "comment", "visibility", issue.IDReadable, comment.ID, "--users", me.ID)
	listed := runJSON[[]api.Comment](t, "issue", "comment", "list", issue.IDReadable, "--json")
	require.Len(t, listed, 1)
	require.NotNil(t, listed[0].Visibility)
	assert.Equal(t, "LimitedVisibility", listed[0].Visibility.Type)
	require.Len(t, listed[0].Visibility.PermittedUsers, 1)
	assert.Equal(t, me.Login, listed[0].Visibility.PermittedUsers[0].Login)
}

func TestCmd_IssueLinkRemove(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	src, err := c.AddIssue(api.AddIssueRequest{Summary: "link remove src", Project: &api.Project{ID: proj.ID}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(src.IDReadable) })

	dst, err := c.AddIssue(api.AddIssueRequest{Summary: "link remove dst", Project: &api.Project{ID: proj.ID}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(dst.IDReadable) })

	types, err := c.GetIssueLinkTypes()
	require.NoError(t, err)
	require.NotEmpty(t, types)

	err = c.LinkIssues(src.IDReadable, dst.IDReadable, types[0].ID)
	require.NoError(t, err)

	links, err := c.GetLinkedIssues(src.IDReadable)
	require.NoError(t, err)
	require.NotEmpty(t, links, "expected a link after LinkIssues")

	linkID := links[0].ID

	runCmd(t, "issue", "link", "remove", src.IDReadable, linkID, "--force")

	links, err = c.GetLinkedIssues(src.IDReadable)
	require.NoError(t, err)
	// After removal the link slot still exists but should have no issues in it
	for _, l := range links {
		if l.ID == linkID {
			assert.Empty(t, l.Issues, "link %s should have no issues after removal", linkID)
		}
	}
}

func TestCmd_IssueLinkTypeList(t *testing.T) {
	out := runCmd(t, "issue", "link", "type", "list", "--format", "json")
	assert.Contains(t, out, "[")
}

func TestCmd_IssueWorkitemView(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	runCmd(t, "project", "timetrack", "set", proj.ShortName, "--enabled")

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "workitem get test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	item, err := c.AddWorkItem(issue.IDReadable, 30, "get this item", "")
	require.NoError(t, err)
	require.NotEmpty(t, item.ID)

	out := runCmd(t, "issue", "workitem", "view", issue.IDReadable, item.ID, "--format", "json")
	assert.Contains(t, out, item.ID)
}

func TestCmd_IssueWorkitemUpdate(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	runCmd(t, "project", "timetrack", "set", proj.ShortName, "--enabled")

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "workitem update test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	item, err := c.AddWorkItem(issue.IDReadable, 30, "original text", "")
	require.NoError(t, err)
	require.NotEmpty(t, item.ID)

	out := runCmd(t, "issue", "workitem", "update", issue.IDReadable, item.ID,
		"--minutes", "90",
		"--text", "updated text",
		"--format", "json",
	)
	assert.Contains(t, out, "updated text")

	got, err := c.GetWorkItem(issue.IDReadable, item.ID)
	require.NoError(t, err)
	require.NotNil(t, got.Duration)
	assert.Equal(t, 90, got.Duration.Minutes)
}

func TestCmd_IssueWorkitemListAndDelete(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	runCmd(t, "project", "timetrack", "set", proj.ShortName, "--enabled")
	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "workitem lifecycle test", Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })
	item, err := c.AddWorkItem(issue.IDReadable, 15, "delete this item", "")
	require.NoError(t, err)

	listOut := runCmd(t, "issue", "workitem", "list", issue.IDReadable, "--json")
	assert.Contains(t, listOut, item.ID)
	delOut := runCmd(t, "issue", "workitem", "delete", issue.IDReadable, item.ID, "--force")
	assert.Contains(t, delOut, "deleted work item "+item.ID)
	items, err := c.ListWorkItems(issue.IDReadable)
	require.NoError(t, err)
	for _, current := range items {
		assert.NotEqual(t, item.ID, current.ID)
	}
}

func TestCmd_IssueAttachmentCRUD(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "attachment crud test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	// Write a temp file to attach
	tmpFile := filepath.Join(t.TempDir(), "testattach.txt")
	require.NoError(t, os.WriteFile(tmpFile, []byte("hello attachment"), 0644))

	// Add attachment
	addOut := runCmd(t, "issue", "attachment", "add", issue.IDReadable, "--file", tmpFile, "--format", "json")
	var attach struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	require.NoError(t, json.Unmarshal([]byte(addOut), &attach))
	require.NotEmpty(t, attach.ID, "attachment ID should not be empty")

	// List attachments
	listOut := runCmd(t, "issue", "attachment", "list", issue.IDReadable, "--format", "json")
	assert.Contains(t, listOut, attach.ID)

	// Delete attachment
	delOut := runCmd(t, "issue", "attachment", "delete", issue.IDReadable, attach.ID, "--force")
	assert.Contains(t, delOut, "deleted attachment "+attach.ID)

	// Verify gone
	attachments, err := c.GetIssueAttachments(issue.IDReadable)
	require.NoError(t, err)
	for _, a := range attachments {
		assert.NotEqual(t, attach.ID, a.ID, "attachment should be deleted")
	}
}

func TestCmd_IssueTag(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "tag test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	// Get tags available for current user
	me, err := c.GetCurrentUser()
	require.NoError(t, err)
	tags, err := c.ListUserTags(me.ID)
	require.NoError(t, err)
	if len(tags) == 0 {
		t.Skip("no user tags available; skipping tag test")
	}
	tagID := tags[0].ID

	// Add tag to issue
	addOut := runCmd(t, "issue", "tag", "add", issue.IDReadable, tagID)
	assert.Contains(t, addOut, "tagged issue "+issue.IDReadable)

	// List tags on issue
	listOut := runCmd(t, "issue", "tag", "list", issue.IDReadable, "--format", "json")
	assert.Contains(t, listOut, tagID)

	removeOut := runCmd(t, "issue", "tag", "remove", issue.IDReadable, tagID)
	assert.Contains(t, removeOut, "removed tag "+tagID)
	tagsAfter, err := c.ListIssueTags(issue.IDReadable)
	require.NoError(t, err)
	for _, current := range tagsAfter {
		assert.NotEqual(t, tagID, current.ID)
	}
}

func TestCmd_IssueFieldView(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	field := createProjectEnumField(t, c, proj, api.FieldTypeEnumSingle, "Normal")

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "field get test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })
	awaitIssueFields(t, c, issue.IDReadable, field.Name)
	require.NoError(t, c.SetCustomField(issue.IDReadable, field.Name, "Normal", ""))

	got := runJSON[api.IssueField](t, "issue", "field", "view", issue.IDReadable, field.Name, "--json")
	assert.Equal(t, field.Name, got.Name)
	value, ok := got.Value.(map[string]any)
	require.True(t, ok, "single-value field returned %T", got.Value)
	assert.Equal(t, "Normal", value["name"])
}

func TestCmd_IssueFieldSet(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	field := createProjectEnumField(t, c, proj, api.FieldTypeEnumSingle, "Normal", "Critical")

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "field set test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })
	awaitIssueFields(t, c, issue.IDReadable, field.Name)
	require.NoError(t, c.SetCustomField(issue.IDReadable, field.Name, "Normal", ""))

	out := runCmd(t, "issue", "field", "set", issue.IDReadable, field.Name, "Critical")
	assert.Contains(t, out, "updated field "+field.Name)

	got, err := c.GetCustomFieldValue(issue.IDReadable, field.Name)
	require.NoError(t, err)
	value, ok := got.Value.(map[string]any)
	require.True(t, ok, "single-value field returned %T", got.Value)
	assert.Equal(t, "Critical", value["name"])
}

func TestCmd_IssueFieldSet_UserBeyondFirstTen(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	me, err := c.GetCurrentUser()
	require.NoError(t, err)

	// All eleven users match the assignee's login through their full names.
	assigneeLogin := fmt.Sprintf("field-user-%s-10", proj.ShortName)
	var users []api.User
	for i := range 11 {
		login := fmt.Sprintf("field-user-%s-%02d", proj.ShortName, i)
		user, err := c.AddUser(&api.AddUserRequest{
			Login: login, FullName: fmt.Sprintf("Field user %02d %s", i, assigneeLogin),
			Email: login + "@example.com", Password: "Test1234!",
		})
		require.NoError(t, err)
		t.Cleanup(func() { _ = c.DeleteUser(user.ID, me.ID) })
		users = append(users, *user)
	}
	assignee := users[len(users)-1]
	firstPage := runJSON[[]api.User](t, "user", "list", "--query", assignee.Login, "--limit", "10", "--json")
	require.Len(t, firstPage, 10)
	for _, user := range firstPage {
		require.NotEqual(t, assignee.ID, user.ID, "fixture assignee must be outside the first ten search results")
	}
	_, err = c.AddUserToProjectTeam(proj.ShortName, assignee.ID)
	require.NoError(t, err)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "user field lookup test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	out := runCmd(t, "issue", "field", "set", issue.IDReadable, "Assignee", assignee.Login, "--type", "user")
	assert.Contains(t, out, "updated field Assignee")
	assertIssueAssignee(t, c, issue.IDReadable, assignee.Login)

	missingLogin := "field-user-" + proj.ShortName
	errOutput := runCmdExpectError(t, "issue", "field", "set", issue.IDReadable, "Assignee", missingLogin, "--type", "user")
	assert.Contains(t, errOutput, "user '"+missingLogin+"' not found")
	assertIssueAssignee(t, c, issue.IDReadable, assignee.Login)
}

func TestCmd_IssueFieldSet_MultiValue(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	field := createProjectEnumField(t, c, proj, api.FieldTypeEnumMulti, "Alpha")

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "multi-value field set test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })
	awaitIssueFields(t, c, issue.IDReadable, field.Name)

	out := runCmd(t, "issue", "field", "set", issue.IDReadable, field.Name, "Alpha")
	assert.Contains(t, out, "updated field "+field.Name)

	got, err := c.GetCustomFieldValue(issue.IDReadable, field.Name)
	require.NoError(t, err)
	values, ok := got.Value.([]any)
	require.True(t, ok, "multi-value field returned %T", got.Value)
	require.Len(t, values, 1)
	value, ok := values[0].(map[string]any)
	require.True(t, ok, "multi-value field element returned %T", values[0])
	assert.Equal(t, "Alpha", value["name"])
}

// Newly attached project fields can reach the issue projection after creation.
// Observe fixture readiness before seeding values or exercising a CLI write.
func awaitIssueFields(t *testing.T, c *api.Client, issueID string, names ...string) {
	t.Helper()
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		var fields []api.IssueField
		err := c.Get("/api/issues/"+url.PathEscape(issueID)+"/customFields?fields=id,name&$top=500", &fields)
		if !assert.NoError(collect, err, "reading issue %s fixture fields", issueID) {
			return
		}
		byName := make(map[string]string, len(fields))
		for _, field := range fields {
			byName[field.Name] = field.ID
		}
		for _, name := range names {
			assert.NotEmpty(collect, byName[name], "issue %s fixture field %q is not ready; visible fields: %v", issueID, name, byName)
		}
	}, 10*time.Second, 20*time.Millisecond, "issue %s fixture fields did not become ready", issueID)
}

// Field command tests own their field and bundle instead of depending on the
// server's default project template to attach Priority and its allowed values.
func createProjectEnumField(t *testing.T, c *api.Client, project *api.Project, fieldType string, values ...string) *api.CustomField {
	t.Helper()
	name := "FixtureEnum-" + project.ShortName
	bundle, err := c.CreateBundle("enum", name)
	require.NoError(t, err)
	t.Cleanup(func() {
		// Removing the final project field can also remove its unused bundle.
		// A confirmed 404 is already clean; every other delete error is a failure.
		err := c.DeleteBundle("enum", bundle.ID)
		var apiErr *api.APIError
		if err != nil && (!errors.As(err, &apiErr) || apiErr.StatusCode != 404) {
			require.NoError(t, err)
		}
		err = c.Get("/api/admin/customFieldSettings/bundles/enum/"+url.PathEscape(bundle.ID)+"?fields=id", nil)
		require.ErrorAs(t, err, &apiErr, "fixture bundle %s still exists", bundle.ID)
		require.Equal(t, 404, apiErr.StatusCode, "reading deleted fixture bundle: %v", err)
	})
	for _, value := range values {
		require.NoError(t, c.AddBundleValue("enum", bundle.ID, value))
	}

	field, err := c.AddCustomField(name, fieldType)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, c.DeleteCustomField(field.ID)) })
	var attached struct {
		ID string `json:"id"`
	}
	require.NoError(t, c.Post("/api/admin/projects/"+url.PathEscape(project.ID)+"/customFields?fields=id", map[string]any{
		"$type":      "EnumProjectCustomField",
		"field":      map[string]any{"id": field.ID},
		"bundle":     map[string]any{"id": bundle.ID, "$type": "EnumBundle"},
		"canBeEmpty": true,
		"isPublic":   true,
	}, &attached))
	require.NotEmpty(t, attached.ID)
	t.Cleanup(func() {
		require.NoError(t, c.RemoveProjectCustomField(project.ShortName, attached.ID))
		// Field removal schedules a server job. Wait for the field to be gone
		// before the later cleanup deletes the field prototype and its bundle.
		require.EventuallyWithT(t, func(collect *assert.CollectT) {
			var status struct {
				HasRunningJob bool `json:"hasRunningJob"`
			}
			err := c.Get("/api/admin/projects/"+url.PathEscape(project.ID)+"/customFields/"+url.PathEscape(attached.ID)+"?fields=hasRunningJob", &status)
			if err == nil {
				assert.Fail(collect, fmt.Sprintf("project field %s still exists (hasRunningJob=%t)", attached.ID, status.HasRunningJob))
				return
			}
			var apiErr *api.APIError
			if assert.True(collect, errors.As(err, &apiErr), "reading deleted project field: %v", err) {
				assert.Equal(collect, 404, apiErr.StatusCode, "reading deleted project field: %v", err)
			}
		}, 10*time.Second, 20*time.Millisecond, "project field %s deletion did not finish", attached.ID)
	})
	return field
}

func attachDefaultProjectFields(t *testing.T, c *api.Client, project *api.Project, names ...string) {
	t.Helper()
	prototypes, err := c.ListCustomFields()
	require.NoError(t, err)
	attached, err := c.GetProjectCustomFields(project.ShortName)
	require.NoError(t, err)
	for _, name := range names {
		var fieldID string
		for _, prototype := range prototypes {
			if prototype.Name == name {
				fieldID = prototype.ID
				break
			}
		}
		require.NotEmpty(t, fieldID, "missing default custom-field prototype %q", name)
		var exists bool
		for _, field := range attached {
			if field.Field != nil && field.Field.ID == fieldID {
				exists = true
				break
			}
		}
		if !exists {
			require.NoError(t, c.AddProjectCustomField(project.ShortName, fieldID))
		}
	}
}

func TestCmd_IssueFieldSet_ValuesAndClear(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)
	attachDefaultProjectFields(t, c, proj, "Priority", "Assignee")

	field := createProjectEnumField(t, c, proj, api.FieldTypeEnumMulti, "Alpha", "Platform. Auth & Licensing")
	name := field.Name

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "field set values and clear test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })
	awaitIssueFields(t, c, issue.IDReadable, name)

	valueNames := func() []string {
		got, err := c.GetCustomFieldValue(issue.IDReadable, name)
		require.NoError(t, err)
		raw, err := json.Marshal(got.Value)
		require.NoError(t, err)
		var values []struct {
			Name string `json:"name"`
		}
		require.NoError(t, json.Unmarshal(raw, &values), "value of %s: %s", name, raw)
		names := make([]string, len(values))
		for i, v := range values {
			names[i] = v.Name
		}
		return names
	}

	out := runCmd(t, "issue", "field", "set", issue.IDReadable, name, "--value", "Alpha", "--value", "Platform. Auth & Licensing")
	assert.Contains(t, out, "updated field "+name)
	assert.ElementsMatch(t, []string{"Alpha", "Platform. Auth & Licensing"}, valueNames())

	issueField, err := c.GetCustomFieldValue(issue.IDReadable, name)
	require.NoError(t, err)
	runCmd(t, "issue", "field", "set", issue.IDReadable, issueField.ID, "--value", "Alpha")
	assert.Equal(t, []string{"Alpha"}, valueNames(), "--value replaces the current values using an opaque field ID")

	out = runCmd(t, "issue", "field", "set", issue.IDReadable, issueField.ID, "--clear")
	assert.Contains(t, out, "cleared field "+issueField.ID)
	assert.Empty(t, valueNames())

	errOut := runCmdExpectError(t, "issue", "field", "set", issue.IDReadable, "Priority", "--value", "Major", "--value", "Minor")
	assert.Contains(t, errOut, "holds a single value")
	errOut = runCmdExpectError(t, "issue", "field", "set", issue.IDReadable, name, "Alpha", "--clear")
	assert.Contains(t, errOut, "pass exactly one of")
	errOut = runCmdExpectError(t, "issue", "field", "set", issue.IDReadable, name)
	assert.Contains(t, errOut, "pass exactly one of")

	me, err := c.GetCurrentUser()
	require.NoError(t, err)
	runCmd(t, "issue", "field", "set", issue.IDReadable, "Assignee", me.Login, "--type", "user")
	assertIssueAssignee(t, c, issue.IDReadable, me.Login)
	out = runCmd(t, "issue", "field", "set", issue.IDReadable, "Assignee", "--clear")
	assert.Contains(t, out, "cleared field Assignee")
	got, err := c.GetCustomFieldValue(issue.IDReadable, "Assignee")
	require.NoError(t, err)
	assert.Nil(t, got.Value)
}

func TestCmd_IssueVisibility(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "visibility test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	// Call with no groups/users — sets empty LimitedVisibility
	out := runCmd(t, "issue", "visibility", issue.IDReadable)
	assert.Contains(t, out, "updated visibility for "+issue.IDReadable)
}

func TestCmd_IssueReactionAddDelete(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "reaction test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	comment, err := c.AddComment(issue.IDReadable, api.AddCommentRequest{Text: "react to me"})
	require.NoError(t, err)
	require.NotEmpty(t, comment.ID)

	// Add reaction via CLI
	addOut := runCmd(t, "issue", "reaction", "add", issue.IDReadable, comment.ID, "👍")
	assert.Contains(t, addOut, "added reaction 👍")

	// List reactions via API to get the ID
	reactions, err := c.ListIssueCommentReactions(issue.IDReadable, comment.ID)
	require.NoError(t, err)
	require.NotEmpty(t, reactions, "expected at least one reaction after adding")
	reactionID := reactions[0].ID
	require.NotEmpty(t, reactionID)

	// Delete reaction via CLI
	delOut := runCmd(t, "issue", "reaction", "delete", issue.IDReadable, comment.ID, reactionID, "--force")
	assert.Contains(t, delOut, "deleted reaction "+reactionID)

	// Verify gone
	reactions, err = c.ListIssueCommentReactions(issue.IDReadable, comment.ID)
	require.NoError(t, err)
	assert.Empty(t, reactions, "reaction should be gone after deletion")
}

func TestCmd_IssueActivityList(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "activity list test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	_, err = c.AddComment(issue.IDReadable, api.AddCommentRequest{Text: "activity comment"})
	require.NoError(t, err)

	out := runCmd(t, "issue", "activity", "list", issue.IDReadable, "--format", "json")
	assert.Contains(t, out, "[")
}

func TestCmd_IssueCommand(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{
		Summary: "command test",
		Project: &api.Project{ID: proj.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	// On a freshly created project the default State bundle may not be attached
	// yet, so the command parser transiently rejects "In Progress"; poll until it applies.
	var out string
	require.Eventually(t, func() bool {
		var err error
		out, err = tryCmdNoFail(t, "issue", "command", issue.IDReadable, "State In Progress")
		return err == nil
	}, 15*time.Second, 500*time.Millisecond, "State command should apply once the project's State bundle is provisioned")
	assert.Contains(t, out, "command applied to "+issue.IDReadable)

	// Verify state changed via API
	got, err := c.GetIssueDetail(issue.IDReadable)
	require.NoError(t, err)
	stateFound := false
	for _, f := range got.CustomFields {
		if f.Name == "State" {
			if v, ok := f.Value.(map[string]any); ok {
				if name, _ := v["name"].(string); strings.Contains(name, "Progress") {
					stateFound = true
				}
			}
		}
	}
	assert.True(t, stateFound, "State field should show In Progress after command")
}

// Collections return at most 42 items without an explicit $top; comment list must not stop there.
func TestCmd_IssueCommentList_BeyondDefaultPage(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{Summary: "many comments", Project: &api.Project{ID: proj.ID}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	const total = 45
	for i := range total {
		_, err := c.AddComment(issue.IDReadable, api.AddCommentRequest{Text: "comment " + strconv.Itoa(i)})
		require.NoError(t, err)
	}

	comments := runJSON[[]api.Comment](t, "issue", "comment", "list", issue.IDReadable, "--format", "json")
	assert.Len(t, comments, total)
}

// issue count and command suggest POST to endpoints that change nothing, so read-only mode must not block them.
func TestCmd_ReadOnlyAllowsQueryEndpoints(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{Summary: "read-only query test", Project: &api.Project{ID: proj.ID}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	roEnv := []string{"HOME=" + t.TempDir(), "YOUTRACK_RO=1"}

	out := runCmdWithEnv(t, "", roEnv, "issue", "count", "--query", "project: "+proj.ShortName)
	count, err := strconv.Atoi(strings.TrimSpace(out))
	require.NoError(t, err, "output should be a plain integer, got: %q", out)
	assert.GreaterOrEqual(t, count, 0)

	out = runCmdWithEnv(t, "", roEnv, "issue", "command", "suggest", issue.IDReadable, "State ", "--format", "json")
	assert.Contains(t, out, "commands")

	cmd := exec.Command(binaryPath, "issue", "comment", "create", issue.IDReadable, "--text", "must not be posted")
	cmd.Env = append(os.Environ(), roEnv...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	require.Error(t, cmd.Run(), "writes must stay blocked in read-only mode")
	assert.Contains(t, stderr.String(), "read-only mode is enabled")

	comments, err := c.ListComments(issue.IDReadable)
	require.NoError(t, err)
	assert.Empty(t, comments)
}

// Readable IDs go to /api/commands as idReadable; database IDs are still resolved first.
func TestCmd_IssueCommand_ByDatabaseIDAndUnknownID(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{Summary: "command by db id", Project: &api.Project{ID: proj.ID}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })
	tag := "cmdtag" + strconv.FormatInt(time.Now().UnixNano()%1_000_000, 10)
	t.Cleanup(func() {
		tags, _ := c.ListTags(tag, 100)
		for _, tg := range tags {
			if tg.Name == tag {
				_ = c.DeleteTag(tg.ID)
			}
		}
	})

	runCmd(t, "issue", "command", issue.ID, "tag "+tag)
	tags, err := c.ListIssueTags(issue.IDReadable)
	require.NoError(t, err)
	var names []string
	for _, tg := range tags {
		names = append(names, tg.Name)
	}
	assert.Contains(t, names, tag, "command addressed by database ID should apply")

	stderr := runCmdExpectError(t, "issue", "command", proj.ShortName+"-999999", "tag "+tag)
	assert.NotEmpty(t, strings.TrimSpace(stderr), "unknown issue should produce an error message")
}

func TestCmd_IssueCommand_NoNewTags(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{Summary: "no new tags", Project: &api.Project{ID: proj.ID}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	var profile struct {
		Locale map[string]any `json:"locale"`
	}
	require.NoError(t, c.Get("/api/users/me/profiles/general?fields=locale(id)", &profile))
	t.Cleanup(func() { require.NoError(t, c.SetUserGeneralProfile("me", map[string]any{"locale": profile.Locale})) })
	require.NoError(t, c.SetUserLocale("me", "de_DE"))
	rejected := runCmdExpectError(t, "issue", "command", issue.IDReadable, "tag must-not-be-created", "--no-new-tags")
	assert.Contains(t, rejected, "requires an English YouTrack profile")
	gotTags, err := c.ListIssueTags(issue.IDReadable)
	require.NoError(t, err)
	assert.Empty(t, gotTags)
	require.NoError(t, c.SetUserLocale("me", "en_US"))
	invalid := runCmdExpectError(t, "issue", "command", issue.IDReadable, "State {In Progress}", "--no-new-tags")
	assert.Contains(t, invalid, "cannot verify command preview")

	suffix := strconv.FormatInt(time.Now().UnixNano()%1_000_000, 10)
	existing, missing := "keeptag"+suffix, "newtag"+suffix
	tag, err := c.AddTag(api.AddTagRequest{Name: existing})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = c.DeleteTag(tag.ID)
		tags, _ := c.ListTags(missing, 100)
		for _, tg := range tags {
			if tg.Name == missing {
				_ = c.DeleteTag(tg.ID)
			}
		}
	})
	issueTags := func() []string {
		tags, err := c.ListIssueTags(issue.IDReadable)
		require.NoError(t, err)
		var names []string
		for _, tg := range tags {
			names = append(names, tg.Name)
		}
		return names
	}

	stderr := runCmdExpectError(t, "issue", "command", issue.IDReadable, "tag "+existing+" tag "+missing, "--no-new-tags")
	assert.Contains(t, stderr, missing)
	assert.Contains(t, stderr, "nothing was applied")
	assert.Empty(t, issueTags(), "a refused command must not apply any part")
	tags, err := c.ListTags(missing, 100)
	require.NoError(t, err)
	for _, tg := range tags {
		assert.NotEqual(t, missing, tg.Name, "--no-new-tags must not create the tag")
	}

	out := runCmd(t, "issue", "command", issue.IDReadable, "tag "+existing, "--no-new-tags")
	assert.Contains(t, out, "command applied to "+issue.IDReadable)
	assert.Equal(t, []string{existing}, issueTags())
}

// File names with quotes and line breaks must reach YouTrack without corrupting the multipart part headers.
func TestCmd_IssueAttachmentAdd_UnusualFileNames(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	issue, err := c.AddIssue(api.AddIssueRequest{Summary: "quoted attachment name", Project: &api.Project{ID: proj.ID}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.DeleteIssue(issue.IDReadable) })

	for _, tc := range []struct{ name, file, want string }{
		{"quote is preserved", `say "hi".txt`, `say "hi".txt`},
		{"line breaks cannot end the part headers early", "a\r\n\r\nsmuggled.txt", "a%0D%0A%0D%0Asmuggled.txt"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmpFile := filepath.Join(t.TempDir(), tc.file)
			require.NoError(t, os.WriteFile(tmpFile, []byte("content"), 0o600))

			attach := runJSON[api.IssueAttachment](t, "issue", "attachment", "add", issue.IDReadable, "--file", tmpFile, "--format", "json")
			assert.Equal(t, tc.want, attach.Name)

			// The stored size proves no header text leaked into the file content.
			var stored struct {
				Size int `json:"size"`
			}
			require.NoError(t, c.Get(fmt.Sprintf("/api/issues/%s/attachments/%s?fields=size", issue.IDReadable, attach.ID), &stored))
			assert.Equal(t, len("content"), stored.Size)
		})
	}
}

// Two children created with --parent must both end up as subtasks of the same parent.
func TestCmd_IssueCreateWithParent(t *testing.T) {
	c := newTestClient(t)
	proj := createTempProject(t, c)

	create := func(args ...string) string {
		var iss api.Issue
		out := runCmd(t, append([]string{"issue", "create", "--project", proj.ShortName, "--format", "json"}, args...)...)
		require.NoError(t, json.Unmarshal([]byte(out), &iss), "output: %s", out)
		t.Cleanup(func() { _ = c.DeleteIssue(iss.IDReadable) })
		return iss.IDReadable
	}
	parent := create("--summary", "parent test")
	first := create("--summary", "child one", "--parent", parent)
	second := create("--summary", "child two", "--parent", parent)

	links, err := c.GetLinkedIssues(parent)
	require.NoError(t, err)
	var children []string
	for _, l := range links {
		if l.LinkType != nil && l.LinkType.Name == "Subtask" && l.Direction == "OUTWARD" {
			for _, iss := range l.Issues {
				children = append(children, iss.IDReadable)
			}
		}
	}
	assert.ElementsMatch(t, []string{first, second}, children)
}
