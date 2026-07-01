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
youtrack user me                                               # verify auth
youtrack issue list --query "assignee: me State: Open"        # find issues
youtrack issue create --project MYPROJ --summary "Bug title"  # create issue
youtrack issue command MYPROJ-1 "State Fixed"                 # update state
```

**Do not guess flags or parameters.** Use the [command reference](references/commands.md) or `youtrack <group> --help`. See [Gotchas & Tips](references/gotchas.md) for known API pitfalls.

## Setup

1. **CLI available?** `which youtrack` — if not found, run `make build && cp youtrack /usr/local/bin/` from the repo root.
2. **Credentials set?** Check `~/.youtrack-cli-config.json` or env vars `YOUTRACK_BASE_URL` + `YOUTRACK_TOKEN`. If missing, run `youtrack auth get-token`.
3. **Connectivity OK?** `youtrack user me` — stop and report if it fails.

## Core Commands

| Area | Commands |
|---|---|
| Auth | `auth get-token` (`login`), `auth status`, `auth logout`, `auth refresh-token` |
| Instances | `instance list`, `instance use`, `instance remove`, `instance update-token` |
| Issues | `issue list`, `get` (`view`), `add` (`create`), `update`, `delete`, `command`, `count` |
| Issue social | `issue assign/unassign`, `watch/unwatch`, `vote/unvote`, `tag`, `visibility` |
| Comments | `issue comment list`, `add`, `update`, `delete`, `pin`, `visibility` |
| Links | `issue link list`, `add`, `remove`, `type list/add` |
| Attachments | `issue attachment list`, `add`, `delete` |
| Custom fields | `issue field get`, `set` |
| Work items | `issue workitem add` |
| Projects | `project list`, `get` (`view`), `add` (`create`), `delete`, `archive` |
| Project team | `project team add-user`, `add-group` |
| Project fields | `project field list`, `add` |
| Users | `user list`, `get` (`view`), `current` (`me`), `add` (`create`), `delete`, `ban`, `unban` |
| User profile | `user profile general`, `appearance`, `notifications`, `articles`, `issues-list`, `grazie`, `tips`, `property` |
| Groups | `group list`, `get`, `add`, `delete` |
| Roles | `role list`, `get`, `grant`, `revoke`, `assignments` |
| Boards | `board list`, `get`, `add`, `update`, `delete` + `board sprint list/get/add` |
| Whiteboards | `whiteboard list`, `add`, `get`, `delete`, `reload` + `whiteboard node list/get/add`, `links`, `link` |
| Articles | `article list`, `get`, `add`, `update`, `delete` + comments, attachments, tags |
| Tags | `tag list`, `get`, `add`, `delete` |
| Dashboards | `dashboard list`, `add`, `delete` |
| Saved searches | `search list`, `add`, `delete` |

## Quick Workflows

- **Verify auth**: `youtrack auth status` → check `status` field per instance
- **Create issue**: `youtrack issue add --project P --summary S` → `issue command <id> "assignee me"`
- **Bulk update**: `youtrack issue list --query Q --json | jq '.[].id'` → loop `issue command`
- **Add user to project**: `youtrack user get <login> --json | jq .id` → `youtrack project team add-user <proj> <userId>`

## References

- [Command Reference](references/commands.md) — full syntax, flags, auth flows, global options
- [Gotchas & Tips](references/gotchas.md) — YouTrack API gotchas (still relevant for API behaviour)
- [Workflows](references/workflows.md) — common multi-step workflows
