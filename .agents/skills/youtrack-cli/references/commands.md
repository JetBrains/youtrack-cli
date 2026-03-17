# Command Reference

## Global options

| Flag | Effect |
|---|---|
| `--verbose` / `--debug` | Log full HTTP requests and responses to stderr (credentials redacted) |
| `--quiet` / `-q` | Suppress all output (exit 0 on success, 1 on error) |
| `--fields <list>` | Return only these comma-separated fields, e.g. `--fields id,summary`. Use `--fields '?'` to list available top-level fields. Supports dot-paths (`project.shortName`). |
| `--base-url URL` | Override `YOUTRACK_BASE_URL` |
| `--token TOKEN` | Override `YOUTRACK_TOKEN` |

`YOUTRACK_RO=1` — env var that blocks all write operations before execution.

On error: `{ "error": "...", "suggestion": "..." }` to stderr, exit code 1.

---

## Issues

| Command | Required params | Optional params |
|---|---|---|
| `add-issue` | `projectShortName`, `summary` | `description`, `descriptionFile`, `asUser` |
| `get-issue` | `issueId` | |
| `get-issues` | `query` | `limit` |
| `get-issue-count` | `query` | |
| `update-issue-summary` | `issueId`, `newSummary` | |
| `update-issue-description` | `issueId`, `newDescription` | |
| `delete-issue` | `issueId` | |
| `apply-command` | `query`, `issueIds` | |
| `get-command-assist` | `query`, `issueIds` | |
| `update-issue-visibility` | `issueId` | `permittedGroups`, `permittedUsers` |
| `get-custom-field-value` | `issueId`, `fieldName` | |
| `set-custom-field` | `issueId`, `fieldName`, `fieldValue`, `fieldType` | |

- `issueIds` — comma-separated, e.g. `MYPROJ-1,MYPROJ-2`
- `fieldType` — `enum` (default), `user`, `period` (e.g. `2h 30m`), `text`, `int`, `string`
- `get-issues` defaults to `$top=100`; pass `--limit -1` for all results
- `asUser` — login to set as reporter (requires admin token)

## Comments

All comment commands accept `place` (`"issues"` | `"articles"`, default `"issues"`), using `parentId` as the issue or article ID.

| Command | Required params | Optional params |
|---|---|---|
| `add-comment` | `parentId` | `text`, `textFile`, `place`, `asUser` |
| `get-comments` | `parentId` | `place` |
| `update-comment` | `parentId`, `commentId` | `text`, `textFile`, `place` |
| `delete-comment` | `parentId`, `commentId` | `place` |
| `update-comment-visibility` | `parentId`, `commentId` | `place`, `permittedGroups`, `permittedUsers` |
| `pin-comment` | `parentId`, `commentId` | `place`, `pinned` |

- `asUser` — login to post as (requires admin token)
- `pinned` — defaults to `true`; pass `--pinned false` to unpin
- Article comments: `--place articles` (no separate `add-article-comment` command)

## Issue links

| Command | Required params | Optional params |
|---|---|---|
| `get-issue-link-types` | | |
| `link-issues` | `issueId`, `linkedIssueId`, `linkType` | |
| `get-linked-issues` | `issueId`, `linkType` | |
| `remove-issue-link` | `issueId`, `linkedIssueId`, `linkType` | |
| `add-issue-link-type` | `name`, `sourceToTarget` | `targetToSource`, `directed`, `aggregation` |

Run `get-issue-link-types` first to find valid `linkType` values (e.g. `depends on`, `duplicates`, `relates to`).

## Articles

| Command | Required params | Optional params |
|---|---|---|
| `add-article` | `projectShortName`, `summary` | `content`, `contentFile`, `asUser` |
| `get-articles` | | `query` |
| `get-article` | `articleId` | |
| `delete-article` | `articleId` | |
| `update-article-summary` | `articleId`, `newSummary` | |
| `update-article-content` | `articleId` | `newContent`, `newContentFile` |
| `set-parent-article` | `articleId`, `parentArticleId` | |
| `update-article-visibility` | `articleId` | `permittedGroups`, `permittedUsers` |
| `tag-article` | `articleId`, `tagId` | |
| `get-removed-documents` | | |
| `restore-document` | `idReadable` | |

- `asUser` — login to create the article as (requires admin token)

## Projects

| Command | Required params | Optional params |
|---|---|---|
| `get-projects` | | |
| `get-project` | `shortName` | |
| `add-project` | `shortName`, `name` | `leader` |
| `delete-project` | `projectId` | |
| `archive-project` | `projectId` | |
| `get-project-templates` | | |
| `convert-project-to-template` | `projectId` | |
| `add-custom-field-to-project` | `projectId`, `customFieldId` | |
| `get-project-custom-fields` | `projectId` | |
| `set-time-tracking-settings` | `projectId` | `enabled`, `spentTimeField`, `estimationField` |
| `add-user-to-project-team` | `projectShortName`, `userId` | |
| `add-group-to-project-team` | `projectShortName`, `groupName` | |
| `attach-workflow-to-project` | `workflowId`, `projectId` | |

- `delete-project` / `archive-project` require internal `projectId` (not short name) — use `get-project` to find it
- `leader` is optional in the CLI but the API rejects project creation without one; always pass it (use `get-current-user` if unspecified)

## Users

| Command | Required params | Optional params |
|---|---|---|
| `get-users` | | `query` |
| `get-user` | `login` | |
| `get-current-user` | | |
| `add-user` | `login`, `fullName`, `email` | `password`, `userType` |
| `delete-user` | `userId` | `successor` |
| `ban-user` / `unban-user` | `userId` | |

- `userType` — `STANDARD_USER` (default), `AGENT`, `REPORTER`

## Groups

| Command | Required params | Optional params |
|---|---|---|
| `get-groups` | | `query` |
| `get-group` | `groupName` | |
| `add-group` | `name` | |
| `delete-group` | `groupName` | `successor` |

Group/user IDs are the opaque `id` field returned by `get-users` / `get-groups`.

> Adding/removing users from groups is not supported by the YouTrack REST API. Use `add-group-to-project-team` instead.

## Roles & permissions

| Command | Required params | Optional params |
|---|---|---|
| `get-roles` | | |
| `add-role` | `name` | `description`, `permissions` |
| `update-role` | `roleId`, `permissions` | |
| `delete-role` | `roleId` | |
| `grant-role-to-user` | `userId`, `roleName` | `projectName` |
| `grant-role-to-group` | `groupName`, `roleName` | `projectName` |
| `get-permissions` | | |

- Omit `--projectName` to grant globally; include it to scope to a project
- `permissions` — comma-separated permission IDs from `get-permissions`
- `update-role` REPLACES the full permission list

## Agile boards & sprints

| Command | Required params | Optional params |
|---|---|---|
| `get-agile-boards` | | |
| `get-agile-board` | `boardId` | |
| `add-agile-board` | `name` | `template` (`scrum`/`kanban`), `projectIds` |
| `delete-agile-board` | `boardId` | |
| `add-sprint` | `boardId`, `name`, `startDate`, `endDate` | `goal` |

`startDate` and `endDate` are Unix timestamps in milliseconds.

## Dashboards

| Command | Required params | Optional params |
|---|---|---|
| `get-dashboards` | | |
| `add-dashboard` | `name` | |
| `delete-dashboard` | `dashboardId` | |
| `update-dashboard-name` | `dashboardId`, `newName` | |

## Tags, work items, reactions, attachments

| Command | Required params | Optional params |
|---|---|---|
| `tag-issue` | `issueId`, `tagId` | |
| `tag-article` | `articleId`, `tagId` | |
| `add-tag` | `name`, `ownerId` | `colorId`, `isShareable`, `isUpdatable`, `untagOnResolve`, `pinned`, `pinnedByDefault` |
| `delete-tag` | `tagId` | |
| `get-work-item-types` | | |
| `add-work-item-type` | `name` | `autoAttach` |
| `add-work-item` | `issueId`, `minutes` | `workType`, `text`, `authorLogin`, `date` |
| `get-work-time-settings` | | |
| `set-work-time-settings` | `hoursInADay` | |
| `get-attachments` | `issueId` | |
| `add-attachment` | `issueId`, `file` | `name` |
| `delete-attachment` | `issueId`, `attachmentId` | |
| `add-article-attachment` | `articleId`, `file` | `name` |
| `get-article-attachments` | `articleId` | |
| `add-reaction` | `issueId`, `commentId`, `reaction` | |
| `delete-reaction` | `issueId`, `commentId`, `reactionId` | |

- `file` — path to local file; `name` overrides the filename in YouTrack (defaults to basename)
- **Always use `add-attachment` / `add-article-attachment`** for uploads — never `curl`

## Workflows

| Command | Required params | Optional params |
|---|---|---|
| `get-workflows` | | |
| `add-workflow` | `name`, `title` | |
| `add-workflow-rule` | `workflowId`, `name`, `script` | |
| `delete-workflow` | `workflowId` | |
| `restore-default-workflows` | | |

## Custom fields & bundles

| Command | Required params | Optional params |
|---|---|---|
| `get-custom-field-prototypes` | | |
| `add-custom-field` | `name`, `fieldTypeId` | |
| `delete-custom-field` | `customFieldId` | |
| `add-bundle-value` | `bundleType`, `bundleId`, `value` | |
| `get-bundles` | `bundleType` | |
| `create-bundle` | `bundleType` | `name` |
| `attach-bundle-to-field` | `customFieldId`, `bundleId`, `bundleType` | |
| `change-custom-field-privacy` | `customFieldId` | `isPublic` (default `false`) |

- `bundleType` — path names (`enum`, `build`, `version`) or `$type` names (`EnumBundle`, etc.)
- Use `get-custom-field-prototypes` to find `customFieldId`, `fieldTypeId`, `bundleType`, `bundleId`

## Global settings

| Command | Required params | Optional params |
|---|---|---|
| `get-global-settings` | | |
| `get-frontend-config` | | |
| `enable-feature-flag` | `feature` | `enabled` |
| `set-license` | `username`, `license` | |
| `set-email-settings` | `serverAddress`, `port`, `fromAddress` | `replyToAddress` |
| `disable-email-settings` | | |
| `enable-global-banner` | `message` | |
| `disable-global-banner` | | |
| `set-reactions-enabled` | `enabled` | |
| `switch-jetbrains-ai` | `enabled` | |
| `switch-text-completion` | `enabled` | |
| `switch-grammar-correction` | `enabled` | |

## Saved searches

| Command | Required params | Optional params |
|---|---|---|
| `add-saved-search` | `query`, `name` | |
| `share-saved-search` | `searchId` | `users`, `groups` |

## Gantt charts

| Command | Required params | Optional params |
|---|---|---|
| `get-gantt-charts` | | |
| `delete-gantt-chart` | `ganttId` | |

## Canvas

| Command | Required params | Optional params |
|---|---|---|
| `get-canvas` | `canvasId` | |
| `add-canvas` | `name`, `ownerLogin` | |
| `delete-canvas` | `canvasId` | |
| `reload-canvas` | `canvasId` | |
| `add-canvas-node` | `canvasId`, `x`, `y` | `label`, `issueId`, `articleId`, `nodeType`, `description` |
| `get-canvas-links` | `canvasId` | |
| `link-canvas-nodes` | `canvasId`, `sourceCardId`, `targetCardId` | `linkType` |

## Apps & plugins

| Command | Required params | Optional params |
|---|---|---|
| `get-apps` | | |
| `delete-app` | `appId` | |
| `deactivate-app` | `appId` | |
| `get-plugins` | | |
| `delete-plugin` | `pluginId` | |

## VCS integration

| Command | Required params | Optional params |
|---|---|---|
| `get-vcs-servers` | | |
| `delete-vcs-server` | `serverId` | |
| `get-vcs-configurations` | `serverId` | |
| `get-vcs-hosting-processors` | | |
| `delete-vcs-hosting-processor` | `processorId` | |

## Mailbox

| Command | Required params | Optional params |
|---|---|---|
| `get-mailbox-servers` | | |
| `add-mailbox-server` | `protocol`, `host`, `login`, `password` | `port` |
| `delete-mailbox-server` | `mailboxId` | |
| `add-mailbox-rule` | `mailboxId`, `projectId` | `name`, `disabled` |

## Organizations

| Command | Required params | Optional params |
|---|---|---|
| `get-organizations` | | |
| `add-organization` | `name` | |
| `delete-organization` | `organizationId` | |

## Raw API (escape hatch)

```bash
youtrack-cli raw --path "/api/admin/users" --apiFields "id,login,name" --top 5
youtrack-cli raw --path "/api/issues/MYPROJ-1" --method POST --body '{"summary":"updated"}'
youtrack-cli raw --path "/api/issues" --apiFields "id,summary" --top 100 --paginate true
```

- `--method` — defaults to `GET`
- `--apiFields` — appends `$fields=` (YouTrack field projection)
- `--top` / `--paginate true` — page size and auto-pagination
- `--body` — JSON string for POST/PUT
