# youtrack-cli

Create issues, manage projects, and automate YouTrack – without leaving your terminal.

A Go CLI for interacting with YouTrack via its REST API. Designed for use by developers and AI agents.

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

Export environment variables:

```bash
export YOUTRACK_BASE_URL=https://your-instance.youtrack.cloud
export YOUTRACK_TOKEN=perm:...
```

Or write them to `~/.youtrack-cli-env` (one `KEY=VALUE` per line) — loaded automatically on every run.

## Commands

| Group | Subcommands |
|---|---|
| `issue` | `get`, `list`, `add`, `update`, `delete`, `command`, `comment list`, `comment add`, `comment delete` |
| `project` | `list`, `get`, `add`, `delete` |
| `user` | `list`, `get`, `current`, `add` |

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
youtrack user add --login john.doe --fullName "John Doe" --email john.doe@example.com
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
make test            # go test ./...
make test-integration # integration tests (requires YOUTRACK_BASE_URL + YOUTRACK_TOKEN)
make dist            # cross-compile for linux-amd64, darwin-arm64, darwin-amd64, windows-amd64
```

## Contributing

Contributions are welcome. Please open an issue before submitting a pull request for significant changes.

## License

Apache-2.0
