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
Returns: `{"url":"...","login":"...","fullName":"...","status":"authenticated"}`
Status values: `"authenticated"` | `"invalid token: ..."` | `"not configured: ..."`
Always exits 0 — check the `status` field programmatically. Use after `auth get-token` to confirm credentials were saved correctly.

### Authenticate interactively
```bash
youtrack auth get-token
```
Prompts for base URL (if `YOUTRACK_BASE_URL` not set), then auth method:
- **1 — Basic auth**: prompts for login + password, creates a Hub permanent token via Hub API.
  - The interactive password prompt requires a TTY. In scripts, pass `--password <pass>` as a flag to avoid the prompt.
  - Use `--scope <service-names>` (comma-separated) to request specific Hub services. Default: YouTrack + YouTrack Administration.
  - If the Hub endpoint returns HTML (cloud/SSO-only instances), falls back automatically to option 3.
- **2 — Web browser**: detects Hub URL via `/.well-known/oauth-protected-resource/mcp`, opens the token page in a browser, prompts for paste, verifies before saving.
  - Internal Hub → `{youtrackURL}/users/me?tab=account-security`
  - External Hub → `{hubURL}/users/me?tab=authentication`
  - If the browser cannot be opened automatically, the CLI prints the token page URL; open it manually.
- **3 — Paste token directly**: shows the correct token page URL (same Hub detection as option 2) without opening a browser. User opens it manually, creates a token, pastes it. Token is verified before saving. Used automatically as fallback when basic auth is unavailable.

All flows save `YOUTRACK_BASE_URL` and `YOUTRACK_TOKEN` to `~/.youtrack-cli-env` by default (`--save=false` to skip).
The CLI auto-prepends `https://` if the base URL is entered without a scheme.

## Setup (run silently before every task)

1. **CLI available?** `which youtrack` — if not found, run `make build && cp youtrack /usr/local/bin/` from the repo root.
2. **Credentials set?** Check `~/.youtrack-cli-env` or `YOUTRACK_BASE_URL` / `YOUTRACK_TOKEN`. If missing, guide the user to export them or write to `~/.youtrack-cli-env`.
3. **Connectivity OK?** `youtrack user current` — stop and report if it fails.

## Behaviour rules

- **No confirmations.** Execute all actions autonomously — `youtrack` commands, directory creation, sourcing `~/.youtrack-cli-env`. Gather ambiguous params upfront, then run to completion.
- **Summarise, don't narrate.** Concise summary after completion; never dump raw JSON.
- **Output format.** Default output is JSON. Use `--format table` for human-readable output in TTY contexts.

## Quick Start

```bash
youtrack user current                                                          # verify auth
youtrack issue list --query "assignee: me State: Open"                        # find issues
youtrack issue add --project MYPROJ --summary "Bug title"                     # create issue
youtrack issue command MYPROJ-1 "State Fixed"                                 # update state
```

**Do not guess flags or parameters.** Check `README.md` or run `youtrack <group> --help`.

**ID vs login/name:** Some commands (e.g. `project team add-user`, `project team add-group`) expect opaque entity IDs (`2-9`, `4-1`), not logins or names. When creating entities, capture the `id` from the JSON response. When using existing entities, retrieve the ID first with `youtrack user get <login>` or `youtrack group get <name>`.

## Common Commands

- Discovery: `youtrack user current`, `youtrack project list`, `youtrack issue list`
- Issue work: `youtrack issue add`, `youtrack issue get`, `youtrack issue update`, `youtrack issue delete`, `youtrack issue command`
- Collaboration: `youtrack issue comment add`, `youtrack issue comment list`, `youtrack issue comment delete`
- Admin work: `youtrack project add`, `youtrack project delete`, `youtrack user add`

## Global Flags

```
--format json|table    Output format (default: json)
--base-url URL         Override YOUTRACK_BASE_URL
--token TOKEN          Override YOUTRACK_TOKEN
--verbose              Print HTTP requests/responses to stderr
```

## References

> **Note:** The `references/` directory contains documentation for the legacy Node.js CLI (v1). Use `README.md` in the repo root as the authoritative command reference for the current Go CLI.

- [Command Reference](references/commands.md) — legacy Node.js CLI reference (pre-v2)
- [Workflows](references/workflows.md) — legacy Node.js CLI workflows (pre-v2)
- [Gotchas & Tips](references/gotchas.md) — YouTrack API gotchas (still relevant for API behaviour, not command syntax)
