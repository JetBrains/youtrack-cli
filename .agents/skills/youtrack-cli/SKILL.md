---
name: youtrack-cli
version: "2.0.0"
author: JetBrains
description: CRUD for YouTrack entities (issues, projects, users, comments) via the youtrack-cli Go CLI. Use when the user asks to manage YouTrack data, run queries, or configure a YouTrack instance.
allowed-tools: Bash
---

## Auth

### Check auth status
```bash
youtrack auth status
```
**Single-instance (no config file):** Returns `{"url":"...","login":"...","fullName":"...","status":"authenticated"}`
**Multi-instance (config file present):** Returns a JSON array `[{"alias":"work","url":"...","login":"...","fullName":"...","status":"authenticated","active":true}, ...]`
Status values per instance: `"authenticated"` | `"invalid token: ..."`.
Always exits 0 — check the `status` field programmatically.

### Authenticate interactively
```bash
youtrack auth get-token [--alias <name>]
```
`--alias` sets the instance name (derived from URL hostname if omitted: `work.youtrack.cloud` → `work`).
Prompts for base URL (if `YOUTRACK_BASE_URL` not set), then auth method:
- **1 — Basic auth**: prompts for login + password, creates a Hub permanent token via Hub API.
  - The interactive password prompt requires a TTY. In scripts, pass `--password <pass>` as a flag.
  - Use `--scope <service-names>` (comma-separated) to request specific Hub services. Default: YouTrack + YouTrack Administration.
  - If the Hub endpoint returns HTML (cloud/SSO-only instances), falls back automatically to option 3.
- **2 — Web browser**: detects Hub URL via `/.well-known/oauth-protected-resource/mcp`, opens the token page in a browser, prompts for paste, verifies before saving.
  - Internal Hub → `{youtrackURL}/users/me?tab=account-security`
  - External Hub → `{hubURL}/users/me?tab=authentication`
  - If the browser cannot be opened automatically, the CLI prints the token page URL; open it manually.
- **3 — Paste token directly**: shows the correct token page URL without opening a browser. Token is verified before saving.

Saves credentials to `~/.youtrack-cli-config.json` by default (`--save=false` to skip).
The CLI auto-prepends `https://` if the base URL is entered without a scheme.
If `~/.youtrack-cli-env` exists with credentials and the config is empty, a one-time import prompt is shown.

### Manage instances
```bash
youtrack instance list                        # list all configured instances (alias, URL, active marker)
youtrack instance use <alias>                 # switch active context
youtrack instance remove <alias>              # remove an instance (prompts for confirmation)
youtrack instance remove --force <alias>      # skip confirmation
youtrack auth logout                          # remove active instance (or --instance <alias> for a specific one)
```

### Refreshing a Token

To replace the saved token for a configured instance (e.g. when it expires):

```bash
youtrack auth refresh-token [<alias>]
youtrack instance update-token <alias>
```

If no alias is given, `auth refresh-token` uses the active instance. Presents the same three
auth methods as `auth get-token`. The alias and URL are preserved; only the token is replaced.

## Setup (run silently before every task)

1. **CLI available?** `which youtrack` — if not found, run `make build && cp youtrack /usr/local/bin/` from the repo root.
2. **Credentials set?** Check `~/.youtrack-cli-config.json` (multi-instance) or `~/.youtrack-cli-env` / `YOUTRACK_BASE_URL` + `YOUTRACK_TOKEN` (single-instance). If missing, guide the user to run `youtrack auth get-token`. To check the active instance: `youtrack instance list`.
3. **Connectivity OK?** `youtrack user me` — stop and report if it fails.

## Behaviour rules

- **No confirmations.** Execute all actions autonomously — `youtrack` commands, directory creation, sourcing `~/.youtrack-cli-env`. Gather ambiguous params upfront, then run to completion.
- **Summarise, don't narrate.** Concise summary after completion; never dump raw JSON.
- **Output format.** Default output is table when stdout is a TTY, JSON when piped or redirected. Use `--json` to force JSON, `--plain` for tab-separated no-header output.

## Quick Start

```bash
youtrack user me                                                               # verify auth
youtrack issue list --query "assignee: me State: Open"                        # find issues
youtrack issue create --project MYPROJ --summary "Bug title"                  # create issue
youtrack issue command MYPROJ-1 "State Fixed"                                 # update state
```

**Do not guess flags or parameters.** Check `README.md` or run `youtrack <group> --help`.

**ID vs login/name:** Some commands (e.g. `project team add-user`, `project team add-group`) expect opaque entity IDs (`2-9`, `4-1`), not logins or names. When creating entities, capture the `id` from the JSON response. When using existing entities, retrieve the ID first with `youtrack user get <login>` or `youtrack group get <name>`.

**Helpdesk agent auto-registration:** `project team add-user` registers the user as a Helpdesk agent of the project when (a) the project is a Helpdesk project AND (b) the user has type `AGENT`. In every other combination it only adds the user to the team.

## Verb Aliases

Each command group registers both the canonical verb and a more intuitive alias — both appear in `--help` and work identically:

| Canonical | Alias | Group |
|---|---|---|
| `get` | `view` | `issue`, `project`, `user`, `board`, `group` |
| `add` | `create` | `issue`, `project`, `user`, `board`, `dashboard`, `group` |
| `current` | `me` | `user` |
| `get-token` | `login` | `auth` |

## Common Commands

- Discovery: `youtrack user me`, `youtrack project list`, `youtrack issue list`
- Issue work: `youtrack issue create`, `youtrack issue view <id>`, `youtrack issue update`, `youtrack issue delete`, `youtrack issue command`
- Collaboration: `youtrack issue comment add`, `youtrack issue comment list`, `youtrack issue comment delete`
- Admin work: `youtrack project create`, `youtrack project delete`, `youtrack user create` (use `--type standard|agent|reporter`; defaults to `standard`)
- Roles: `youtrack role grant --user <id>|--group <id> --role <id> [--project <id>]`, `youtrack role assignments [--holder <id>]`, `youtrack role revoke <assignedRoleID>`

## Full Command Reference

### issue

```bash
youtrack issue get <issueId>           # alias: view
youtrack issue list [--query Q] [--limit N]
youtrack issue add --project P --summary S [--description D] [--parent <issueId>] [--description-file path]
                                       # alias: create
youtrack issue update <issueId> [--summary S] [--description D]
youtrack issue delete <issueId>
youtrack issue command <issueId[,issueId...]> <query>   # apply a YouTrack command string
youtrack issue count [--query Q]       # may return -1 on first call (async); retry manually
youtrack issue suggest <issueId[,issueId...]> <query>   # get command completions
youtrack issue assist <query> <issueId...>              # AI-assisted command suggestions
```

**issue — social actions**
```bash
youtrack issue assign <issueId> <login>
youtrack issue unassign <issueId>
youtrack issue watch <issueId>
youtrack issue unwatch <issueId>
youtrack issue vote <issueId>          # cannot vote on issues you reported yourself
youtrack issue unvote <issueId>
youtrack issue tag <issueId> <tagId>
youtrack issue visibility <issueId> [flags]
```

**issue comment**
```bash
youtrack issue comment list <issueId>
youtrack issue comment add <issueId> --text "..."
youtrack issue comment update <issueId> <commentId> --text "..."
youtrack issue comment delete <issueId> <commentId>
youtrack issue comment pin <issueId> <commentId>
youtrack issue comment visibility <issueId> <commentId> [flags]
```

**issue link**
```bash
youtrack issue link list <issueId>
youtrack issue link add <issueId> --target <targetId> --type <linkTypeId>   # ⚠ known bug: HTTP 405 (F-01)
youtrack issue link remove <issueId> <linkId>   # linkId from 'issue link list --json | jq .[].id'
youtrack issue link type list
youtrack issue link type add [flags]
```

**issue attachment**
```bash
youtrack issue attachment list <issueId>
youtrack issue attachment add <issueId> <filepath>
youtrack issue attachment delete <issueId> <attachmentId>
```

**issue field (custom fields)**
```bash
youtrack issue field get <issueId> <fieldName>
youtrack issue field set <issueId> <fieldName> <value>
```

**issue reaction**
```bash
youtrack issue reaction add <issueId> <commentId> <reaction>
youtrack issue reaction delete <issueId> <commentId> <reactionId>
```

**issue draft**
```bash
youtrack issue draft create [--project P] [--summary S]
```

**issue workitem**
```bash
youtrack issue workitem add <issueId> --duration <ISO8601>   # e.g. PT1H30M, P1DT2H
```

---

### project

```bash
youtrack project list
youtrack project get <shortName>       # alias: view
youtrack project add --shortName P --name "..." --leader <login> [--template helpdesk]
                                       # alias: create; --leader is required (API rejects without it)
youtrack project delete <shortName>
youtrack project archive <shortName>
```

**project team**
```bash
# ⚠ Both commands require opaque IDs, not logins/names
youtrack project team add-user <shortName> <userId>    # userId: 'user get <login> --json | jq .id'
youtrack project team add-group <shortName> <groupId>  # groupId: 'group get <name> --json | jq .id'
```

**project field**
```bash
youtrack project field list <shortName>
youtrack project field add <shortName> [flags]
```

**project template**
```bash
youtrack project template list
youtrack project template convert <shortName>
```

**project timetrack**
```bash
youtrack project timetrack set <shortName> --enabled        # enable time tracking
youtrack project timetrack set <shortName> --enabled=false  # disable time tracking
```

**project workflow**
```bash
youtrack project workflow add <shortName> [flags]
```

---

### user

```bash
youtrack user list [--query Q]
youtrack user get <login>              # alias: view
youtrack user current                  # alias: me
youtrack user add --login L --fullName "..." --email E --password P [--type standard|agent|reporter]
                                       # alias: create; defaults to standard
youtrack user delete <login> --successor <login>   # --successor is required; cannot delete yourself
youtrack user ban <login>
youtrack user unban <login>
youtrack user pin-project <userId> <projectId>     # ⚠ userId is opaque (not login)
```

**user drafts**
```bash
youtrack user drafts list <login>
youtrack user drafts articles <login>
youtrack user drafts delete <login>     # deletes ALL issue drafts for the user
```

**user locale**
```bash
youtrack user locale set <login> <locale>   # e.g. en_US, de_DE  ⚠ known bug: HTTP 500 (F-03)
```

**user timezone**
```bash
youtrack user timezone get <login>
youtrack user timezone set <login> <timezoneId>   # e.g. "Europe/Berlin"
```

**user profile** (set arbitrary profile fields via JSON body flags)
```bash
youtrack user profile general <login> [flags]
youtrack user profile appearance <login> [flags]
youtrack user profile notifications <login> [flags]
youtrack user profile articles <login> [flags]
youtrack user profile issues-list <login> [flags]
youtrack user profile grazie <login> [flags]
youtrack user profile tips <login> <key=true|false>...
youtrack user profile property <login> <key> <value>
```

### user profile examples

| Subcommand | Example `--body` JSON |
|---|---|
| `general` | `'{"timezone":{"id":"Europe/Berlin"}}'` |
| `appearance` | `'{"theme":"DARK","language":"en"}'` |
| `notifications` | `'{"notifyOnOwnChanges":false}'` |
| `articles` | `'{"showSidebar":true}'` |
| `issues-list` | `'{"defaultQuery":"for: me"}'` |
| `grazie` | `'{"enableGrazie":true}'` |
| `tips` | `'{"showNavigationTips":false}'` |

Run `youtrack user profile general <userId> --help` for all available flag names.

---

### Known Limitations

See `.agents/skills/youtrack-cli/references/cli-research-findings.md` for full details.

| Command | Problem |
|---|---|
| `issue link add` | HTTP 405 — wrong API endpoint (F-01) |
| `field bundle *` | HTTP 404 — PascalCase bundle type in path (F-02) |
| `user locale set` | HTTP 500 — wrong payload key (F-03) |
| `issue count` | Returns `-1` when async not ready; no retry (F-05) |
| `user pin-project` | Requires opaque userId, not login (F-06) |
| `issue link remove` | Requires opaque linkId from `link list --json` (F-07) |

## Global Flags

```
--json / -j            Force JSON output
--plain                Force tab-separated output (no headers, pipeline-friendly)
--format json|table|plain  Explicit format override (default: table on TTY, json when piped)
--fields FIELDS / -f   Comma-separated field projection, e.g. id,summary,reporter
--server ALIAS / -s    Use a named instance profile for this invocation (does not persist)
--base-url URL         Override YOUTRACK_BASE_URL
--token TOKEN          Override YOUTRACK_TOKEN
--verbose              Print HTTP requests/responses to stderr
```

**`--fields` syntax:**
- Bare names: `--fields id,summary,reporter` (nested entities auto-expand)
- Sub-field spec: `--fields 'comments(id,text)'` (shell quotes required for parens)
- Dot path: `--fields project.shortName`
- Auto-expanded: `reporter` → `reporter(id,login,fullName)`, `project` → `project(id,shortName,name)`

## References

> **Note:** The `references/` directory contains documentation for the legacy Node.js CLI (v1). Use `README.md` in the repo root as the authoritative command reference for the current Go CLI.

- [Command Reference](references/commands.md) — legacy Node.js CLI reference (pre-v2)
- [Workflows](references/workflows.md) — legacy Node.js CLI workflows (pre-v2)
- [Gotchas & Tips](references/gotchas.md) — YouTrack API gotchas (still relevant for API behaviour, not command syntax)
