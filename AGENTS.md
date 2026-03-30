# AGENTS.md

Go CLI for interacting with the YouTrack REST API. Built with Cobra. Designed for use by AI agents and developers.

## Agent Skill

A full YouTrack interaction guide is available as an agent skill — load it when the user asks to manage YouTrack entities (issues, articles, projects, users, sprints, etc.):

**`.agents/skills/youtrack-cli/SKILL.md`** — command reference, common workflows, output formats, and YouTrack-specific gotchas.

## Development Commands

```bash
make build                                           # compile ./youtrack binary
make test                                            # go test -tags integration ./... (unit + integration)
make test-integration                                # integration tests only, verbose
go run . issue list --query "assignee: me"           # run without building
go test ./test/ -run TestGetIssue -v                 # run a single test by name
```

## Environment

```bash
export YOUTRACK_BASE_URL=https://your-instance.youtrack.cloud
export YOUTRACK_TOKEN=perm:...
```

Or write to `~/.youtrack-cli-env` (one `KEY=VALUE` per line) — loaded automatically on each invocation.

## Project Structure

| Path | Role |
|---|---|
| `main.go` | Entrypoint — calls `cmd.Execute()` |
| `go.mod` / `go.sum` | Module definition (module: `youtrack-cli`, Go 1.22) |
| `Makefile` | Build, test, and dist targets |
| `api/client.go` | `Client` struct, `Get`/`Post`/`Put`/`Delete` HTTP primitives |
| `api/types.go` | Shared struct types (Issue, Project, User, Comment, etc.) |
| `api/issues.go` | Issue and comment API methods |
| `api/projects.go` | Project API methods |
| `api/users.go` | User API methods |
| `cmd/root.go` | Cobra root command, global flags, subcommand registration |
| `cmd/client.go` | `getClient()` helper — reads flags and env vars |
| `cmd/env.go` | `loadEnvFile()` — loads `~/.youtrack-cli-env` |
| `cmd/issue*.go` | Issue subcommands |
| `cmd/project*.go` | Project subcommands |
| `cmd/user*.go` | User subcommands |
| `cmd/output/` | JSON and table output formatters |
| `test/` | All unit and integration tests (`package api_test`) |

## Command Registration

Add a new file in `cmd/` (e.g., `cmd/issue_foo.go`). Create a constructor `newIssueFooCmd() *cobra.Command` and register it by calling `cmd.AddCommand(newIssueFooCmd())` in the parent command's constructor (e.g., `newIssueCmd()` in `cmd/issue.go`).

- Use `getClient(cmd)` to obtain the API client.
- Return errors from `RunE` — the root command prints them to stderr.
- Use `output.Print(cmd, data)` for all output — it respects `--format`.

## When Adding a Command

- Add a usage example and table row to `README.md`.
- Add to `.agents/skills/youtrack-cli/SKILL.md` if part of a non-trivial workflow.

## Coding Conventions

- `gofmt` formatting — run automatically by most editors with gopls.
- Command names: lowercase verbs (`get`, `list`, `add`, `update`, `delete`).
- Go identifiers: camelCase for unexported, PascalCase for exported.
- Error strings: lowercase, no trailing period (Go convention).
- All output goes through `output.Print(cmd, data)` — never `fmt.Println` directly in a command handler.

## Testing

Unit tests use `httptest.NewServer` — no real instance needed. Run with `make test`.

Integration tests use the `//go:build integration` build tag and require a real YouTrack instance with a `GOTEST` project. Run with `make test-integration`. Clean up all created resources in `t.Cleanup()` callbacks or the `integCleanup()` helper.

## ID Conventions

| Entity | Identifier | Example |
|---|---|---|
| Issue | `<PROJECT>-<n>` | `FOO-42` |
| Project | short name | `FOO` |
| User | login | `john.doe` |
| Group / Role | name | `Developers` / `Developer` |
| Everything else | opaque `id` from API responses | `2-7` |

## Gotchas

- **IMPORTANT — `project add` requires `--leader`**: the YouTrack API rejects project creation without it. Use `user current` to get the current user's login if none is specified.
- **`issue command` logins**: the CLI auto-wraps logins containing `.`, `-`, or `@` in braces for YouTrack's command parser (`alice.qa` → `{alice.qa}`). Write the login as-is.
- **`--asUser <login>`**: the target user must have at least a Contributor role in the project.

## Commits

Use Conventional Commit prefixes: `feat:`, `fix:`, `chore:`, `docs:`. Keep commits focused; don't mix refactors with behavior changes.
