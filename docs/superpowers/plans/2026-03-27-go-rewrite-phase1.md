# Go Rewrite — Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a cross-platform Go binary for youtrack-cli covering the foundation and core domains (issues, projects, users), runnable without Node.js.

**Architecture:** Two-layer design — `api/` is a pure Go SDK (HTTP client, typed structs, domain methods) and `cmd/` is a thin Cobra CLI layer over it. Reference implementation: https://github.com/JetBrains/teamcity-cli

**Tech Stack:** Go 1.22+, `spf13/cobra`, stdlib `net/http` + `encoding/json` + `text/tabwriter`, `stretchr/testify` (tests only)

---

## File Map

```
go/
├── main.go
├── go.mod
├── go.sum
├── Makefile
├── api/
│   ├── client.go          # HTTP primitives: Get/Post/Put/Delete/Multipart
│   ├── client_test.go
│   ├── types.go           # Issue, Project, User, Comment, LinkType structs
│   ├── issues.go          # GetIssue, ListIssues, AddIssue, UpdateIssue, DeleteIssue,
│   │                      #   AddComment, ListComments, DeleteComment, ApplyCommand
│   ├── issues_test.go
│   ├── projects.go        # ListProjects, GetProject, AddProject, DeleteProject
│   ├── projects_test.go
│   ├── users.go           # ListUsers, GetUser, GetCurrentUser, AddUser
│   ├── users_test.go
│   └── integration_test.go
└── cmd/
    ├── root.go            # rootCmd, global flags, AddCommand calls
    ├── env.go             # loadEnvFile() — reads ~/.youtrack-cli-env
    ├── client.go          # getClient(cmd) helper
    ├── issue.go           # newIssueCmd() group
    ├── issue_get.go
    ├── issue_list.go
    ├── issue_add.go
    ├── issue_update.go
    ├── issue_delete.go
    ├── issue_comment.go   # comment add/list/delete subgroup
    ├── issue_command.go   # apply-command equivalent
    ├── project.go         # newProjectCmd() group
    ├── project_list.go
    ├── project_get.go
    ├── project_add.go
    ├── project_delete.go
    ├── user.go            # newUserCmd() group
    ├── user_list.go
    ├── user_get.go
    ├── user_current.go
    ├── user_add.go
    └── output/
        ├── json.go        # printJSON via json.Encoder
        └── table.go       # PrintTable via text/tabwriter, type switch
```

**Build order matters:** The `api/` layer must be complete before `cmd/` files are added, because `cmd/issue_add.go` calls `GetProject()` and `GetUser()` from other domains. Tasks 1–8 build the full API layer. Tasks 9–11 add all CLI commands on top.

---

## Task 1: Project scaffold

**Files:**
- Create: `go/go.mod`
- Create: `go/main.go`

- [ ] **Step 1: Create `go/go.mod`**

```
module youtrack-cli

go 1.22

require github.com/spf13/cobra v1.8.0

require (
	github.com/stretchr/testify v1.9.0
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
)
```

- [ ] **Step 2: Create `go/main.go`**

```go
package main

import (
	"fmt"
	"os"

	"youtrack-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 3: Fetch dependencies**

```bash
cd go && go mod tidy
```

Expected: `go.sum` created, no errors.

- [ ] **Step 4: Commit**

```bash
git add go/go.mod go/go.sum go/main.go
git commit -m "chore: scaffold Go module"
```

---

## Task 2: HTTP client

**Files:**
- Create: `go/api/client.go`
- Create: `go/api/client_test.go`

- [ ] **Step 1: Write the failing tests**

Create `go/api/client_test.go`:

```go
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet_success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "1"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-token")
	var result map[string]string
	err := c.Get("/test", &result)

	require.NoError(t, err)
	assert.Equal(t, "1", result["id"])
}

func TestGet_httpError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(map[string]string{"error": "Not Found"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-token")
	err := c.Get("/missing", nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestPost_success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "hello", body["name"])
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "42"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-token")
	var result map[string]string
	err := c.Post("/test", map[string]string{"name": "hello"}, &result)

	require.NoError(t, err)
	assert.Equal(t, "42", result["id"])
}

func TestDelete_success(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		called = true
		w.WriteHeader(204)
	}))
	defer srv.Close()

	err := NewClient(srv.URL, "tok").Delete("/test/1")

	require.NoError(t, err)
	assert.True(t, called)
}
```

- [ ] **Step 2: Run test — verify it fails**

```bash
cd go && go test ./api/ -run "TestGet_success|TestGet_httpError|TestPost_success|TestDelete_success" -v
```

Expected: compile error — `NewClient` not defined.

- [ ] **Step 3: Create `go/api/client.go`**

```go
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
)

// Client sends authenticated requests to a YouTrack instance.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
	Verbose    bool
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL:    baseURL,
		Token:      token,
		HTTPClient: &http.Client{},
	}
}

func (c *Client) doRequest(method, path string, body io.Reader, contentType string) (*http.Response, error) {
	req, err := http.NewRequest(method, c.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if c.Verbose {
		fmt.Fprintf(os.Stderr, "> %s %s%s\n", method, c.BaseURL, path)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	if c.Verbose {
		fmt.Fprintf(os.Stderr, "< %d\n", resp.StatusCode)
	}
	return resp, nil
}

func readResponse(resp *http.Response, result any) error {
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		var apiErr struct {
			ErrorDescription string `json:"error_description"`
			ErrorDevMessage  string `json:"error_developer_message"`
			Error            string `json:"error"`
			Message          string `json:"message"`
		}
		if jsonErr := json.Unmarshal(data, &apiErr); jsonErr == nil {
			for _, detail := range []string{
				apiErr.ErrorDescription,
				apiErr.ErrorDevMessage,
				apiErr.Error,
				apiErr.Message,
			} {
				if detail != "" {
					return fmt.Errorf("HTTP %d: %s", resp.StatusCode, detail)
				}
			}
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data))
	}
	if result == nil || len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, result)
}

func (c *Client) Get(path string, result any) error {
	resp, err := c.doRequest("GET", path, nil, "")
	if err != nil {
		return err
	}
	return readResponse(resp, result)
}

func (c *Client) Post(path string, body, result any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	resp, err := c.doRequest("POST", path, bytes.NewReader(data), "application/json")
	if err != nil {
		return err
	}
	return readResponse(resp, result)
}

func (c *Client) Put(path string, body, result any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	resp, err := c.doRequest("PUT", path, bytes.NewReader(data), "application/json")
	if err != nil {
		return err
	}
	return readResponse(resp, result)
}

func (c *Client) Delete(path string) error {
	resp, err := c.doRequest("DELETE", path, nil, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data))
	}
	return nil
}

// Multipart uploads a file as a multipart/form-data attachment.
func (c *Client) Multipart(path, filePath, fileName, mimeType string, result any) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="attachment"; filename="%s"`, fileName))
	h.Set("Content-Type", mimeType)

	part, err := w.CreatePart(h)
	if err != nil {
		return err
	}
	if _, err = io.Copy(part, f); err != nil {
		return err
	}
	w.Close()

	resp, err := c.doRequest("POST", path, &buf, w.FormDataContentType())
	if err != nil {
		return err
	}
	return readResponse(resp, result)
}
```

- [ ] **Step 4: Run tests — verify they pass**

```bash
cd go && go test ./api/ -run "TestGet_success|TestGet_httpError|TestPost_success|TestDelete_success" -v
```

Expected: all 4 PASS.

- [ ] **Step 5: Commit**

```bash
git add go/api/client.go go/api/client_test.go
git commit -m "feat(go): add HTTP client with unit tests"
```

---

## Task 3: API types

**Files:**
- Create: `go/api/types.go`

- [ ] **Step 1: Create `go/api/types.go`**

```go
package api

// Issue represents a YouTrack issue.
type Issue struct {
	ID          string   `json:"id,omitempty"`
	IDReadable  string   `json:"idReadable,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	Description string   `json:"description,omitempty"`
	Project     *Project `json:"project,omitempty"`
	Reporter    *User    `json:"reporter,omitempty"`
	Resolved    *int64   `json:"resolved,omitempty"`
	Created     int64    `json:"created,omitempty"`
}

// Comment on an issue or article.
type Comment struct {
	ID      string `json:"id,omitempty"`
	Text    string `json:"text,omitempty"`
	Author  *User  `json:"author,omitempty"`
	Created int64  `json:"created,omitempty"`
}

// Project is a YouTrack project.
type Project struct {
	ID        string `json:"id,omitempty"`
	ShortName string `json:"shortName,omitempty"`
	Name      string `json:"name,omitempty"`
	Leader    *User  `json:"leader,omitempty"`
}

// User is a YouTrack user.
type User struct {
	ID       string `json:"id,omitempty"`
	Login    string `json:"login,omitempty"`
	FullName string `json:"fullName,omitempty"`
	Email    string `json:"email,omitempty"`
	Guest    bool   `json:"guest,omitempty"`
	Online   bool   `json:"online,omitempty"`
}

// LinkType describes a type of issue link (e.g. "depends on").
type LinkType struct {
	ID             string `json:"id,omitempty"`
	Name           string `json:"name,omitempty"`
	SourceToTarget string `json:"sourceToTarget,omitempty"`
	TargetToSource string `json:"targetToSource,omitempty"`
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd go && go build ./api/
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add go/api/types.go
git commit -m "feat(go): add API types"
```

---

## Task 4: Issues API

**Files:**
- Create: `go/api/issues.go`
- Create: `go/api/issues_test.go`

- [ ] **Step 1: Write the failing tests**

Create `go/api/issues_test.go`:

```go
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetIssue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "FOO-1")
		json.NewEncoder(w).Encode(Issue{ID: "2-1", IDReadable: "FOO-1", Summary: "Test"})
	}))
	defer srv.Close()

	issue, err := NewClient(srv.URL, "tok").GetIssue("FOO-1")

	require.NoError(t, err)
	assert.Equal(t, "FOO-1", issue.IDReadable)
	assert.Equal(t, "Test", issue.Summary)
}

func TestListIssues(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "query=")
		json.NewEncoder(w).Encode([]Issue{{IDReadable: "FOO-1"}, {IDReadable: "FOO-2"}})
	}))
	defer srv.Close()

	issues, err := NewClient(srv.URL, "tok").ListIssues("project: FOO", 10)

	require.NoError(t, err)
	assert.Len(t, issues, 2)
	assert.Equal(t, "FOO-1", issues[0].IDReadable)
}

func TestAddIssue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		var body AddIssueRequest
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "New issue", body.Summary)
		json.NewEncoder(w).Encode(Issue{IDReadable: "FOO-3", Summary: "New issue"})
	}))
	defer srv.Close()

	issue, err := NewClient(srv.URL, "tok").AddIssue(AddIssueRequest{
		Summary: "New issue",
		Project: &Project{ID: "0-0"},
	})

	require.NoError(t, err)
	assert.Equal(t, "FOO-3", issue.IDReadable)
}

func TestDeleteIssue(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "FOO-1")
		called = true
		w.WriteHeader(204)
	}))
	defer srv.Close()

	err := NewClient(srv.URL, "tok").DeleteIssue("FOO-1")

	require.NoError(t, err)
	assert.True(t, called)
}

func TestApplyCommand(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Method == "GET" {
			json.NewEncoder(w).Encode(Issue{ID: "2-1", IDReadable: "FOO-1"})
			return
		}
		var body ApplyCommandRequest
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "State Fixed", body.Query)
		assert.Len(t, body.Issues, 1)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	err := NewClient(srv.URL, "tok").ApplyCommand("State Fixed", []string{"FOO-1"})

	require.NoError(t, err)
	assert.Equal(t, 2, callCount) // one GET to resolve ID, one POST for command
}

func TestWrapUserValues(t *testing.T) {
	cases := []struct{ in, out string }{
		{"assignee alice", "assignee alice"},
		{"assignee alice.qa", "assignee {alice.qa}"},
		{"reporter bob-dev State Fixed", "reporter {bob-dev} State Fixed"},
		{"assignee {alice.qa}", "assignee {alice.qa}"},
	}
	for _, c := range cases {
		assert.Equal(t, c.out, wrapUserValues(c.in), "input: %q", c.in)
	}
}
```

- [ ] **Step 2: Run tests — verify they fail**

```bash
cd go && go test ./api/ -run "TestGetIssue|TestListIssues|TestAddIssue|TestDeleteIssue|TestApplyCommand|TestWrapUserValues" -v
```

Expected: compile error — `GetIssue`, `AddIssueRequest` not defined.

- [ ] **Step 3: Create `go/api/issues.go`**

```go
package api

import (
	"fmt"
	"net/url"
	"regexp"
)

const issueFields = "id,idReadable,summary,description,project(id,shortName,name),reporter(id,login,fullName),resolved,created"
const commentFields = "id,text,author(id,login,fullName),created"

// AddIssueRequest is the body for creating an issue.
type AddIssueRequest struct {
	Summary     string   `json:"summary"`
	Description string   `json:"description,omitempty"`
	Project     *Project `json:"project"`
	Reporter    *User    `json:"reporter,omitempty"`
}

// UpdateIssueRequest is the body for updating an issue.
type UpdateIssueRequest struct {
	Summary     string `json:"summary,omitempty"`
	Description string `json:"description,omitempty"`
}

// AddCommentRequest is the body for creating a comment.
type AddCommentRequest struct {
	Text   string `json:"text"`
	Author *User  `json:"author,omitempty"`
}

// ApplyCommandRequest is the body for the /api/commands endpoint.
type ApplyCommandRequest struct {
	Query  string  `json:"query"`
	Issues []Issue `json:"issues"`
}

var userValueRe = regexp.MustCompile(`(?i)\b(assignee|reporter)\s+(?!\{)(\S*[.\-@]\S+)`)

// wrapUserValues braces logins containing dots/hyphens for YouTrack's command parser.
// Example: "assignee alice.qa" → "assignee {alice.qa}"
func wrapUserValues(query string) string {
	return userValueRe.ReplaceAllString(query, "$1 {$2}")
}

func (c *Client) GetIssue(id string) (*Issue, error) {
	var result Issue
	path := fmt.Sprintf("/api/issues/%s?fields=%s", url.PathEscape(id), issueFields)
	return &result, c.Get(path, &result)
}

func (c *Client) ListIssues(query string, limit int) ([]Issue, error) {
	var result []Issue
	path := fmt.Sprintf("/api/issues?query=%s&$top=%d&fields=%s",
		url.QueryEscape(query), limit, issueFields)
	return result, c.Get(path, &result)
}

func (c *Client) AddIssue(req AddIssueRequest) (*Issue, error) {
	var result Issue
	return &result, c.Post("/api/issues?fields="+issueFields, req, &result)
}

func (c *Client) UpdateIssue(id string, req UpdateIssueRequest) (*Issue, error) {
	var result Issue
	path := fmt.Sprintf("/api/issues/%s?fields=%s", url.PathEscape(id), issueFields)
	return &result, c.Post(path, req, &result)
}

func (c *Client) DeleteIssue(id string) error {
	return c.Delete(fmt.Sprintf("/api/issues/%s", url.PathEscape(id)))
}

func (c *Client) ListComments(issueID string) ([]Comment, error) {
	var result []Comment
	path := fmt.Sprintf("/api/issues/%s/comments?fields=%s", url.PathEscape(issueID), commentFields)
	return result, c.Get(path, &result)
}

func (c *Client) AddComment(issueID string, req AddCommentRequest) (*Comment, error) {
	var result Comment
	path := fmt.Sprintf("/api/issues/%s/comments?fields=%s", url.PathEscape(issueID), commentFields)
	return &result, c.Post(path, req, &result)
}

func (c *Client) DeleteComment(issueID, commentID string) error {
	return c.Delete(fmt.Sprintf("/api/issues/%s/comments/%s",
		url.PathEscape(issueID), url.PathEscape(commentID)))
}

// ApplyCommand resolves each issue ID to its internal ID then POSTs to /api/commands.
func (c *Client) ApplyCommand(query string, issueIDs []string) error {
	issues := make([]Issue, len(issueIDs))
	for i, id := range issueIDs {
		issue, err := c.GetIssue(id)
		if err != nil {
			return fmt.Errorf("resolve issue %s: %w", id, err)
		}
		issues[i] = Issue{ID: issue.ID}
	}
	return c.Post("/api/commands", ApplyCommandRequest{
		Query:  wrapUserValues(query),
		Issues: issues,
	}, nil)
}
```

- [ ] **Step 4: Run tests — verify they pass**

```bash
cd go && go test ./api/ -run "TestGetIssue|TestListIssues|TestAddIssue|TestDeleteIssue|TestApplyCommand|TestWrapUserValues" -v
```

Expected: all 6 PASS.

- [ ] **Step 5: Commit**

```bash
git add go/api/issues.go go/api/issues_test.go
git commit -m "feat(go): add issues API (CRUD, comments, apply-command)"
```

---

## Task 5: Projects API

**Files:**
- Create: `go/api/projects.go`
- Create: `go/api/projects_test.go`

- [ ] **Step 1: Write the failing tests**

Create `go/api/projects_test.go`:

```go
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListProjects(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Project{
			{ID: "0-1", ShortName: "FOO", Name: "Foo Project"},
			{ID: "0-2", ShortName: "BAR", Name: "Bar Project"},
		})
	}))
	defer srv.Close()

	projects, err := NewClient(srv.URL, "tok").ListProjects()

	require.NoError(t, err)
	assert.Len(t, projects, 2)
	assert.Equal(t, "FOO", projects[0].ShortName)
}

func TestGetProject_found(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Project{{ID: "0-1", ShortName: "FOO", Name: "Foo Project"}})
	}))
	defer srv.Close()

	project, err := NewClient(srv.URL, "tok").GetProject("FOO")

	require.NoError(t, err)
	assert.Equal(t, "FOO", project.ShortName)
	assert.Equal(t, "0-1", project.ID)
}

func TestGetProject_notFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Project{})
	}))
	defer srv.Close()

	_, err := NewClient(srv.URL, "tok").GetProject("MISSING")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "MISSING")
}

func TestAddProject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		var body AddProjectRequest
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "FOO", body.ShortName)
		json.NewEncoder(w).Encode(Project{ID: "0-1", ShortName: "FOO", Name: "Foo Project"})
	}))
	defer srv.Close()

	proj, err := NewClient(srv.URL, "tok").AddProject(AddProjectRequest{
		ShortName: "FOO",
		Name:      "Foo Project",
		Leader:    &User{ID: "1-1"},
	})

	require.NoError(t, err)
	assert.Equal(t, "FOO", proj.ShortName)
}
```

- [ ] **Step 2: Run tests — verify they fail**

```bash
cd go && go test ./api/ -run "TestListProjects|TestGetProject|TestAddProject" -v
```

Expected: compile error — `ListProjects`, `AddProjectRequest` not defined.

- [ ] **Step 3: Create `go/api/projects.go`**

```go
package api

import (
	"fmt"
	"net/url"
)

const projectFields = "id,shortName,name,leader(id,login,fullName)"

// AddProjectRequest is the body for creating a project.
type AddProjectRequest struct {
	ShortName string `json:"shortName"`
	Name      string `json:"name"`
	Leader    *User  `json:"leader"`
}

func (c *Client) ListProjects() ([]Project, error) {
	var result []Project
	return result, c.Get("/api/admin/projects?fields="+projectFields, &result)
}

// GetProject finds a project by its short name (case-sensitive).
func (c *Client) GetProject(shortName string) (*Project, error) {
	projects, err := c.ListProjects()
	if err != nil {
		return nil, err
	}
	for i := range projects {
		if projects[i].ShortName == shortName {
			return &projects[i], nil
		}
	}
	return nil, fmt.Errorf("project '%s' not found", shortName)
}

func (c *Client) AddProject(req AddProjectRequest) (*Project, error) {
	var result Project
	return &result, c.Post("/api/admin/projects?fields="+projectFields, req, &result)
}

func (c *Client) DeleteProject(shortName string) error {
	p, err := c.GetProject(shortName)
	if err != nil {
		return err
	}
	return c.Delete(fmt.Sprintf("/api/admin/projects/%s", url.PathEscape(p.ID)))
}
```

- [ ] **Step 4: Run tests — verify they pass**

```bash
cd go && go test ./api/ -run "TestListProjects|TestGetProject|TestAddProject" -v
```

Expected: all 4 PASS.

- [ ] **Step 5: Commit**

```bash
git add go/api/projects.go go/api/projects_test.go
git commit -m "feat(go): add projects API"
```

---

## Task 6: Users API

**Files:**
- Create: `go/api/users.go`
- Create: `go/api/users_test.go`

- [ ] **Step 1: Write the failing tests**

Create `go/api/users_test.go`:

```go
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListUsers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "query=")
		json.NewEncoder(w).Encode([]User{{Login: "alice", FullName: "Alice Smith"}})
	}))
	defer srv.Close()

	users, err := NewClient(srv.URL, "tok").ListUsers("alice")

	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "alice", users[0].Login)
}

func TestGetUser_found(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]User{{Login: "alice", FullName: "Alice Smith"}})
	}))
	defer srv.Close()

	user, err := NewClient(srv.URL, "tok").GetUser("alice")

	require.NoError(t, err)
	assert.Equal(t, "alice", user.Login)
}

func TestGetUser_notFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]User{})
	}))
	defer srv.Close()

	_, err := NewClient(srv.URL, "tok").GetUser("nobody")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nobody")
}

func TestGetCurrentUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/users/me")
		json.NewEncoder(w).Encode(User{Login: "admin", FullName: "Admin User"})
	}))
	defer srv.Close()

	user, err := NewClient(srv.URL, "tok").GetCurrentUser()

	require.NoError(t, err)
	assert.Equal(t, "admin", user.Login)
}
```

- [ ] **Step 2: Run tests — verify they fail**

```bash
cd go && go test ./api/ -run "TestListUsers|TestGetUser|TestGetCurrentUser" -v
```

Expected: compile error — `ListUsers`, `GetCurrentUser` not defined.

- [ ] **Step 3: Create `go/api/users.go`**

```go
package api

import (
	"fmt"
	"net/url"
)

const userFields = "id,login,fullName,email,guest,online"

// AddUserRequest is the body for creating a user.
type AddUserRequest struct {
	Login    string `json:"login"`
	FullName string `json:"fullName"`
	Email    string `json:"email"`
}

func (c *Client) ListUsers(query string) ([]User, error) {
	var result []User
	path := fmt.Sprintf("/api/users?query=%s&fields=%s", url.QueryEscape(query), userFields)
	return result, c.Get(path, &result)
}

// GetUser finds a user by exact login match.
func (c *Client) GetUser(login string) (*User, error) {
	users, err := c.ListUsers(login)
	if err != nil {
		return nil, err
	}
	for i := range users {
		if users[i].Login == login {
			return &users[i], nil
		}
	}
	return nil, fmt.Errorf("user '%s' not found", login)
}

func (c *Client) GetCurrentUser() (*User, error) {
	var result User
	return &result, c.Get("/api/users/me?fields="+userFields, &result)
}

func (c *Client) AddUser(req AddUserRequest) (*User, error) {
	var result User
	return &result, c.Post("/api/users?fields="+userFields, req, &result)
}
```

- [ ] **Step 4: Run all API tests**

```bash
cd go && go test ./api/ -v
```

Expected: all tests PASS (client + issues + projects + users).

- [ ] **Step 5: Commit**

```bash
git add go/api/users.go go/api/users_test.go
git commit -m "feat(go): add users API"
```

---

## Task 7: Output module + CLI foundation

**Files:**
- Create: `go/cmd/output/json.go`
- Create: `go/cmd/output/table.go`
- Create: `go/cmd/env.go`
- Create: `go/cmd/client.go`
- Create: `go/cmd/root.go`

- [ ] **Step 1: Create `go/cmd/output/json.go`**

```go
package output

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
)

// Print writes data to stdout using the format specified by the global --format flag.
func Print(cmd *cobra.Command, data any) error {
	format, _ := cmd.Root().PersistentFlags().GetString("format")
	if format == "table" {
		return PrintTable(data)
	}
	return printJSON(data)
}

func printJSON(data any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}
```

- [ ] **Step 2: Create `go/cmd/output/table.go`**

```go
package output

import (
	"fmt"
	"os"
	"text/tabwriter"

	"youtrack-cli/api"
)

// PrintTable renders data as a human-readable table. Falls back to JSON for unknown types.
func PrintTable(data any) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	switch v := data.(type) {
	case *api.Issue:
		fmt.Fprintln(w, "ID\tSUMMARY\tPROJECT")
		fmt.Fprintf(w, "%s\t%s\t%s\n", v.IDReadable, v.Summary, projectKey(v.Project))
	case []api.Issue:
		fmt.Fprintln(w, "ID\tSUMMARY\tPROJECT")
		for _, i := range v {
			fmt.Fprintf(w, "%s\t%s\t%s\n", i.IDReadable, i.Summary, projectKey(i.Project))
		}
	case *api.Project:
		fmt.Fprintln(w, "KEY\tNAME")
		fmt.Fprintf(w, "%s\t%s\n", v.ShortName, v.Name)
	case []api.Project:
		fmt.Fprintln(w, "KEY\tNAME")
		for _, p := range v {
			fmt.Fprintf(w, "%s\t%s\n", p.ShortName, p.Name)
		}
	case *api.User:
		fmt.Fprintln(w, "LOGIN\tNAME\tEMAIL")
		fmt.Fprintf(w, "%s\t%s\t%s\n", v.Login, v.FullName, v.Email)
	case []api.User:
		fmt.Fprintln(w, "LOGIN\tNAME\tEMAIL")
		for _, u := range v {
			fmt.Fprintf(w, "%s\t%s\t%s\n", u.Login, u.FullName, u.Email)
		}
	case []api.Comment:
		fmt.Fprintln(w, "ID\tAUTHOR\tTEXT")
		for _, c := range v {
			author := ""
			if c.Author != nil {
				author = c.Author.Login
			}
			text := c.Text
			if len(text) > 60 {
				text = text[:57] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", c.ID, author, text)
		}
	default:
		return printJSON(data)
	}
	return nil
}

func projectKey(p *api.Project) string {
	if p == nil {
		return ""
	}
	return p.ShortName
}
```

- [ ] **Step 3: Create `go/cmd/env.go`**

```go
package cmd

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// loadEnvFile reads ~/.youtrack-cli-env and sets unset env vars from it.
// Format: KEY=VALUE one per line. Lines starting with # are ignored.
func loadEnvFile() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	f, err := os.Open(filepath.Join(home, ".youtrack-cli-env"))
	if err != nil {
		return // file doesn't exist — that's fine
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}
```

- [ ] **Step 4: Create `go/cmd/client.go`**

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"youtrack-cli/api"
)

// getClient builds an API client from global flags and env vars.
// Exits with an error message if base URL or token are missing.
func getClient(cmd *cobra.Command) *api.Client {
	flags := cmd.Root().PersistentFlags()

	baseURL, _ := flags.GetString("base-url")
	token, _ := flags.GetString("token")
	verbose, _ := flags.GetBool("verbose")

	if baseURL == "" {
		baseURL = os.Getenv("YOUTRACK_BASE_URL")
	}
	if token == "" {
		token = os.Getenv("YOUTRACK_TOKEN")
	}

	if baseURL == "" {
		fmt.Fprintln(os.Stderr, "error: YOUTRACK_BASE_URL is not set. Set it via env or --base-url flag.")
		os.Exit(1)
	}
	if token == "" {
		fmt.Fprintln(os.Stderr, "error: YOUTRACK_TOKEN is not set. Set it via env or --token flag.")
		os.Exit(1)
	}

	c := api.NewClient(baseURL, token)
	c.Verbose = verbose
	return c
}
```

- [ ] **Step 5: Create `go/cmd/root.go`** (without AddCommand calls — those come in Tasks 8–10)

```go
package cmd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "youtrack",
	Short: "YouTrack REST API CLI",
	Long:  "CLI for interacting with the YouTrack REST API. Designed for AI agents and developers.",
}

// Execute is the entrypoint called from main.
func Execute() error {
	loadEnvFile()
	return rootCmd.Execute()
}

func init() {
	f := rootCmd.PersistentFlags()
	f.String("format", "json", "Output format: json or table")
	f.String("base-url", "", "Override YOUTRACK_BASE_URL")
	f.String("token", "", "Override YOUTRACK_TOKEN")
	f.Bool("verbose", false, "Print HTTP request/response to stderr")
}
```

- [ ] **Step 6: Verify full build compiles**

```bash
cd go && go build ./...
```

Expected: no errors.

- [ ] **Step 7: Commit**

```bash
git add go/cmd/
git commit -m "feat(go): add output formatters and CLI foundation"
```

---

## Task 8: Issue commands

**Files:**
- Create: `go/cmd/issue.go`
- Create: `go/cmd/issue_get.go`
- Create: `go/cmd/issue_list.go`
- Create: `go/cmd/issue_add.go`
- Create: `go/cmd/issue_update.go`
- Create: `go/cmd/issue_delete.go`
- Create: `go/cmd/issue_comment.go`
- Create: `go/cmd/issue_command.go`
- Modify: `go/cmd/root.go`

- [ ] **Step 1: Create `go/cmd/issue.go`**

```go
package cmd

import "github.com/spf13/cobra"

func newIssueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue",
		Short: "Manage issues",
	}
	cmd.AddCommand(newIssueGetCmd())
	cmd.AddCommand(newIssueListCmd())
	cmd.AddCommand(newIssueAddCmd())
	cmd.AddCommand(newIssueUpdateCmd())
	cmd.AddCommand(newIssueDeleteCmd())
	cmd.AddCommand(newIssueCommentCmd())
	cmd.AddCommand(newIssueCommandCmd())
	return cmd
}
```

- [ ] **Step 2: Create `go/cmd/issue_get.go`**

```go
package cmd

import (
	"youtrack-cli/cmd/output"

	"github.com/spf13/cobra"
)

func newIssueGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <issueId>",
		Short: "Get issue by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issue, err := getClient(cmd).GetIssue(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, issue)
		},
	}
}
```

- [ ] **Step 3: Create `go/cmd/issue_list.go`**

```go
package cmd

import (
	"youtrack-cli/cmd/output"

	"github.com/spf13/cobra"
)

func newIssueListCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Search issues by query",
		RunE: func(cmd *cobra.Command, args []string) error {
			query, _ := cmd.Flags().GetString("query")
			issues, err := getClient(cmd).ListIssues(query, limit)
			if err != nil {
				return err
			}
			return output.Print(cmd, issues)
		},
	}
	cmd.Flags().String("query", "", "YouTrack query string (required)")
	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum number of issues to return")
	cmd.MarkFlagRequired("query")
	return cmd
}
```

- [ ] **Step 4: Create `go/cmd/issue_add.go`**

```go
package cmd

import (
	"youtrack-cli/api"
	"youtrack-cli/cmd/output"

	"github.com/spf13/cobra"
)

func newIssueAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a new issue",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectShortName, _ := cmd.Flags().GetString("project")
			summary, _ := cmd.Flags().GetString("summary")
			description, _ := cmd.Flags().GetString("description")
			asUser, _ := cmd.Flags().GetString("asUser")

			client := getClient(cmd)

			project, err := client.GetProject(projectShortName)
			if err != nil {
				return err
			}

			req := api.AddIssueRequest{
				Summary:     summary,
				Description: description,
				Project:     &api.Project{ID: project.ID},
			}

			if asUser != "" {
				user, err := client.GetUser(asUser)
				if err != nil {
					return err
				}
				req.Reporter = &api.User{ID: user.ID}
			}

			issue, err := client.AddIssue(req)
			if err != nil {
				return err
			}
			return output.Print(cmd, issue)
		},
	}
	cmd.Flags().String("project", "", "Project short name (required)")
	cmd.Flags().String("summary", "", "Issue summary (required)")
	cmd.Flags().String("description", "", "Issue description")
	cmd.Flags().String("asUser", "", "Create as this user (login)")
	cmd.MarkFlagRequired("project")
	cmd.MarkFlagRequired("summary")
	return cmd
}
```

- [ ] **Step 5: Create `go/cmd/issue_update.go`**

```go
package cmd

import (
	"youtrack-cli/api"
	"youtrack-cli/cmd/output"

	"github.com/spf13/cobra"
)

func newIssueUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <issueId>",
		Short: "Update issue summary or description",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			summary, _ := cmd.Flags().GetString("summary")
			description, _ := cmd.Flags().GetString("description")

			issue, err := getClient(cmd).UpdateIssue(args[0], api.UpdateIssueRequest{
				Summary:     summary,
				Description: description,
			})
			if err != nil {
				return err
			}
			return output.Print(cmd, issue)
		},
	}
	cmd.Flags().String("summary", "", "New summary")
	cmd.Flags().String("description", "", "New description")
	return cmd
}
```

- [ ] **Step 6: Create `go/cmd/issue_delete.go`**

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newIssueDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <issueId>",
		Short: "Delete an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := getClient(cmd).DeleteIssue(args[0]); err != nil {
				return err
			}
			fmt.Printf("deleted %s\n", args[0])
			return nil
		},
	}
}
```

- [ ] **Step 7: Create `go/cmd/issue_comment.go`**

```go
package cmd

import (
	"fmt"
	"youtrack-cli/api"
	"youtrack-cli/cmd/output"

	"github.com/spf13/cobra"
)

func newIssueCommentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Manage issue comments",
	}
	cmd.AddCommand(newCommentListCmd())
	cmd.AddCommand(newCommentAddCmd())
	cmd.AddCommand(newCommentDeleteCmd())
	return cmd
}

func newCommentListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <issueId>",
		Short: "List comments on an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			comments, err := getClient(cmd).ListComments(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, comments)
		},
	}
}

func newCommentAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <issueId>",
		Short: "Add a comment to an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			text, _ := cmd.Flags().GetString("text")
			asUser, _ := cmd.Flags().GetString("asUser")

			client := getClient(cmd)
			req := api.AddCommentRequest{Text: text}

			if asUser != "" {
				user, err := client.GetUser(asUser)
				if err != nil {
					return err
				}
				req.Author = &api.User{ID: user.ID}
			}

			comment, err := client.AddComment(args[0], req)
			if err != nil {
				return err
			}
			return output.Print(cmd, comment)
		},
	}
	cmd.Flags().String("text", "", "Comment text (required)")
	cmd.Flags().String("asUser", "", "Post as this user (login)")
	cmd.MarkFlagRequired("text")
	return cmd
}

func newCommentDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <issueId> <commentId>",
		Short: "Delete a comment",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := getClient(cmd).DeleteComment(args[0], args[1]); err != nil {
				return err
			}
			fmt.Printf("deleted comment %s\n", args[1])
			return nil
		},
	}
}
```

- [ ] **Step 8: Create `go/cmd/issue_command.go`**

```go
package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newIssueCommandCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "command <issueId[,issueId...]> <query>",
		Short: "Apply a YouTrack command to one or more issues",
		Example: `  youtrack issue command FOO-42 "State Fixed"
  youtrack issue command FOO-1,FOO-2 "assignee alice Priority Critical"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueIDs := strings.Split(args[0], ",")
			for i, id := range issueIDs {
				issueIDs[i] = strings.TrimSpace(id)
			}
			if err := getClient(cmd).ApplyCommand(args[1], issueIDs); err != nil {
				return err
			}
			fmt.Printf("command applied to %s\n", args[0])
			return nil
		},
	}
}
```

- [ ] **Step 9: Register issue command in `go/cmd/root.go`**

Add to the `init()` function in `go/cmd/root.go`:

```go
func init() {
	f := rootCmd.PersistentFlags()
	f.String("format", "json", "Output format: json or table")
	f.String("base-url", "", "Override YOUTRACK_BASE_URL")
	f.String("token", "", "Override YOUTRACK_TOKEN")
	f.Bool("verbose", false, "Print HTTP request/response to stderr")

	rootCmd.AddCommand(newIssueCmd()) // add this line
}
```

- [ ] **Step 10: Build and smoke test**

```bash
cd go && go build -o youtrack . && ./youtrack issue --help
```

Expected: help text listing `get`, `list`, `add`, `update`, `delete`, `comment`, `command`.

- [ ] **Step 11: Commit**

```bash
git add go/cmd/issue*.go go/cmd/root.go
git commit -m "feat(go): add issue commands"
```

---

## Task 9: Project commands

**Files:**
- Create: `go/cmd/project.go`
- Create: `go/cmd/project_list.go`
- Create: `go/cmd/project_get.go`
- Create: `go/cmd/project_add.go`
- Create: `go/cmd/project_delete.go`
- Modify: `go/cmd/root.go`

- [ ] **Step 1: Create `go/cmd/project.go`**

```go
package cmd

import "github.com/spf13/cobra"

func newProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
	}
	cmd.AddCommand(newProjectListCmd())
	cmd.AddCommand(newProjectGetCmd())
	cmd.AddCommand(newProjectAddCmd())
	cmd.AddCommand(newProjectDeleteCmd())
	return cmd
}
```

- [ ] **Step 2: Create `go/cmd/project_list.go`**

```go
package cmd

import (
	"youtrack-cli/cmd/output"

	"github.com/spf13/cobra"
)

func newProjectListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			projects, err := getClient(cmd).ListProjects()
			if err != nil {
				return err
			}
			return output.Print(cmd, projects)
		},
	}
}
```

- [ ] **Step 3: Create `go/cmd/project_get.go`**

```go
package cmd

import (
	"youtrack-cli/cmd/output"

	"github.com/spf13/cobra"
)

func newProjectGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <shortName>",
		Short: "Get a project by short name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := getClient(cmd).GetProject(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, project)
		},
	}
}
```

- [ ] **Step 4: Create `go/cmd/project_add.go`**

```go
package cmd

import (
	"youtrack-cli/api"
	"youtrack-cli/cmd/output"

	"github.com/spf13/cobra"
)

func newProjectAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a new project",
		RunE: func(cmd *cobra.Command, args []string) error {
			shortName, _ := cmd.Flags().GetString("shortName")
			name, _ := cmd.Flags().GetString("name")
			leader, _ := cmd.Flags().GetString("leader")

			client := getClient(cmd)

			// Default leader to current user if not specified
			if leader == "" {
				me, err := client.GetCurrentUser()
				if err != nil {
					return err
				}
				leader = me.Login
			}

			leaderUser, err := client.GetUser(leader)
			if err != nil {
				return err
			}

			project, err := client.AddProject(api.AddProjectRequest{
				ShortName: shortName,
				Name:      name,
				Leader:    &api.User{ID: leaderUser.ID},
			})
			if err != nil {
				return err
			}
			return output.Print(cmd, project)
		},
	}
	cmd.Flags().String("shortName", "", "Project short name / key (required)")
	cmd.Flags().String("name", "", "Project full name (required)")
	cmd.Flags().String("leader", "", "Leader login (defaults to current user)")
	cmd.MarkFlagRequired("shortName")
	cmd.MarkFlagRequired("name")
	return cmd
}
```

- [ ] **Step 5: Create `go/cmd/project_delete.go`**

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newProjectDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <shortName>",
		Short: "Delete a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := getClient(cmd).DeleteProject(args[0]); err != nil {
				return err
			}
			fmt.Printf("deleted project %s\n", args[0])
			return nil
		},
	}
}
```

- [ ] **Step 6: Register project command in `go/cmd/root.go`**

```go
rootCmd.AddCommand(newIssueCmd())
rootCmd.AddCommand(newProjectCmd()) // add this line
```

- [ ] **Step 7: Build and smoke test**

```bash
cd go && go build -o youtrack . && ./youtrack project --help
```

Expected: help text listing `list`, `get`, `add`, `delete`.

- [ ] **Step 8: Commit**

```bash
git add go/cmd/project*.go go/cmd/root.go
git commit -m "feat(go): add project commands"
```

---

## Task 10: User commands

**Files:**
- Create: `go/cmd/user.go`
- Create: `go/cmd/user_list.go`
- Create: `go/cmd/user_get.go`
- Create: `go/cmd/user_current.go`
- Create: `go/cmd/user_add.go`
- Modify: `go/cmd/root.go`

- [ ] **Step 1: Create `go/cmd/user.go`**

```go
package cmd

import "github.com/spf13/cobra"

func newUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage users",
	}
	cmd.AddCommand(newUserListCmd())
	cmd.AddCommand(newUserGetCmd())
	cmd.AddCommand(newUserCurrentCmd())
	cmd.AddCommand(newUserAddCmd())
	return cmd
}
```

- [ ] **Step 2: Create `go/cmd/user_list.go`**

```go
package cmd

import (
	"youtrack-cli/cmd/output"

	"github.com/spf13/cobra"
)

func newUserListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List users",
		RunE: func(cmd *cobra.Command, args []string) error {
			query, _ := cmd.Flags().GetString("query")
			users, err := getClient(cmd).ListUsers(query)
			if err != nil {
				return err
			}
			return output.Print(cmd, users)
		},
	}
	cmd.Flags().String("query", "", "Filter users by query")
	return cmd
}
```

- [ ] **Step 3: Create `go/cmd/user_get.go`**

```go
package cmd

import (
	"youtrack-cli/cmd/output"

	"github.com/spf13/cobra"
)

func newUserGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <login>",
		Short: "Get user by login",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			user, err := getClient(cmd).GetUser(args[0])
			if err != nil {
				return err
			}
			return output.Print(cmd, user)
		},
	}
}
```

- [ ] **Step 4: Create `go/cmd/user_current.go`**

```go
package cmd

import (
	"youtrack-cli/cmd/output"

	"github.com/spf13/cobra"
)

func newUserCurrentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Get the currently authenticated user",
		RunE: func(cmd *cobra.Command, args []string) error {
			user, err := getClient(cmd).GetCurrentUser()
			if err != nil {
				return err
			}
			return output.Print(cmd, user)
		},
	}
}
```

- [ ] **Step 5: Create `go/cmd/user_add.go`**

```go
package cmd

import (
	"youtrack-cli/api"
	"youtrack-cli/cmd/output"

	"github.com/spf13/cobra"
)

func newUserAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a new user",
		RunE: func(cmd *cobra.Command, args []string) error {
			login, _ := cmd.Flags().GetString("login")
			fullName, _ := cmd.Flags().GetString("fullName")
			email, _ := cmd.Flags().GetString("email")

			user, err := getClient(cmd).AddUser(api.AddUserRequest{
				Login:    login,
				FullName: fullName,
				Email:    email,
			})
			if err != nil {
				return err
			}
			return output.Print(cmd, user)
		},
	}
	cmd.Flags().String("login", "", "User login (required)")
	cmd.Flags().String("fullName", "", "User full name (required)")
	cmd.Flags().String("email", "", "User email (required)")
	cmd.MarkFlagRequired("login")
	cmd.MarkFlagRequired("fullName")
	cmd.MarkFlagRequired("email")
	return cmd
}
```

- [ ] **Step 6: Register user command in `go/cmd/root.go`**

```go
rootCmd.AddCommand(newIssueCmd())
rootCmd.AddCommand(newProjectCmd())
rootCmd.AddCommand(newUserCmd()) // add this line
```

- [ ] **Step 7: Full build + all unit tests**

```bash
cd go && go build -o youtrack . && go test ./... -v
```

Expected: binary built, all unit tests PASS.

- [ ] **Step 8: Commit**

```bash
git add go/cmd/user*.go go/cmd/root.go
git commit -m "feat(go): add user commands"
```

---

## Task 11: Integration tests

**Files:**
- Create: `go/api/integration_test.go`

These tests require a real YouTrack instance and use the `integration` build tag so they are excluded from `go test ./...` by default.

- [ ] **Step 1: Create `go/api/integration_test.go`**

```go
//go:build integration

package api

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// integProjectKey must exist in the YouTrack instance before running these tests.
const integProjectKey = "GOTEST"

var (
	integClient    *Client
	createdIssueID string
)

func TestMain(m *testing.M) {
	baseURL := os.Getenv("YOUTRACK_BASE_URL")
	token := os.Getenv("YOUTRACK_TOKEN")
	if baseURL == "" || token == "" {
		os.Exit(0) // skip silently when env not configured
	}
	integClient = NewClient(baseURL, token)
	code := m.Run()
	integCleanup()
	os.Exit(code)
}

func integCleanup() {
	if createdIssueID != "" {
		_ = integClient.DeleteIssue(createdIssueID)
	}
}

func TestIntegration_GetCurrentUser(t *testing.T) {
	user, err := integClient.GetCurrentUser()
	require.NoError(t, err)
	assert.NotEmpty(t, user.Login)
}

func TestIntegration_ListProjects(t *testing.T) {
	projects, err := integClient.ListProjects()
	require.NoError(t, err)
	assert.NotEmpty(t, projects)
}

func TestIntegration_AddGetDeleteIssue(t *testing.T) {
	proj, err := integClient.GetProject(integProjectKey)
	require.NoError(t, err, "project %s must exist in YouTrack", integProjectKey)

	issue, err := integClient.AddIssue(AddIssueRequest{
		Summary: "Integration test — safe to delete",
		Project: &Project{ID: proj.ID},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, issue.IDReadable)
	createdIssueID = issue.IDReadable

	fetched, err := integClient.GetIssue(createdIssueID)
	require.NoError(t, err)
	assert.Equal(t, "Integration test — safe to delete", fetched.Summary)
}
```

- [ ] **Step 2: Run integration tests (requires real YouTrack)**

```bash
export $(cat ~/.youtrack-cli-env | xargs)
cd go && go test -tags integration -run "TestIntegration" -v ./api/
```

Expected: all 3 integration tests PASS. Created issue is cleaned up in teardown.

- [ ] **Step 3: Commit**

```bash
git add go/api/integration_test.go
git commit -m "test(go): add integration tests"
```

---

## Task 12: Build script and README

**Files:**
- Create: `go/Makefile`
- Modify: `README.md`

- [ ] **Step 1: Create `go/Makefile`**

```makefile
BINARY=youtrack

build:
	go build -o $(BINARY) .

test:
	go test ./...

test-integration:
	go test -tags integration -run TestIntegration -v ./api/

dist:
	mkdir -p dist
	GOOS=linux   GOARCH=amd64 go build -o dist/youtrack-linux-amd64 .
	GOOS=darwin  GOARCH=arm64 go build -o dist/youtrack-darwin-arm64 .
	GOOS=darwin  GOARCH=amd64 go build -o dist/youtrack-darwin-amd64 .
	GOOS=windows GOARCH=amd64 go build -o dist/youtrack-windows-amd64.exe .

clean:
	rm -f $(BINARY)
	rm -rf dist/

.PHONY: build test test-integration dist clean
```

- [ ] **Step 2: Verify Makefile works**

```bash
cd go && make build && make test
```

Expected: binary at `go/youtrack`, all unit tests PASS.

- [ ] **Step 3: Add `go/dist/` to `.gitignore`**

Append to the root `.gitignore` (create if missing):

```
go/dist/
go/youtrack
```

- [ ] **Step 4: Commit**

```bash
git add go/Makefile .gitignore
git commit -m "chore(go): add Makefile and update .gitignore"
```

---

## Phase 2 scope (next plan)

The following domains follow the identical pattern established in this plan (`api/<domain>.go` + tests + `cmd/<domain>*.go`) and are covered in a separate Phase 2 plan:

- `article` — articles, article comments, article attachments
- `sprint` + `board`
- `tag`
- `field` + `bundle` — custom fields and bundles
- `workflow`
- `admin` — global settings, feature flags
- Attachments — `issue attachment add`, `article attachment add`
