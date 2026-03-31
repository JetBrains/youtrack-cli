# Design: Directory Restructuring + Integration Test Rewrite

## Context

The `cmd/` package has grown to 62 flat files, making navigation and ownership hard. All ~164 tests use `httptest.NewServer()` mocks — they don't validate real YouTrack API behaviour and give false confidence. This change groups `cmd/` into domain sub-packages and replaces all mock tests with fully self-contained integration tests against a real YouTrack instance. The `api/` package stays flat (all methods live on `*Client`; splitting would add indirection with no gain at this scale).

---

## 1. Directory Structure

### `internal/` (new)

Extracted from current `cmd/client.go` and `cmd/env.go`. Needed to break the circular dependency that would arise if domain sub-packages imported `package cmd`.

```
internal/
  client.go    # package internal — GetClient(cmd *cobra.Command) *api.Client
  env.go       # package internal — LoadEnvFile()
```

### `cmd/` after restructure

Only domains with multiple files get a subdirectory. Single-file domains stay flat.

**Subdirectories** (multiple files → moved to sub-package):

```
cmd/
  youtrack.go          # package cmd — root command, Execute(), persistent flags, registers all subcommands
  output/              # unchanged (json.go, table.go, logo.go)
  issue/
    issue.go           # package issue — NewCmd(), wires subcommands
    add.go, get.go, list.go, update.go, delete.go
    command.go, comment.go, attachment.go, field.go
    tag.go, link.go, workitem.go, reaction.go, visibility.go, count.go, assist.go
  project/
    project.go         # NewCmd()
    list.go, get.go, add.go, delete.go, archive.go
    field.go, team.go, workflow.go, template.go, timetrack.go
  user/
    user.go            # NewCmd()
    list.go, get.go, current.go, add.go, delete.go
    ban.go, profile.go, timezone.go, locale.go, drafts.go, pin_project.go
```

**Flat files** (single file — unchanged location, just updated imports):
```
cmd/
  article.go, board.go, canvas.go, dashboard.go, field.go
  group.go, mailbox.go, org.go, report.go, role.go
  search.go, settings.go, tag.go, vcs.go, workflow.go
  app.go, raw.go
```

`cmd/root.go` registration:
```go
root.AddCommand(
    issue.NewCmd(), project.NewCmd(), user.NewCmd(),
    // flat commands register themselves via newXxxCmd() in the same package
)
```

### `api/` — unchanged

Stays flat. All methods remain on `*api.Client`.

---

## 2. Tests

### Philosophy

- All tests are integration tests against a real YouTrack instance
- No mocks, no `httptest.NewServer()`
- No `//go:build integration` build tag — all tests require real instance
- Each test is fully self-contained: creates all needed entities, registers `t.Cleanup()` teardown

### Structure

```
test/
  helpers_test.go      # newTestClient(t), createTempProject(t,c), TestMain (env guard)
  issues_test.go       # api methods + cmd commands for issues
  projects_test.go
  users_test.go
  articles_test.go
  admin_test.go
  boards_test.go
  dashboards_test.go
  fields_test.go
  groups_test.go
  roles_test.go
  workflows_test.go
  misc_test.go         # tags, searches, orgs, apps, canvas
```

### Isolation Pattern

```go
func TestIssue_CreateGetDelete(t *testing.T) {
    c := newTestClient(t)
    proj := createTempProject(t, c)  // t.Cleanup registers deletion

    id, err := c.AddIssue(proj.ShortName, "Test issue", "")
    require.NoError(t, err)
    t.Cleanup(func() { c.DeleteIssue(id) })

    got, err := c.GetIssue(id)
    require.NoError(t, err)
    assert.Equal(t, "Test issue", got.Summary)
}
```

### Cmd Coverage

Each domain file covers both layers:
- **API layer**: calls `api.Client` methods directly, asserts on returned structs
- **Cmd layer**: invokes cobra commands via `rootCmd.SetArgs([]string{...})` + `Execute()`, captures stdout, asserts on JSON/table output

### `TestMain` in `helpers_test.go`

```go
func TestMain(m *testing.M) {
    if os.Getenv("YOUTRACK_BASE_URL") == "" || os.Getenv("YOUTRACK_TOKEN") == "" {
        fmt.Fprintln(os.Stderr, "YOUTRACK_BASE_URL and YOUTRACK_TOKEN must be set")
        os.Exit(1)
    }
    os.Exit(m.Run())
}
```

### Makefile

```makefile
test:
    go test ./...

test-integration:   # kept as alias, same behaviour
    go test -v ./test/
```

---

## 3. Critical Files

| File | Change |
|------|--------|
| `cmd/root.go` → `cmd/youtrack.go` | Rename; slim registrar; imports domain packages |
| `cmd/client.go` → `internal/client.go` | Move + export `GetClient` |
| `cmd/env.go` → `internal/env.go` | Move + export `LoadEnvFile` |
| `cmd/issue*.go` (17 files) → `cmd/issue/*.go` | Move to sub-package |
| `cmd/project*.go` (10 files) → `cmd/project/*.go` | Move to sub-package |
| `cmd/user*.go` (12 files) → `cmd/user/*.go` | Move to sub-package |
| All other `cmd/*.go` single-file domains | Update imports only (stay flat) |
| `test/*.go` (14 files) | Rewrite: remove mocks, add real + cmd tests |
| `Makefile` | Drop `-tags integration` from `test` target |

---

## 4. Verification

```bash
make build                        # binary compiles cleanly
export $(cat ~/.youtrack-cli-env | xargs)
make test                         # all integration tests pass
go vet ./...                      # no vet errors
./youtrack issue list             # smoke test real instance
./youtrack project list           # smoke test real instance
```
