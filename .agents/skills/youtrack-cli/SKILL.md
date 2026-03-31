---
name: youtrack-cli
version: "2.0.0"
author: JetBrains
description: CRUD for YouTrack entities (issues, projects, users, comments) via the youtrack-cli Go CLI. Use when the user asks to manage YouTrack data, run queries, or configure a YouTrack instance.
allowed-tools: Bash
---

## Bootstrap: Get a Token

Use `auth get-token` when no `YOUTRACK_TOKEN` exists yet. Requires Hub login/password.

```bash
youtrack auth get-token --base-url https://your.youtrack.cloud --login alice --save
# Password: (prompted, hidden)
```

`--save` writes both `YOUTRACK_BASE_URL` and `YOUTRACK_TOKEN` to `~/.youtrack-cli-env`, loaded automatically on every CLI invocation. No need to source or export manually.

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
