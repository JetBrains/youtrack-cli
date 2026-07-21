---
name: youtrack-cli
version: "2.0.0"
author: JetBrains
description: CRUD for YouTrack entities (issues, projects, users, comments) via the youtrack-cli Go CLI. Use when the user asks to manage YouTrack data, run queries, or configure a YouTrack instance.
allowed-tools: Bash
---

# YouTrack CLI (`youtrack`)

## Quick Start

```bash
youtrack user current                                          # verify auth
youtrack issue list --query "assignee: me State: Open"        # find issues
youtrack issue create --project MYPROJ --summary "Bug title"  # create issue
youtrack issue command MYPROJ-1 "State Fixed"                 # update state
```

**Do not guess flags or parameters.** Use the [command reference](references/commands.md) or `youtrack <group> --help`. See [Gotchas & Tips](references/gotchas.md) for known API pitfalls.

## Setup

1. **CLI available?** `which youtrack` — if not found, run `make build && cp youtrack /usr/local/bin/` from the repo root.
2. **Credentials set?** Check the configured instances with `youtrack instance list` or use env vars `YOUTRACK_BASE_URL` + `YOUTRACK_TOKEN`. If missing, run `youtrack instance add` (or `youtrack auth login`) for a permanent token. `youtrack instance login <url>` is the browser SSO option, but it yields a short-lived (~1 hour) session token that is not refreshed.
3. **Connectivity OK?** `youtrack user current` — stop and report if it fails.

## Core Commands

Canonical verbs: `view`, `list`, `create`, `update`, `delete` (`get`/`add` still work as deprecated aliases — prefer the canonical form below).

| Area | Commands |
|---|---|
| Auth | `auth login`, `auth status`, `auth logout`, `auth refresh-token` |
| Instances | `instance add`, `instance login` (browser OAuth), `instance list`, `instance use`, `instance version`, `instance delete` (`remove`), `instance refresh-token` (`update-token`) |
| Issues | `issue list`, `view` (`get`), `create` (`add`), `update`, `delete`, `command`, `count` |
| Issue social | `issue assign/unassign`, `watch/unwatch`, `vote/unvote`, `tag`, `visibility` |
| Comments | `issue comment list`, `create` (`add`), `update`, `delete`, `pin`, `visibility` |
| Links | `issue link list`, `add`, `remove`, `type list/create` (`add`) |
| Attachments | `issue attachment list`, `add`, `delete` |
| Custom fields | `issue field view` (`get`), `set` |
| Work items | `issue workitem create` (`add`) |
| Projects | `project list`, `view` (`get`), `create` (`add`), `delete`, `archive` |
| Project team | `project team add-user`, `add-group` |
| Project fields | `project field list`, `add` |
| Users | `user list`, `view` (`get`), `current` (`me`), `create` (`add`), `delete`, `ban`, `unban` |
| User profile | `user profile view` (`get`), `general`, `appearance`, `notifications`, `articles`, `issues-list`, `grazie`, `tips`, `property`, `timetracking view` |
| Groups | `group list`, `view` (`get`), `create` (`add`), `delete` |
| Roles | `role list`, `view` (`get`), `grant`, `revoke`, `assignments` |
| Boards | `board list`, `view` (`get`), `create` (`add`), `update`, `delete` + `board sprint list/view(get)/create(add)` |
| Whiteboards | `whiteboard list`, `create` (`add`), `view` (`get`), `delete`, `reload` + `whiteboard node list/view(get)/create(add)`, `links`, `link` |
| Articles | `article list`, `view` (`get`), `create` (`add`), `update`, `delete` + comments, attachments, tags |
| Tags | `tag list`, `create` (`add`), `delete`, `tag issue list <tagId>` |
| Dashboards | `dashboard list`, `create` (`add`), `update`, `delete` |
| Saved searches | `search list`, `view` (`get`), `create` (`add`), `delete`, `share` |

## Output & fields

Bare `--json` returns every field the API sends for that entity. `--fields` / `-f` selects columns in table/plain output and widens what the API returns, but does **not** trim JSON output — pipe JSON through `jq` to narrow it. Use `--fields` to pull nested fields the default projection omits:

```bash
youtrack issue list --query "assignee: me" --fields id,summary,state
```

On large projects, pass `--limit` on `issue list` (default 100) to cap how many issues come back instead of pulling the full list.

## Quick Workflows

- **Browser sign-in**: `youtrack instance login <url>` → OAuth 2.0 + PKCE against JetBrains Hub, saves a ~1-hour session token (re-run to renew, or pass an existing alias); falls back to paste-token when the Hub has no public loopback client. Needs an interactive terminal.
- **Verify auth**: `youtrack auth status` → check `status` field per instance
- **Create issue**: `youtrack issue create --project P --summary S` → `issue command <id> "assignee me"`
- **Bulk update**: `youtrack issue list --query Q --json | jq '.[].id'` → loop `issue command`
- **Add user to project**: `youtrack user view <login> --json | jq .id` → `youtrack project team add-user <proj> <userId>`

## References

- [Command Reference](references/commands.md) — full syntax, flags, auth flows, global options
- [Gotchas & Tips](references/gotchas.md) — YouTrack API gotchas (still relevant for API behaviour)
- [Workflows](references/workflows.md) — common multi-step workflows
