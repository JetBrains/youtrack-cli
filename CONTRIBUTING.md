# Contributing to youtrack-cli

## Prerequisites

- **Go 1.26+** — check with `go version`
- **Docker** — required for E2E tests (`docker version`)
- **`curl`, `jq`** — used by the E2E harness
- **`goreleaser`** — optional, for running local snapshot builds (`brew install goreleaser/tap/goreleaser-pro`)

## Building from Source

```bash
git clone https://github.com/JetBrains/youtrack-cli.git
cd youtrack-cli
make build          # compiles ./youtrack
```

The binary is written to `./youtrack` in the repo root. Copy to your PATH:

```bash
cp youtrack /usr/local/bin/
```

## Running Checks

Run this before every commit or push. It executes formatting check, lint, build, and the full E2E test suite:

```bash
make check
```

Individual targets:

```bash
make fmt-check      # fail if any Go file is not gofmt'd
make fmt            # apply gofmt to all files
make lint           # run golangci-lint
make build          # compile ./youtrack
make test           # run E2E suite (requires Docker)
```

## End-to-End Tests

The E2E suite runs against a real YouTrack instance managed by Docker. It does not use mocks.

```bash
make e2e-up         # start the local YouTrack Docker instance (first run takes ~2 min)
make test           # run the full suite
make e2e-down       # stop and remove the instance
make e2e-logs       # collect logs (useful after failures)
```

To reset YouTrack state from scratch:

```bash
YOUTRACK_E2E_RESET=1 make test
```

To run a single test:

```bash
go test ./test/ -run TestGetIssue -v
```

After `make e2e-up`, the harness writes credentials to `.e2e/youtrack.env`. Source it to use the CLI against the local instance manually:

```bash
set -a; source .e2e/youtrack.env; set +a
youtrack user current
```

## Local Snapshot Build

To validate the GoReleaser config and cross-compile all platforms locally (requires goreleaser-pro):

```bash
make snapshot
```

Output goes to `dist/`. Clean up with `make clean`.

## Commit Conventions

Use [Conventional Commits](https://www.conventionalcommits.org/):

| Prefix | When to use |
|---|---|
| `feat:` | New command or user-visible behavior |
| `fix:` | Bug fix |
| `chore:` | Tooling, deps, config (no behavior change) |
| `docs:` | README, comments, examples only |
| `ci:` | Changes to GitHub Actions workflows |

Keep commits focused — don't mix refactors with behavior changes.

## Adding a Command

1. Create a new file in `cmd/` (e.g., `cmd/issue/foo.go`)
2. Define `newIssueFooCmd() *cobra.Command`
3. Register it in the parent `NewCmd()` constructor
4. Use `internal.GetClient(cmd)` for the API client
5. Return errors from `RunE`; use `output.Print(cmd, data)` for all output
6. Add a usage example and table row to `README.md`

## Opening a Pull Request

- Open an issue before submitting a PR for significant changes
- `make check` must pass before requesting review
- Include E2E test coverage for new commands — see `test/` for examples
