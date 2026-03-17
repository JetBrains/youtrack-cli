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

## Core Commands

| Area | Commands |
|---|---|
| Issues | `add-issue`, `get-issue`, `get-issues`, `get-issue-count`, `update-issue-summary`, `update-issue-description`, `delete-issue`, `apply-command`, `get-command-assist`, `update-issue-visibility`, `get-custom-field-value`, `set-custom-field` |
| Comments | `add-comment`, `get-comments`, `update-comment`, `delete-comment`, `update-comment-visibility`, `pin-comment` |
| Issue links | `get-issue-link-types`, `link-issues`, `get-linked-issues`, `remove-issue-link`, `add-issue-link-type` |
| Articles | `add-article`, `get-articles`, `get-article`, `delete-article`, `update-article-summary`, `update-article-content`, `set-parent-article`, `update-article-visibility`, `tag-article`, `get-removed-documents`, `restore-document` |
| Projects | `get-projects`, `get-project`, `add-project`, `delete-project`, `archive-project`, `get-project-templates`, `convert-project-to-template`, `add-custom-field-to-project`, `get-project-custom-fields`, `set-time-tracking-settings`, `add-user-to-project-team`, `add-group-to-project-team`, `attach-workflow-to-project` |
| Users | `get-users`, `get-user`, `get-current-user`, `add-user`, `delete-user`, `ban-user`, `unban-user` |
| Groups | `get-groups`, `get-group`, `add-group`, `delete-group` |
| Roles & permissions | `get-roles`, `add-role`, `update-role`, `delete-role`, `grant-role-to-user`, `grant-role-to-group`, `get-permissions` |
| Boards & sprints | `get-agile-boards`, `get-agile-board`, `add-agile-board`, `delete-agile-board`, `add-sprint` |
| Dashboards | `get-dashboards`, `add-dashboard`, `delete-dashboard`, `update-dashboard-name` |
| Tags | `tag-issue`, `tag-article`, `add-tag`, `delete-tag` |
| Work items | `get-work-item-types`, `add-work-item-type`, `add-work-item`, `get-work-time-settings`, `set-work-time-settings` |
| Attachments | `get-attachments`, `add-attachment`, `delete-attachment`, `add-article-attachment`, `get-article-attachments` |
| Reactions | `add-reaction`, `delete-reaction` |
| Workflows | `get-workflows`, `add-workflow`, `add-workflow-rule`, `delete-workflow`, `restore-default-workflows` |
| Custom fields | `get-custom-field-prototypes`, `add-custom-field`, `delete-custom-field`, `add-bundle-value`, `get-bundles`, `create-bundle`, `attach-bundle-to-field`, `change-custom-field-privacy` |
| Global settings | `get-global-settings`, `get-frontend-config`, `enable-feature-flag`, `set-license`, `set-email-settings`, `disable-email-settings`, `enable-global-banner`, `disable-global-banner`, `set-reactions-enabled`, `switch-jetbrains-ai`, `switch-text-completion`, `switch-grammar-correction` |
| Saved searches | `add-saved-search`, `share-saved-search` |
| Gantt / Canvas | `get-gantt-charts`, `delete-gantt-chart`, `get-canvas`, `add-canvas`, `delete-canvas`, `reload-canvas`, `add-canvas-node`, `get-canvas-links`, `link-canvas-nodes` |
| Apps & plugins | `get-apps`, `delete-app`, `deactivate-app`, `get-plugins`, `delete-plugin` |
| VCS | `get-vcs-servers`, `delete-vcs-server`, `get-vcs-configurations`, `get-vcs-hosting-processors`, `delete-vcs-hosting-processor` |
| Mailbox | `get-mailbox-servers`, `add-mailbox-server`, `delete-mailbox-server`, `add-mailbox-rule` |
| Organizations | `get-organizations`, `add-organization`, `delete-organization` |
| Raw API | `raw --path <endpoint> --method GET/POST --apiFields ... --top N --paginate true` |

## References

- [Command Reference](references/commands.md) — all commands with required/optional params
- [Workflows](references/workflows.md) — step-by-step patterns, query syntax, `--asUser` usage
- [Gotchas & Tips](references/gotchas.md) — known defaults, Hub limitations, tips
