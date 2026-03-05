---
name: youtrack-cli
description: Interacts with YouTrack project management instances via the youtrack-cli command-line tool. Covers creating, reading, updating, and deleting YouTrack issues, articles, projects, users, groups, roles, agile boards, sprints, dashboards, tags, work items, custom fields, and workflows. Use when the user asks to manage YouTrack entities, run YouTrack queries, apply commands, or configure a YouTrack instance.
compatibility: Designed for Claude Code. Requires Node.js and youtrack-cli installed globally (run npm run setup from the repo root if unavailable).
allowed-tools: Bash, Python3, Node.js, curl
---

Before helping the user with any YouTrack task, run through the following preparation steps silently. Do not skip any step, do not report progress unless a step fails.

**Step 1 — Ensure the CLI is available**
Run `which youtrack-cli`. If not found, run `npm run setup` from the repo root.

**Step 2 — Ensure credentials are configured**
Check `~/.youtrack-cli-env` or the env vars `YOUTRACK_BASE_URL` / `YOUTRACK_TOKEN`. If neither is present, ask the user for their instance URL and a permanent token (`perm:...`), then guide them to run `get-token --login ... --password ... --setEnv true` or export the vars manually.

**Step 3 — Verify connectivity**
Run `youtrack-cli get-current-user`. If it fails, report the error and stop until it is resolved.

---

You are helping the user interact with their YouTrack instance using the `youtrack-cli` command-line tool.

## Behaviour rules

- **No confirmations required.** Execute all YouTrack CLI actions (creates, updates, deletes, links, uploads, bulk operations) autonomously without asking for confirmation at any point. Never pause mid-task to ask "are you sure?" or "shall I proceed?". Gather all required parameters upfront if truly ambiguous, then run to completion.
- **No confirmation needed for `youtrack-cli` commands.** Any invocation of `youtrack-cli` or `youtrack-cli <subcommand>` must be executed immediately without prompting the user.
- **No confirmation needed for session directory creation.** Creating the `/tmp/<sessionId>/` directory (and any subdirectories within it) must happen silently and immediately without asking for approval.
- **No confirmation needed to source `~/.youtrack-cli-env`.** Reading or sourcing `~/.youtrack-cli-env` to load credentials must be done silently without prompting.
- **No confirmation needed to run `python3` for YouTrack entity content.** Any `python3` invocation that processes, transforms, or prepares content related to YouTrack entities (issues, articles, comments, projects, etc.) must execute immediately without asking for confirmation.
- **Summarise, don't narrate.** Report a concise summary after the operation completes rather than describing each individual step or printing raw JSON.
- **Issue descriptions must not contain custom field values.** Do not embed type labels (e.g. "Bug:", "Feature:", "Task:"), priority, state, assignee, or any other custom field value inside the description text. Custom fields must be set via `set-custom-field` after creation, not inlined into the description.
- **Temporary files go in `/tmp/<sessionId>/`.** Derive `sessionId` from the current timestamp at skill invocation (e.g. `$(date +%s)`). Create the directory before writing any temporary files, and use it for all attachment stubs, description drafts, or any other ephemeral data produced during the session.

## How to run commands

Use the Bash tool to execute commands. The CLI is available globally as:

```
youtrack-cli <command> [--param value ...]
```

All output is JSON. Always pretty-print or summarize the result for the user rather than dumping raw JSON.

## Prerequisites

The following environment variables must be set (loaded automatically from `~/.youtrack-cli-env` if present):

- `YOUTRACK_BASE_URL` — e.g. `https://your-instance.youtrack.cloud`
- `YOUTRACK_TOKEN` — a permanent token starting with `perm:`

If not set, generate and save a token first:
```
youtrack-cli get-token --login <login> --password <password> --setEnv true
```

## Command reference

### Issues
| Command | Required params | Optional params |
|---|---|---|
| `add-issue` | `projectShortName`, `summary` | `description`, `descriptionFile`, `asUser` |
| `get-issue` | `issueId` | |
| `get-issues` | `query` | |
| `update-issue-summary` | `issueId`, `newSummary` | |
| `update-issue-description` | `issueId`, `newDescription` | |
| `delete-issue` | `issueId` | |
| `apply-command` | `query`, `issueIds` | |
| `get-command-assist` | `query`, `issueIds` | |
| `update-issue-visibility` | `issueId` | `permittedGroups`, `permittedUsers` |
| `get-custom-field-value` | `issueId`, `fieldName` | |
| `set-custom-field` | `issueId`, `fieldName`, `fieldValue`, `fieldType` | |

`issueIds` accepts comma-separated IDs, e.g. `MYPROJ-1,MYPROJ-2`.
`fieldType` is one of: `enum` (default), `user`, `period` (e.g. `2h 30m`), `text`, `int`, `string`.
`asUser` — login of another user to set as the issue reporter (requires admin token).

### Comments
| Command | Required params | Optional params |
|---|---|---|
| `add-comment` | `issueId` | `text`, `textFile`, `asUser` |
| `get-comments` | `parentId` | `place` (`issues` or `articles`) |
| `update-comment` | `issueId`, `commentId` | `text`, `textFile` |
| `delete-comment` | `issueId`, `commentId` | |
| `update-comment-visibility` | `issueId`, `commentId` | `permittedGroups`, `permittedUsers` |
| `add-article-comment` | `articleId` | `text`, `textFile`, `asUser` |
| `delete-article-comment` | `articleId`, `commentId` | |

`asUser` — login of another user to post the comment as (requires admin token).

### Issue links
| Command | Required params | Optional params |
|---|---|---|
| `get-issue-link-types` | | |
| `link-issues` | `issueId`, `linkedIssueId`, `linkType` | |
| `add-issue-link-type` | `name`, `sourceToTarget` | `targetToSource`, `directed`, `aggregation` |

Run `get-issue-link-types` first to find valid `linkType` values (e.g. `depends on`, `duplicates`, `relates to`).

### Articles
| Command | Required params | Optional params |
|---|---|---|
| `add-article` | `projectShortName`, `summary` | `content`, `contentFile`, `asUser` |
| `get-articles` | | `query` |
| `get-article` | `articleId` | |
| `delete-article` | `articleId` | |
| `update-article-summary` | `articleId`, `newSummary` | |
| `set-parent-article` | `articleId`, `parentArticleId` | |
| `update-article-visibility` | `articleId` | `permittedGroups`, `permittedUsers` |
| `tag-article` | `articleId`, `tagId` | |
| `get-removed-documents` | | |
| `restore-document` | `idReadable` | |

`asUser` — login of another user to create the article as (requires admin token).

### Projects
| Command | Required params | Optional params |
|---|---|---|
| `get-projects` | | |
| `get-project` | `shortName` | |
| `add-project` | `shortName`, `name`, `leader` | |
| `delete-project` | `projectId` | |
| `archive-project` | `projectId` | |
| `get-project-templates` | | |
| `convert-project-to-template` | `projectId` | |
| `add-custom-field-to-project` | `projectId`, `customFieldId` | |
| `set-time-tracking-settings` | `projectId` | `enabled`, `spentTimeField`, `estimationField` |
| `add-user-to-project-team` | `projectShortName`, `userId` | |
| `add-group-to-project-team` | `projectShortName`, `groupName` | |
| `attach-workflow-to-project` | `workflowId`, `projectId` | |

`delete-project` and `archive-project` require the internal `projectId`, not the short name. Use `get-project` to find it.
`add-project` requires `leader` — the API rejects the request without it. Use `get-current-user` to get the current user's login if no leader is specified.

### Users
| Command | Required params | Optional params |
|---|---|---|
| `get-users` | | `query` |
| `get-user` | `login` | |
| `get-current-user` | | |
| `add-user` | `login`, `fullName`, `email`, `password` | |
| `delete-user` | `userId` | |
| `ban-user` / `unban-user` | `userId` | |

### Groups
| Command | Required params | Optional params |
|---|---|---|
| `get-groups` | | `query` |
| `get-group` | `groupName` | |
| `add-group` | `name` | |
| `delete-group` | `groupName` | `successor` |

Group/user IDs are the opaque `id` field returned by `get-users` and `get-groups`.

> **Note:** Adding or removing users from groups is not supported by the YouTrack REST API. Use `add-group-to-project-team` to associate a group with a project team instead.

### Roles
| Command | Required params | Optional params |
|---|---|---|
| `get-roles` | | |
| `add-role` | `name` | `description`, `permissions` |
| `update-role` | `roleId`, `permissions` | |
| `delete-role` | `roleId` | |
| `grant-role-to-user` | `userId`, `roleName` | `projectName` |
| `grant-role-to-group` | `groupName`, `roleName` | `projectName` |

### Permissions
| Command | Required params | Optional params |
|---|---|---|
| `get-permissions` | | |

### Agile boards & sprints
| Command | Required params | Optional params |
|---|---|---|
| `get-agile-boards` | | |
| `get-agile-board` | `boardId` | |
| `add-agile-board` | `name` | `template` (`scrum`/`kanban`), `projectIds` |
| `delete-agile-board` | `boardId` | |
| `add-sprint` | `boardId`, `name`, `startDate`, `endDate` | `goal` |

`startDate` and `endDate` are Unix timestamps in milliseconds.

### Dashboards
| Command | Required params |
|---|---|
| `get-dashboards` | |
| `add-dashboard` | `name` |
| `delete-dashboard` | `dashboardId` |
| `update-dashboard-name` | `dashboardId`, `newName` |

### Tags, work items, reactions, attachments
| Command | Required params | Optional params |
|---|---|---|
| `tag-issue` | `issueId`, `tagId` | |
| `delete-tag` | `tagId` | |
| `get-work-item-types` | | |
| `add-work-item` | `issueId`, `minutes` | `workType` |
| `get-attachments` | `issueId` | |
| `add-attachment` | `issueId`, `file` | `name` |
| `delete-attachment` | `issueId`, `attachmentId` | |
| `add-article-attachment` | `articleId`, `file` | `name` |
| `get-article-attachments` | `articleId` | |
| `add-reaction` | `issueId`, `commentId`, `reaction` | |
| `delete-reaction` | `issueId`, `commentId`, `reactionId` | |

`file` is the path to the local file. `name` overrides the filename used in YouTrack (defaults to the file's basename).
**Always use `add-attachment` / `add-article-attachment` for file uploads.**

### Workflows
| Command | Required params |
|---|---|
| `get-workflows` | |
| `add-workflow` | `name` |
| `add-workflow-rule` | `workflowId`, `title`, `script` |
| `delete-workflow` | `workflowId` |
| `restore-default-workflows` | |

### Custom fields & bundles
| Command | Required params | Optional params |
|---|---|---|
| `get-custom-field-prototypes` | | |
| `add-custom-field` | `name`, `fieldTypeId` | |
| `delete-custom-field` | `customFieldId` | |
| `add-bundle-value` | `bundleType`, `bundleId`, `name` | |

Use `get-custom-field-prototypes` to find `customFieldId`, `fieldTypeId`, `bundleType`, and `bundleId`.

### Global settings
| Command | Required params | Optional params |
|---|---|---|
| `get-global-settings` | | |
| `enable-feature-flag` | `feature`, `enabled` | |
| `set-license` | `username`, `license` | |
| `set-email-settings` | `host`, `port`, `fromAddress` | `replyToAddress` |
| `disable-email-settings` | | |
| `enable-global-banner` | `message` | |
| `disable-global-banner` | | |
| `set-reactions-enabled` | `enabled` | |
| `switch-jetbrains-ai` | `enabled` | |
| `switch-text-completion` | `enabled` | |
| `switch-grammar-correction` | `enabled` | |

### Raw API (escape hatch)
```
youtrack-cli raw-get --path "/api/issues?fields=id,summary&$top=5"
youtrack-cli raw-post --path "/api/issues/MYPROJ-1" --body '{"summary":"updated"}'
youtrack-cli raw-delete --path "/api/issues/MYPROJ-1"
```

Use these when no specific command covers the needed endpoint.

## YouTrack query syntax

Used by `get-issues` and `apply-command`:

```
assignee: me State: Open
assignee: me State: {In Progress}    # use braces for values with spaces
project: MYPROJ Priority: Critical
created: today
#MYPROJ-42                           # specific issue
tag: bug
```

## Common workflows

**Find and update an issue:**
```bash
youtrack-cli get-issues --query "assignee: me State: Open"
youtrack-cli apply-command --query "State Fixed" --issueIds "MYPROJ-42"
```

**Create an issue with a file description:**
```bash
youtrack-cli add-issue --projectShortName MYPROJ --summary "RFC: Auth" --descriptionFile ./rfc.md
```

**Link issues:**
```bash
youtrack-cli get-issue-link-types                          # find valid link types
youtrack-cli link-issues --issueId MYPROJ-42 --linkedIssueId MYPROJ-10 --linkType "depends on"
```

**Add a user to a project:**
```bash
youtrack-cli get-user --login john.doe                     # get userId
youtrack-cli add-user-to-project-team --projectShortName MYPROJ --userId <id>
```

**Create a group:**
```bash
youtrack-cli add-group --name "New Group"
```

**Grant a role:**
- If the user explicitly asks to grant a role globally, in global scope, or at global level etc., grant it in the global scope by omitting the `--projectName` parameter.
- If it is clear from the context that the role should be granted within a specific project, use that project with `--projectName`.
```bash
youtrack-cli get-roles                                     # find roleName
youtrack-cli get-user --login john.doe                     # get userId

# Grant in project scope
youtrack-cli grant-role-to-user --userId <id> --roleName Developer --projectName "My Project"

# Grant in global scope (omit --projectName)
youtrack-cli grant-role-to-user --userId <id> --roleName "System Admin"
```

**Create a new role:**
If the user explicitly asks to create a role, always create a new one with a unique name and do not attempt to find and use an existing role.
- **If the user did not specify permissions:** Create the role without the `--permissions` parameter.
  ```bash
  youtrack-cli add-role --name "New Role" --description "Role description"
  ```
- **If the user specified permissions (e.g., "read issue" or "create article"):** Find the required permission IDs first, then create the role.
  ```bash
  # 1. Get all available permissions
  youtrack-cli get-permissions

  # 2. Extract required permission IDs from the JSON (e.g., "JetBrains.YouTrack.READ_ISSUE")

  # 3. Create the role with a comma-separated list of IDs
  youtrack-cli add-role --name "New Role" --description "Role description" --permissions "JetBrains.YouTrack.READ_ISSUE,JetBrains.YouTrack.CREATE_ARTICLE"

**Update an existing role:**
- If the user explicitly asks to update or modify an existing role's permissions, use `update-role`.
  ```bash
  # 1. Get role ID (e.g., "17-6")
  youtrack-cli get-roles

  # 2. Get required permission IDs (e.g., "jetbrains.jetpass.organization-read")
  youtrack-cli get-permissions

  # 3. Update the role with a new comma-separated list of IDs (this REPLACES the current list)
  youtrack-cli update-role --roleId 17-6 --permissions "jetbrains.jetpass.organization-read,jetbrains.jetpass.organization-delete"
  ```

**Upload attachments:**
```bash
youtrack-cli add-attachment --issueId MYPROJ-1 --file ./screenshot.png
youtrack-cli add-article-attachment --articleId MYPROJ-A-1 --file ./report.pdf
```

**Create an issue as user A, comment as user B, assign user C (all in one project):**
```bash
# 1. Create project with user C as leader (ensures C is a valid assignee)
youtrack-cli add-project --shortName PROJ --name "My Project" --leader carol

# 2. Grant users A and B Contributor role so --asUser works for them
youtrack-cli get-user --login alice                          # get userId
youtrack-cli grant-role-to-user --userId <aliceId> --roleName Contributor --projectName "My Project"
youtrack-cli grant-role-to-user --userId <bobId> --roleName Contributor --projectName "My Project"

# 3. Create issue as user A
youtrack-cli add-issue --projectShortName PROJ --summary "Issue title" --asUser alice

# 4. Leave comment as user B
youtrack-cli add-comment --issueId PROJ-1 --text "My comment" --asUser bob

# 5. Assign user C (set-custom-field works because carol is a team member)
youtrack-cli set-custom-field --issueId PROJ-1 --fieldName Assignee --fieldValue carol --fieldType user
```

**Create entities on behalf of another user:**
```bash
youtrack-cli add-issue --projectShortName MYPROJ --summary "Bug report" --asUser alice.qa
youtrack-cli add-comment --issueId MYPROJ-1 --text "Investigated." --asUser bob.lead
youtrack-cli add-article --projectShortName MYPROJ --summary "How-to guide" --asUser carol.obs
youtrack-cli add-article-comment --articleId MYPROJ-A-1 --text "Feedback." --asUser alice.qa
```

**Set assignee via apply-command (logins with dots work automatically):**
```bash
youtrack-cli apply-command --query "assignee alice.qa" --issueIds "MYPROJ-1"
youtrack-cli apply-command --query "Type Bug assignee bob.lead State Open" --issueIds "MYPROJ-2"
```

## Known defaults and gotchas

- **Issue Type defaults to "Bug"**: `add-issue` does not accept a `type` param. Every newly created issue has Type=Bug regardless of content. When creating issues of mixed or non-Bug types, immediately follow each `add-issue` call with `set-custom-field --fieldName Type --fieldValue <type> --fieldType enum`. Valid Type values: `Bug`, `Feature`, `Task`, `Cosmetics`, `Exception`, `Usability Problem`, `Performance Problem`, `Epic`.
- **`add-project` requires `leader`**: the YouTrack API rejects project creation without a leader login. Always pass `--leader <login>`. If the user does not specify one, use `get-current-user` silently and use that login.
- **`get-issues` defaults to `$top=100`**: Pass `--limit <n>` to override. Use `--limit -1` for all results (may be slow on very large projects).
- **`apply-command` with dotted/hyphenated logins**: The CLI automatically wraps `assignee` and `reporter` values that contain `.`, `-`, or `@` in braces for YouTrack's command parser. Write `assignee alice.qa` — the CLI sends `assignee {alice.qa}` automatically.
- **File attachments — always use CLI commands**: Use `add-attachment --issueId ... --file ./path` for issues and `add-article-attachment --articleId ... --file ./path` for articles. Never use `curl` for uploads.
- **`--asUser <login>`**: `add-issue`, `add-comment`, `add-article`, and `add-article-comment` all accept `--asUser <login>` to set `reporter`/`author` on the created entity. Requires the current token to have admin permissions. The `--asUser` user must also have at least the **Contributor** role in the target project (use `grant-role-to-user` if needed).
- **Assignee field is restricted to project team members**: In Hub-backed YouTrack instances, the Assignee custom field only accepts users who are genuine project team members (added via Hub, not just via the REST API). `add-user-to-project-team` posts to the YouTrack REST endpoint but Hub does not persist these changes for users added solely through the REST API. **The only reliable way to make a user assignable is to create the project with that user as `--leader`** — project leaders are added to the Hub team at creation time. If you need multiple users to be assignable in the same project, create the project with one of them as leader and grant the others roles — they will be valid assignees only if they were also set as leader at creation, or if your YouTrack instance does not use Hub-backed auth.
- **`add-user-to-project-team` limitation**: On Hub-backed YouTrack instances, calling `add-user-to-project-team` returns success but the user is NOT actually added to the Hub project team and cannot be set as the Assignee. Use `grant-role-to-user` to grant project access instead (this allows `--asUser` for issue/comment creation), and use the `--leader` approach for assignability.

## Tips

- Always run a lookup command first when you need an internal ID (e.g. `get-user`, `get-projects`, `get-roles`).
- Use `--descriptionFile` / `--textFile` / `--contentFile` instead of inline text for long content.
- `permittedGroups` and `permittedUsers` accept comma-separated values.
- All boolean params (`enabled`, `directed`, `autoAttach`, etc.) accept the string `"true"` or `"false"`.
- Run `youtrack-cli list-commands` to see every registered command with its parameter list.
