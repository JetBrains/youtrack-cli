# AGENTS.md

Go CLI for interacting with the YouTrack REST API. Built with Cobra. Designed for use by AI agents and developers.

## Agent Skill

A full YouTrack interaction guide is available as an agent skill — load it when the user asks to manage YouTrack entities (issues, articles, projects, users, sprints, etc.):

**`.agents/skills/youtrack-cli/SKILL.md`** — command reference, common workflows, output formats, and YouTrack-specific gotchas.

## Development Commands

```bash
make build                                           # compile ./youtrack binary
make check                                          # fmt-check + lint + build + test — run before committing/pushing
make test                                            # Dockerized YouTrack E2E suite against a real local instance
go run . issue list --query "assignee: me"           # run without building
go test ./test/ -run TestGetIssue -v                 # run a single test by name
```

## Environment

```bash
export YOUTRACK_BASE_URL=https://your-instance.youtrack.cloud
export YOUTRACK_TOKEN=perm:...
```

## Project Structure

| Path | Role |
|---|---|
| `main.go` | Entrypoint — calls `cmd.Execute()` |
| `go.mod` / `go.sum` | Module definition (module: `youtrack-cli`, Go 1.26) |
| `Makefile` | Build, test, and dist targets |
| `api/client.go` | `Client` struct, `Get`/`Post`/`Put`/`Delete` HTTP primitives |
| `api/types.go` | Shared struct types (Issue, Project, User, Comment, etc.) |
| `api/issues.go` | Issue and comment API methods |
| `api/projects.go` | Project API methods |
| `api/users.go` | User API methods |
| `cmd/youtrack.go` | Cobra root command, exports `NewRootCmd()`, global flags, subcommand registration |
| `internal/client.go` | `GetClient()` helper — reads flags and env vars |
| `cmd/issue/` | Issue subcommands (23 files) |
| `cmd/project/` | Project subcommands (11 files) |
| `cmd/user/` | User subcommands (13 files) |
| `internal/output/` | JSON and table output formatters |
| `test/` | End-to-end tests against a real YouTrack instance (`package api_test`) |

## Command Registration

Add a new file in `cmd/` (e.g., `cmd/issue_foo.go`). Create a constructor `newIssueFooCmd() *cobra.Command` and register it in the parent command's constructor (e.g., `NewCmd()` in `cmd/issue/issue.go`). Parents with grouped `--help` menus (root, `issue`, `project`, `user`) register via `internal.AddToGroup(cmd, "<groupID>", ...)` so the command lands in a help group; flat parents call `cmd.AddCommand(...)` directly.

- Use `internal.GetClient(cmd)` to obtain the API client.
- Return errors from `RunE` — the root command prints them to stderr.
- Use `output.Print(cmd, data)` for all output — it respects `--json`/`--plain`.

## When Adding a Command

- Add a usage example and table row to `README.md`.
- Add to `.agents/skills/youtrack-cli/SKILL.md` if part of a non-trivial workflow.

## Coding Conventions

- `gofmt` formatting — run automatically by most editors with gopls.
- Command names: lowercase verbs; canonical set is `view`, `list`, `create`, `update`, `delete` (`get`/`add` live on as aliases — don't use them for new commands).
- `add` is reserved for associating existing entities or registering config (`issue tag add`, `project team add-user`, `instance add`); anything that makes a new record uses `create`.
- Go identifiers: camelCase for unexported, PascalCase for exported.
- Error strings: lowercase, no trailing period (Go convention).
- All output goes through `output.Print(cmd, data)` — never `fmt.Println` directly in a command handler.

## Testing

CLI behavior is covered exclusively by end-to-end tests: no mocks, no fake transport, and no
`api.Client` conformance suite running in parallel with CLI coverage. The subject under test is the
compiled `youtrack` binary against a real YouTrack instance.

Two narrow, deliberately scoped exceptions live outside the E2E flow. Neither uses mocks or fake
transport, and neither is a second product surface asserting client behavior:

- **Pure-function unit tests** (`api/*_test.go`, `internal/*_test.go`) — table tests for pure, branchy
  string helpers (`mergeFieldsParam`, `WrapUserValues`, `expandFields`, `SplitFields`) that
  E2E cannot exercise exhaustively. They touch no client, no transport, and no network, and live in the
  package under test. CI runs them via `go test ./api/ ./internal/...`.
- **API drift detection** (`test/drift_test.go`, `TestAPIDrift`) — gated on `requireE2EEnv` and run
  inside the E2E job, it diffs our Go structs against the live server's OpenAPI spec to catch contract
  changes. It asserts no client behavior; it is a static schema-vs-struct guard, not a conformance suite.

Run the full suite with `make test`. It boots or reuses the local Dockerized YouTrack harness, bootstraps credentials,
and runs the entire `./test` package against that live instance. Each test is self-contained: it creates all needed
entities and registers `t.Cleanup()` teardown. Helper functions in `test/helpers_test.go`: `newTestClient(t)`,
`createTempProject(t, c)`, `runCmd(t, args...)`.

Tests may use `api.Client` for fixture setup, teardown, or external-state verification, but not as a second product
surface with its own parallel contract suite. Keep CLI coverage centered on observable CLI behavior.

## ID Conventions

| Entity | Identifier | Example |
|---|---|---|
| Issue | `<PROJECT>-<n>` | `FOO-42` |
| Project | short name | `FOO` |
| User | login | `john.doe` |
| Group / Role | name | `Developers` / `Developer` |
| Everything else | opaque `id` from API responses | `2-7` |

## Gotchas

- **IMPORTANT — `project create` requires `--leader`**: the YouTrack API rejects project creation without it. Use `user current` to get the current user's login if none is specified.
- **`project team add-user` / `project team add-group` expect opaque IDs** (e.g. `2-9`, `4-1`), not logins or group names. When creating entities, capture the `id` field from the JSON response and pass that. When using existing entities, retrieve the ID first with `user view <login>` or `group view <name>`.
- **`issue command` logins**: the CLI auto-wraps logins containing `.`, `-`, or `@` in braces for YouTrack's command parser (`alice.qa` → `{alice.qa}`). Write the login as-is.
- **`--asUser <login>`**: the target user must have at least a Contributor role in the project.

## Commits

Use Conventional Commit prefixes: `feat:`, `fix:`, `chore:`, `docs:`. Keep commits focused; don't mix refactors with behavior changes.

Always run `make check` before committing or pushing. It runs formatting, linting, build, and the full E2E test suite. Do not push code that fails `make check`.
