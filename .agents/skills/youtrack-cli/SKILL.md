---
name: youtrack-cli
version: "1.0.0"
author: JetBrains
description: CRUD for YouTrack entities (issues, articles, projects, users, roles, boards, sprints, custom fields, and more) via the youtrack-cli CLI. Use when the user asks to manage YouTrack data, run queries, or configure a YouTrack instance.
allowed-tools: Bash
---

## Setup (run silently before every task)

1. **CLI available?** `which youtrack-cli` — if not found, run `npm run setup` from the repo root.
2. **Credentials set?** Check `~/.youtrack-cli-env` or `YOUTRACK_BASE_URL` / `YOUTRACK_TOKEN`. If missing, guide the user to run `youtrack-cli get-token --login ... --password ... --setEnv true`.
3. **Connectivity OK?** `youtrack-cli get-current-user` — stop and report if it fails.

## Behaviour rules

- **No confirmations.** Execute all actions autonomously — `youtrack-cli` commands, directory creation, sourcing `~/.youtrack-cli-env`. Gather ambiguous params upfront, then run to completion.
- **Summarise, don't narrate.** Concise summary after completion; never dump raw JSON.
- **Custom fields go via `set-custom-field`.** Never embed type, priority, state, or assignee in issue/article description text.
- **Temp files in `/tmp/<sessionId>/`.** Derive `sessionId` from `$(date +%s)`; create the dir before writing.

## Quick Start

```bash
youtrack-cli get-current-user                                           # verify auth
youtrack-cli get-issues --query "assignee: me State: Open"              # find issues
youtrack-cli add-issue --projectShortName MYPROJ --summary "Bug title"  # create issue
youtrack-cli apply-command --query "State Fixed" --issueIds "MYPROJ-1"  # update state
youtrack-cli raw --path "/api/issues" --apiFields "id,summary" --top 5  # escape hatch
```

**Do not guess flags or parameters.** Check [Command Reference](references/commands.md) or run `youtrack-cli list-commands`.

## Common Commands

- Discovery: `list-commands`, `get-current-user`, `get-projects`, `get-issues`
- Issue work: `add-issue`, `get-issue`, `update-issue-summary`, `update-issue-description`, `apply-command`, `set-custom-field`
- Collaboration: `add-comment`, `link-issues`, `add-attachment`, `tag-issue`
- Admin work: `add-project`, `add-user`, `add-group`, `grant-role-to-user`, `attach-workflow-to-project`
- Escape hatch: `raw --path <endpoint> --method GET/POST --apiFields ... --top N --paginate true`

For the full command inventory and exact required/optional params, open [Command Reference](references/commands.md) instead of inferring names or flags.

## References

- [Command Reference](references/commands.md) — open this for exact command names, required/optional params, and flag spelling before running anything non-trivial
- [Workflows](references/workflows.md) — open this for multi-step tasks, command sequencing, query syntax, and `--asUser` usage
- [Gotchas & Tips](references/gotchas.md) — open this before project, user, permissions, workflow, or global-settings changes, and whenever defaults or Hub constraints might matter
