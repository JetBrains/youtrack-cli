# Development Guide

## Build

```bash
make build           # compile ./youtrack binary
make dist            # cross-compile: linux-amd64, darwin-arm64, darwin-amd64, windows-amd64
cp youtrack /usr/local/bin/
```

## Code Quality

```bash
make fmt             # apply gofmt to the repo
make fmt-check       # fail if any file is not gofmt'd
make lint            # run golangci-lint via go run
make check           # fmt-check + lint + build + test — run before committing
```

Always run `make check` before pushing. It runs formatting, linting, build, and the full E2E suite.

## Testing

CLI behavior is tested end-to-end only — no mocks, no fake transport. The subject under test is the compiled `youtrack` binary against a real YouTrack instance running in Docker.

Two scoped exceptions complement the E2E suite:

- Pure-function unit tests live in `api/*_test.go` and `internal/*_test.go`; run them with `go test ./api/ ./internal/...`.
- API drift detection lives in `test/drift_test.go` (`TestAPIDrift`, gated on the E2E environment), which diffs the Go structs against the live OpenAPI spec.

CI also runs a `govulncheck` scan.

```bash
make e2e-up          # start (or reuse) the local Docker YouTrack instance
make test            # run the full E2E suite
make e2e-down        # stop and remove the local instance
```

Prerequisites: `docker`, `curl`, `jq`, and `go` on `PATH`.

The harness starts YouTrack at `http://127.0.0.1:8088` and ignores any ambient `YOUTRACK_BASE_URL`. When bootstrap succeeds it writes `.e2e/youtrack.env` — source it to run the CLI manually against the local instance:

```bash
set -a
source .e2e/youtrack.env
set +a
youtrack issue list
```

### E2E environment variables

| Variable | Description |
|---|---|
| `YOUTRACK_E2E_RESET=1` | Recreate local YouTrack state from scratch |
| `YOUTRACK_E2E_VERSION=2026.2.17765` | Override the Docker image tag |
| `YOUTRACK_E2E_LICENSE_NAME` | License owner name for `configure` |
| `YOUTRACK_E2E_LICENSE_KEY` | License key for `configure` |

### Writing tests

Each test is self-contained: it creates all entities it needs and registers `t.Cleanup()` teardown. Use helpers from `test/helpers_test.go`:

- `newTestClient(t)` — returns a preconfigured API client
- `createTempProject(t, c)` — creates a temp project, cleaned up after the test
- `runCmd(t, args...)` — runs the compiled CLI binary

Tests may use `api.Client` for fixture setup and teardown, but not as a separate conformance suite. Keep tests centered on observable CLI behavior.

`TestEmbeddedCommands` builds `test/testdata/embedding` as a separate CLI using the
public embedding API, then runs it against the same live YouTrack instance. See
[Embedding the CLI](embedding.md) for the supported integration points.

Run a single test by name:

```bash
go test ./test/ -run TestGetIssue -v
```

## Project Structure

| Path | Role |
|---|---|
| `main.go` | Entrypoint — calls `cmd.Execute()` |
| `api/client.go` | HTTP primitives (`Get`, `Post`, `Put`, `Delete`) |
| `api/types.go` | Shared struct types (Issue, Project, User, etc.) |
| `api/issues.go` | Issue and comment API methods |
| `api/projects.go` | Project API methods |
| `api/users.go` | User API methods |
| `cmd/youtrack.go` | Cobra root command, global flags, subcommand registration |
| `internal/client.go` | `GetClient()` helper — resolves flags, config, env vars |
| `cmd/issue/` | Issue subcommands |
| `cmd/project/` | Project subcommands |
| `cmd/user/` | User subcommands |
| `internal/output/` | JSON and table output formatters |
| `test/` | End-to-end tests (`package api_test`) |

## Adding a Command

1. Create a new file in the relevant `cmd/` subdirectory (e.g. `cmd/issue/foo.go`).
2. Implement `newIssueFooCmd() *cobra.Command`.
3. Register it in the parent's `NewCmd()`: grouped parents (root, `issue`, `project`, `user`) use `internal.AddToGroup(cmd, "<groupID>", newIssueFooCmd())` so the command lands in a help group; flat parents call `cmd.AddCommand(newIssueFooCmd())`.
4. Use `internal.GetClient(cmd)` to obtain the API client.
5. Return errors from `RunE` — the root command prints them to stderr.
6. Use `output.Print(cmd, data)` for all output (respects `--json`/`--plain`, with TTY auto-detection).
7. Add a usage example to `README.md` and update `.agents/skills/youtrack-cli/SKILL.md` if part of a non-trivial workflow.

## Coding Conventions

- `gofmt` formatting.
- Command names: lowercase verbs; canonical set is `view`, `list`, `create`, `update`, `delete` (`get`/`add` live on as aliases — don't use them for new commands).
- `add` is reserved for associating existing entities or registering config (`issue tag add`, `project team add-user`, `instance add`); anything that makes a new record uses `create`.
- Go identifiers: camelCase for unexported, PascalCase for exported.
- Error strings: lowercase, no trailing period.
- All output goes through `output.Print(cmd, data)` — never `fmt.Println` in a command handler.
- Commits: `feat:`, `fix:`, `chore:`, `docs:` prefixes. Keep commits focused.
