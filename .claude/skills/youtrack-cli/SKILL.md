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
| `add-issue` | `projectShortName`, `summary` | `description`, `descriptionFile` |
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

### Comments
| Command | Required params | Optional params |
|---|---|---|
| `add-comment` | `issueId` | `text`, `textFile` |
| `get-comments` | `parentId` | `place` (`issues` or `articles`) |
| `update-comment` | `issueId`, `commentId` | `text`, `textFile` |
| `delete-comment` | `issueId`, `commentId` | |
| `update-comment-visibility` | `issueId`, `commentId` | `permittedGroups`, `permittedUsers` |
| `add-article-comment` | `articleId` | `text`, `textFile` |
| `delete-article-comment` | `articleId`, `commentId` | |

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
| `add-article` | `projectShortName`, `summary` | `content`, `contentFile` |
| `get-articles` | | `query` |
| `get-article` | `articleId` | |
| `delete-article` | `articleId` | |
| `update-article-summary` | `articleId`, `newSummary` | |
| `set-parent-article` | `articleId`, `parentArticleId` | |
| `update-article-visibility` | `articleId` | `permittedGroups`, `permittedUsers` |
| `tag-article` | `articleId`, `tagId` | |
| `get-removed-documents` | | |
| `restore-document` | `idReadable` | |

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
| `add-user` | `login`, `fullName`, `email` | `password` |
| `delete-user` | `userId` | |
| `ban-user` / `unban-user` | `userId` | |

### Groups & Hub
| Command | Required params | Optional params |
|---|---|---|
| `get-groups` | | |
| `get-group` | `groupName` | |
| `hub-add-user-to-group` | `userRingId`, `groupId` | |
| `hub-remove-user-from-group` | `userRingId`, `groupId` | |
| `hub-create-user` | `login`, `fullName`, `email`, `password` | |
| `hub-delete-user` | `userRingId` | |
| `hub-delete-group` | `groupName` | |

Hub commands require Ring IDs. Use `get-users` to find `ringId` and `get-groups` for group Ring IDs.

### Roles
| Command | Required params | Optional params |
|---|---|---|
| `get-roles` | | |
| `add-role` | `name` | `description` |
| `delete-role` | `roleId` | |
| `grant-role-to-user` | `userId`, `roleName`, `projectName` | |
| `grant-role-to-group` | `groupName`, `roleName`, `projectName` | |

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
| `delete-attachment` | `issueId`, `attachmentId` | |
| `add-reaction` | `issueId`, `commentId`, `reaction` | |
| `delete-reaction` | `issueId`, `commentId`, `reactionId` | |

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

**Grant a role:**
```bash
youtrack-cli get-roles                                     # find roleName
youtrack-cli get-user --login john.doe                     # get userId
youtrack-cli grant-role-to-user --userId <id> --roleName Developer --projectName "My Project"
```

## Known defaults and gotchas

- **Issue Type defaults to "Bug"**: `add-issue` does not accept a `type` param. Every newly created issue has Type=Bug regardless of content. When creating issues of mixed or non-Bug types, immediately follow each `add-issue` call with `set-custom-field --fieldName Type --fieldValue <type> --fieldType enum`. Valid Type values: `Bug`, `Feature`, `Task`, `Cosmetics`, `Exception`, `Usability Problem`, `Performance Problem`, `Epic`.
- **`add-project` requires `leader`**: the YouTrack API rejects project creation without a leader login. Always pass `--leader <login>`. If the user does not specify one, use `get-current-user` silently and use that login.

## Tips

- Always run a lookup command first when you need an internal ID (e.g. `get-user`, `get-projects`, `get-roles`).
- Use `--descriptionFile` / `--textFile` / `--contentFile` instead of inline text for long content.
- `permittedGroups` and `permittedUsers` accept comma-separated values.
- All boolean params (`enabled`, `directed`, `autoAttach`, etc.) accept the string `"true"` or `"false"`.
- Run `youtrack-cli list-commands` to see every registered command with its parameter list.
