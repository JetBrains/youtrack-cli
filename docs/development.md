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

Tests are end-to-end only — no mocks, no fake transport. The subject under test is the compiled `youtrack` binary against a real YouTrack instance running in Docker.

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
| `YOUTRACK_E2E_VERSION=2026.1.12024` | Override the Docker image tag |
| `YOUTRACK_E2E_LICENSE_NAME` | License owner name for `configure` |
| `YOUTRACK_E2E_LICENSE_KEY` | License key for `configure` |

### Writing tests

Each test is self-contained: it creates all entities it needs and registers `t.Cleanup()` teardown. Use helpers from `test/helpers_test.go`:

- `newTestClient(t)` — returns a preconfigured API client
- `createTempProject(t, c)` — creates a temp project, cleaned up after the test
- `runCmd(t, args...)` — runs the compiled CLI binary

Tests may use `api.Client` for fixture setup and teardown, but not as a separate conformance suite. Keep tests centered on observable CLI behavior.

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
| `internal/env.go` | `LoadEnvFile()` — loads `~/.youtrack-cli-env` |
| `cmd/issue/` | Issue subcommands |
| `cmd/project/` | Project subcommands |
| `cmd/user/` | User subcommands |
| `cmd/output/` | JSON and table output formatters |
| `test/` | End-to-end tests (`package api_test`) |

## Adding a Command

1. Create a new file in the relevant `cmd/` subdirectory (e.g. `cmd/issue/foo.go`).
2. Implement `newIssueFooCmd() *cobra.Command`.
3. Register it: call `cmd.AddCommand(newIssueFooCmd())` in the parent's `NewCmd()`.
4. Use `internal.GetClient(cmd)` to obtain the API client.
5. Return errors from `RunE` — the root command prints them to stderr.
6. Use `output.Print(cmd, data)` for all output (respects `--format`).
7. Add a usage example to `README.md` and update `.agents/skills/youtrack-cli/SKILL.md` if part of a non-trivial workflow.

## Coding Conventions

- `gofmt` formatting.
- Command names: lowercase verbs (`get`, `list`, `add`, `update`, `delete`).
- Go identifiers: camelCase for unexported, PascalCase for exported.
- Error strings: lowercase, no trailing period.
- All output goes through `output.Print(cmd, data)` — never `fmt.Println` in a command handler.
- Commits: `feat:`, `fix:`, `chore:`, `docs:` prefixes. Keep commits focused.
