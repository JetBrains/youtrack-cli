# youtrack-cli

Create issues, manage projects, and automate YouTrack — without leaving your terminal.

A Go CLI for interacting with YouTrack via its REST API. Designed for developers and AI agents.

> **Security notice:** YouTrack does not currently support granular token permissions. Any token grants full account access. Review [docs/security.md](docs/security.md) before using in pipelines or agent integrations.

## Installation

> The repository is currently internal. Use one of the options below.

**Via GitHub CLI** (requires [`gh`](https://cli.github.com/) authenticated with `gh auth login`):

```bash
gh release download --repo JetBrains/youtrack-cli --pattern install.sh -O - | bash
```

**Via `go install`** (requires [Go 1.21+](https://go.dev/dl/)):

```bash
go install github.com/JetBrains/youtrack-cli@latest
```

**Windows / manual:** Download from [Releases](https://github.com/JetBrains/youtrack-cli/releases/latest), extract, add `youtrack` to `PATH`.

## Quick Start

```bash
# Authenticate once — saves credentials to ~/.youtrack-cli-config.json
youtrack instance add
# YouTrack base URL: https://your-instance.youtrack.cloud
# Instance alias [your-instance]: mywork
# Select authentication method: ...
# ✓ Authenticated as Jane Doe (jane.doe)

# Verify connection
youtrack user current

# Find your open issues
youtrack issue list --query "assignee: me State: Open"

# Create an issue
youtrack issue add --project MYPROJ --summary "Bug title"

# Apply a command
youtrack issue command MYPROJ-42 "State In Progress"
```

> **CI / scripting:** skip the interactive setup and export env vars instead:
> ```bash
> export YOUTRACK_BASE_URL=https://your-instance.youtrack.cloud
> export YOUTRACK_TOKEN=perm:...
> ```

## Authentication

For a single instance, env vars are enough — see the CI/scripting note in Quick Start above.

**Multiple instances** — use `instance add` to register each under a named alias:

```bash
youtrack instance add
youtrack instance login https://your-instance.youtrack.cloud   # browser OAuth sign-in (~1h token)
youtrack instance login mywork        # renew an existing alias's token via the browser
youtrack instance use personal        # switch active context
youtrack instance list                # show all instances
youtrack instance remove work         # remove an instance
youtrack instance update-token work   # replace the saved token for an instance
youtrack auth refresh-token           # replace token for the active instance
youtrack auth logout                  # remove active instance credentials
```

Auth methods: **Browser SSO** (`instance login` — OAuth 2.0 + PKCE against JetBrains Hub, no paste needed, ~1-hour session token), **Basic** (login + password), **Web browser** (opens token page), or **Paste token** (fallback for cloud/SSO).

Credentials are saved to `~/.youtrack-cli-config.json`. Env vars are a single-instance fallback.

### Credential Precedence

The CLI resolves credentials in this order (highest wins):

1. `--base-url` / `--token` flags — per-invocation override
2. `--server <alias>` flag — selects a named profile from `~/.youtrack-cli-config.json`
3. `YOUTRACK_BASE_URL` / `YOUTRACK_TOKEN` env vars — explicit, process-scoped (CI/Docker)
4. Active instance in `~/.youtrack-cli-config.json` — convenience fallback

**Gotcha:** Env vars (Tier 3) always win over the active config instance (Tier 4).

## Read-Only Mode

Set `YOUTRACK_RO=1` to block all write operations — useful for AI agents, CI read jobs, or safe exploration:

```bash
export YOUTRACK_RO=1
youtrack issue list   # works
youtrack issue add …  # error: read-only mode is enabled (YOUTRACK_RO=1): write operations are blocked
```

See [docs/security.md](docs/security.md) for the full token disclaimer and mutability reference.

## Commands

| Group | Subcommands |
|---|---|
| `auth` | `get-token` (`login`), `refresh-token`, `logout`, `status` |
| `instance` | `add`, `login`, `list`, `use`, `remove`, `update-token` |
| `issue` | `get` (`view`), `list`, `add` (`create`), `update`, `delete`, `command`, `count`, `suggest`, `assist`, `activity`, `sprint`, `vcs` |
| `issue comment` | `list`, `add`, `update`, `delete`, `pin`, `visibility` |
| `issue command` | `suggest` |
| `issue link` | `list`, `add`, `remove` |
| `issue link type` | `list`, `add` |
| `issue attachment` | `list`, `add`, `delete` |
| `issue field` | `get`, `set` |
| `issue reaction` | `add`, `delete` |
| `issue draft` | `create`, `update`, `submit` |
| `issue` (actions) | `assign`, `unassign`, `watch`, `unwatch`, `vote`, `unvote`, `tag`, `visibility`, `workitem add` |
| `project` | `list`, `get` (`view`), `add` (`create`), `delete`, `archive` |
| `project team` | `add-user`, `add-group` |
| `project field` | `list`, `add` |
| `project template` | `list`, `convert` |
| `project timetrack` | `set` |
| `project workflow` | `add` |
| `user` | `list`, `get` (`view`), `current` (`me`), `add` (`create`), `delete`, `ban`, `unban`, `pin-project` |
| `user drafts` | `list`, `delete`, `articles` |
| `user locale` | `set` |
| `user timezone` | `get`, `set` |
| `user profile` | `general`, `appearance`, `notifications`, `articles`, `issues-list`, `grazie`, `tips`, `property` |
| `board` | `list`, `get` (`view`), `add` (`create`), `update`, `delete` |
| `board sprint` | `list`, `get`, `add` |
| `whiteboard` | `list`, `get`, `add` (`create`), `delete`, `reload` |
| `whiteboard node` | `list`, `get`, `add` |
| `whiteboard links` | list links |
| `whiteboard link` | create a link between two nodes |

## Examples

```bash
# Issues — CRUD
youtrack issue get MYPROJ-42
youtrack issue list --query "assignee: me State: Open" --limit 50
youtrack issue add --project MYPROJ --summary "Login button unresponsive on Safari"
youtrack issue add --project MYPROJ --summary "Child issue" --parent MYPROJ-10
youtrack issue update MYPROJ-42 --summary "New title"
youtrack issue delete MYPROJ-42
youtrack issue count --query "State: Open"

# Issues — workflow and actions
youtrack issue command MYPROJ-42 "State In Progress"
youtrack issue command MYPROJ-42 "assignee alice.doe"
youtrack issue command suggest MYPROJ-42 "State "
youtrack issue assign MYPROJ-42 john.doe
youtrack issue watch MYPROJ-42
youtrack issue vote MYPROJ-42

# Comments
youtrack issue comment list MYPROJ-42
youtrack issue comment add MYPROJ-42 --text "Reproducing locally."
youtrack issue comment update MYPROJ-42 <commentId> --text "Updated text"
youtrack issue comment delete MYPROJ-42 <commentId>

# Links and attachments
youtrack issue link list MYPROJ-42
youtrack issue link add MYPROJ-42 --target MYPROJ-10 --type <linkTypeId>
youtrack issue attachment add MYPROJ-42 ./screenshot.png
youtrack issue field set MYPROJ-42 Priority Critical

# Drafts
youtrack issue draft create --project MYPROJ --format json
youtrack issue draft update <draftId> --summary "New title"
youtrack issue draft submit <draftId>

# Work items
youtrack issue workitem add MYPROJ-42 --minutes 90 --text "Investigation"

# Projects
youtrack project list
youtrack project add --shortName MYPROJ --name "My Project" --leader john.doe
youtrack project archive MYPROJ
youtrack project team add-user MYPROJ 2-9   # opaque ID from: youtrack user get john.doe | jq .id
youtrack project field list MYPROJ
youtrack project timetrack set MYPROJ --enabled

# Users
youtrack user current
youtrack user list --query "john"
youtrack user add --login john.doe --fullName "John Doe" --email john.doe@example.com --password "Test1234!"
youtrack user ban john.doe
youtrack user delete john.doe --successor admin
youtrack user timezone set john.doe "Europe/Berlin"
```

## Global Flags

| Flag | Description |
|---|---|
| `--json` / `-j` | Force JSON output |
| `--plain` | Force tab-separated output, no headers |
| `--fields FIELDS` / `-f` | Field projection: `--fields id,summary,reporter` |
| `--server ALIAS` / `-s` | Use a named instance for this invocation |
| `--base-url URL` | Override `YOUTRACK_BASE_URL` |
| `--token TOKEN` | Override `YOUTRACK_TOKEN` |
| `--verbose` | Print HTTP requests/responses to stderr |
| `--quiet` | Suppress tips and hints (useful for scripting) |

**Output defaults:** table when stdout is a TTY; JSON when piped or redirected.

### --fields projection

```bash
youtrack issue list --fields id,summary,reporter
youtrack issue list --fields 'comments(id,text)'
youtrack issue list --fields project.shortName
```

Nested entity names auto-expand: `reporter` → `reporter(id,login,fullName)`.

## AI Agent Integration

A skill file for AI coding agents is included at `.agents/skills/youtrack-cli/SKILL.md`. Claude Code picks it up automatically from the repo root.

To install it into another project with the [GitHub CLI](https://cli.github.com/manual/gh_skill_install):

```bash
gh skill install JetBrains/youtrack-cli youtrack-cli --allow-hidden-dirs --agent claude-code
```

`--allow-hidden-dirs` is required because the skill lives under `.agents/` (gh skips dot-prefixed directories by default). Add `--scope user` to install once for all projects instead of just the current repo.

## Further Reading

- [Security & Token Permissions](docs/security.md) — token disclaimer, `YOUTRACK_RO`, mutability reference
- [Development Guide](docs/development.md) — build targets, E2E harness, project structure, adding commands

## Contributing

Open an issue before submitting a pull request for significant changes.

## License

Apache-2.0
