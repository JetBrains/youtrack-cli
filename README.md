# youtrack-cli

Create issues, manage projects, and automate YouTrack – without leaving your terminal.

A Go CLI for interacting with YouTrack via its REST API. Designed for use by developers and AI agents.

## Installation

> The repository is currently internal. Use one of the options below.

**Via GitHub CLI** — requires [GitHub CLI](https://cli.github.com/) installed and authenticated (`gh auth login`):
```bash
gh release download --repo JetBrains/youtrack-cli --pattern install.sh -O - | bash
```

**Via `go install`** — requires [Go 1.21+](https://go.dev/dl/), binary is placed in `$GOPATH/bin` (make sure it's on your `$PATH`):
```bash
go install github.com/JetBrains/youtrack-cli@latest
```

**Windows / manual:** Download the archive for your platform from [Releases](https://github.com/JetBrains/youtrack-cli/releases/latest), extract, and add `youtrack` to your PATH.

## Quick Start

```bash
# Authenticate (set credentials once)
export YOUTRACK_BASE_URL=https://your-instance.youtrack.cloud
export YOUTRACK_TOKEN=perm:...

# Verify connection
youtrack user current

# Find your open issues
youtrack issue list --query "assignee: me State: Open"

# Create an issue
youtrack issue add --project MYPROJ --summary "Bug title"

# Apply a command
youtrack issue command MYPROJ-42 "State In Progress"
```

## Setup

### Build the binary

```bash
make build
cp youtrack /usr/local/bin/
```

Or cross-compile for your target platform:

```bash
make dist   # produces binaries in dist/ for linux-amd64, darwin-arm64, darwin-amd64, windows-amd64
```

### Set credentials

**Single instance** — export environment variables or write to `~/.youtrack-cli-env`:

```bash
export YOUTRACK_BASE_URL=https://your-instance.youtrack.cloud
export YOUTRACK_TOKEN=perm:...
```

**Multi-instance** — use `instance add` to authenticate and register each instance under a named alias:

```bash
youtrack instance add
# YouTrack base URL: https://work.youtrack.cloud
# Instance alias [work]: work
# Select authentication method: ...
# ✓ Authenticated as Jane Doe (jane.doe)
#   Saved as: "work" (~/.youtrack-cli-config.json)

youtrack instance add
# YouTrack base URL: https://personal.youtrack.cloud
# Instance alias [personal]: personal
# ...

youtrack instance use personal   # switch active context
youtrack instance list            # show all instances with auth status
youtrack instance remove work     # remove an instance
youtrack auth logout              # remove active instance credentials
```

If a `~/.youtrack-cli-env` file is present when running `auth get-token` for the first time, the CLI offers to import its credentials as a named instance.

Three auth methods are available:
- **Basic auth** — login + password via Hub API. Requires the Hub basic-auth endpoint to be enabled.
- **Web browser** — the CLI detects your Hub URL, opens the token management page in a browser, then asks you to paste the token.
- **Paste token directly** — shows the token page URL for you to open manually, then asks you to paste the token. Used automatically as a fallback when basic auth is unavailable (e.g. cloud or SSO-only instances).

Credentials are saved to `~/.youtrack-cli-config.json` by default (`--save=false` to skip).
Env vars `YOUTRACK_BASE_URL` / `YOUTRACK_TOKEN` still work as a single-instance fallback.

## Commands

| Group | Subcommands |
|---|---|
| `auth` | `get-token`, `logout`, `status` |
| `instance` | `add`, `list`, `use`, `remove` |
| `issue` | `get`, `list`, `add`, `update`, `delete`, `command`, `comment list`, `comment add`, `comment delete` |
| `project` | `list`, `get`, `add`, `delete` |
| `user` | `list`, `get`, `current`, `add`, `delete`, `ban`, `unban` |

## Examples

```bash
# Issues
youtrack issue get MYPROJ-42
youtrack issue list --query "assignee: me State: Open" --limit 50
youtrack issue add --project MYPROJ --summary "Login button unresponsive on Safari"
youtrack issue add --project MYPROJ --summary "RFC: Auth redesign" --description "See attached doc."
youtrack issue update MYPROJ-42 --summary "New title"
youtrack issue delete MYPROJ-42
youtrack issue command MYPROJ-42 "State In Progress"
youtrack issue command MYPROJ-42 "assignee alice.doe"

# Comments
youtrack issue comment list MYPROJ-42
youtrack issue comment add MYPROJ-42 --text "Reproducing locally."
youtrack issue comment delete MYPROJ-42 <commentId>

# Projects
youtrack project list
youtrack project get MYPROJ
youtrack project add --shortName MYPROJ --name "My Project" --leader john.doe
youtrack project delete MYPROJ

# Users
youtrack user list --query "john"
youtrack user get john.doe
youtrack user current
youtrack user add --login john.doe --fullName "John Doe" --email john.doe@example.com --password "Test1234!"
youtrack user ban john.doe
youtrack user delete john.doe --successor admin
```

## Global flags

| Flag | Description |
|---|---|
| `--format json\|table` | Output format (default: `json`) |
| `--base-url URL` | Override `YOUTRACK_BASE_URL` |
| `--token TOKEN` | Override `YOUTRACK_TOKEN` |
| `--verbose` | Print HTTP requests/responses to stderr |

## AI Agent Integration

A skill file is included for AI coding agents (Claude Code, Cursor, and others). It provides command examples, workflows, and YouTrack-specific gotchas.

Install for Claude Code:

```bash
# The skill is already in .agents/skills/youtrack-cli/SKILL.md
# Claude Code picks it up automatically from the repo root
```

Or reference it manually in your agent's context:

```
.agents/skills/youtrack-cli/SKILL.md
```

## Build

```bash
make build           # compile ./youtrack binary
make fmt             # apply gofmt to the repo
make fmt-check       # fail if any Go file is not gofmt'd
make lint            # run a pinned golangci-lint via go run
make check           # fmt-check + lint + build + test (run before pushing)
make test            # boot local Docker YouTrack and run the full E2E suite
make e2e-up          # start or reuse the local Docker YouTrack instance
make e2e-down        # stop and remove the local Docker YouTrack instance
make dist            # cross-compile for linux-amd64, darwin-arm64, darwin-amd64, windows-amd64
```

## Development

The project is organized in three layers:

- `api/` contains the typed YouTrack REST client.
- `cmd/` contains Cobra command wiring and request/response shaping.
- `test/` contains end-to-end tests that exercise the compiled CLI against a real YouTrack instance.

## Docker E2E

The E2E harness uses the official `jetbrains/youtrack` Docker image and stores runtime state under `.e2e/`.
It always starts a local YouTrack instance at `http://127.0.0.1:8088` and ignores any ambient `YOUTRACK_BASE_URL` in the caller shell.

Prerequisites: `docker`, `curl`, `jq`, and `go` must be available in `PATH`.

```bash
make e2e-up
make test
make e2e-down
```

Useful environment variables:

| Variable | Description |
|---|---|
| `YOUTRACK_E2E_RESET=1` | Recreate the local YouTrack state from scratch before startup |
| `YOUTRACK_E2E_VERSION=2026.1.12024` | Override the Docker image tag used for the local YouTrack E2E instance |
| `YOUTRACK_E2E_LICENSE_NAME` | Override the license owner name passed to YouTrack during `configure` |
| `YOUTRACK_E2E_LICENSE_KEY` | Override the license key passed to YouTrack during `configure` |

When the Docker harness completes bootstrap successfully, it writes `.e2e/youtrack.env` with:

- `YOUTRACK_BASE_URL`
- `YOUTRACK_TOKEN`

`make test` sources that file automatically before running the E2E suite. If you want to run the CLI manually against the local harness after `make e2e-up`, source it in your shell first:

```bash
set -a
source .e2e/youtrack.env
set +a
```

## Contributing

Contributions are welcome. Please open an issue before submitting a pull request for significant changes.

## License

Apache-2.0
