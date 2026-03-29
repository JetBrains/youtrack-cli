# RFC: YouTrack CLI v2

## Context

The current `youtrack-cli` is a functional 131-command flat CLI built as a zero-dependency ESM Node.js tool. It works well for AI agents but falls short as a public product:

- **Flat command namespace** (`add-issue`, `get-issues`, `get-issue`) — hard to discover, no grouping
- **JSON-only output** — unfriendly for human terminal use
- **No installation packages** — requires Node.js + manual setup
- **No multi-server support** — one env file, one instance
- **No built-in help hierarchy** — `--help` dumps everything

This RFC defines the target state for **youtrack-cli v2** — a polished, public, open-source CLI for all YouTrack users, inspired by the TeamCity CLI.

---

## 1. Design Principles

1. **Hierarchical commands** — `youtrack-cli <group> <action> [target] [flags]`
2. **Human-first output** — readable tables by default; `--json` for machines
3. **Positional identifiers** — `youtrack-cli issue view JT-42`, not `--issueId JT-42`
4. **Consistent verbs** — `list`, `view`, `create`, `update`, `delete` across all groups
5. **Progressive disclosure** — core commands front and center; admin commands discoverable via `--help`
6. **AI-agent ready** — `--json`, structured errors, schema discovery, and draft-safe issue flows
7. **Zero-surprise** — destructive actions require confirmation (overridable with `--yes`/`-y`)

---

## 2. Binary & Invocation

```
youtrack-cli <group> <action> [positional-args...] [flags...]
```

**Binary name:** `youtrack-cli`

**Aliases:** Users can configure shell aliases (`alias yt='youtrack-cli'`).

---

## 3. Global Flags

| Flag | Short | Description |
|---|---|---|
| `--json` | `-j` | Output as JSON (default when stdout is not a TTY) |
| `--plain` | | Output as plain text — no colors, no borders, tab-separated. For `grep`/`awk` pipelines |
| `--quiet` | `-q` | Suppress all output; exit code only |
| `--verbose` | | Log HTTP requests/responses to stderr (credentials redacted) |
| `--yes` | `-y` | Skip confirmation prompts (for destructive actions) |
| `--server` | `-s` | Select named server profile (see `auth` group) |
| `--fields` | `-f` | Comma-separated field projection (dot paths: `id,summary,project.shortName`). `--fields ?` to discover available fields |
| `--help` | `-h` | Show help for group or command |
| `--version` | `-v` | Print CLI version |

**Environment variable overrides** (for CI/scripts — take precedence over saved profiles):

| Variable | Purpose |
|---|---|
| `YOUTRACK_BASE_URL` | Instance URL |
| `YOUTRACK_TOKEN` | Permanent token |
| `YOUTRACK_RO=1` | Read-only mode: blocks all write commands |

---

## 4. Output Format Spec

### 4.1 Default (TTY)

Human-readable tables with borders and colors. Lists show a subset of key columns; `view` commands show a detailed card.

```
$ youtrack-cli issue list --query "assignee: me State: Open" --limit 5

 ID       Summary                          State    Assignee       Updated
 JT-1042  Login button broken on Safari    Open     max.zh         2h ago
 JT-1038  API rate limiting not working     Open     max.zh         5h ago
 JT-1035  Dark mode toggle flickers        Open     max.zh         1d ago
```

```
$ youtrack-cli issue view JT-1042

 JT-1042  Login button broken on Safari
 ─────────────────────────────────────────────
 Project:    JT (YouTrack)
 State:      Open
 Priority:   Critical
 Assignee:   max.zh
 Created:    2026-03-28 14:30
 Updated:    2h ago

 Description:
   Reproducible on Safari 17, works fine on Chrome.
   Steps: 1. Open login page  2. Click "Sign in"  3. Nothing happens
```

### 4.2 `--json`

Full API response as JSON. Auto-enabled when stdout is piped.

```
$ youtrack-cli issue view JT-1042 --json
{
  "id": "2-1042",
  "idReadable": "JT-1042",
  "summary": "Login button broken on Safari",
  ...
}
```

### 4.3 `--plain`

Tab-separated, no headers, no colors. For scripting pipelines.

```
$ youtrack-cli issue list --query "State: Open" --plain
JT-1042	Login button broken on Safari	Open	max.zh
JT-1038	API rate limiting not working	Open	max.zh
```

### 4.4 Errors

Errors go to stderr. Exit code 1. In `--json` mode, structured error object:

```json
{ "error": "Issue not found: JT-99999", "code": 404, "suggestion": "Check the issue ID" }
```

In default mode, a colored human-readable message:

```
Error: Issue not found: JT-99999
Hint: Check the issue ID and ensure you have access to the project.
```

---

## 5. Authentication & Multi-Server

### 5.1 Commands

| Command | Description |
|---|---|
| `youtrack-cli auth login` | Interactive: prompt for URL + token (or login/password). Saves as named profile. |
| `youtrack-cli auth login --url <url> --token <token>` | Non-interactive login |
| `youtrack-cli auth login --url <url> --login <l> --password <p>` | Login with credentials (generates permanent token) |
| `youtrack-cli auth status` | Show current user and server |
| `youtrack-cli auth logout` | Remove saved credentials for current (or `--server <name>`) profile |
| `youtrack-cli auth list` | List all saved server profiles |
| `youtrack-cli auth switch <name>` | Set default server profile |

### 5.2 Credential Storage

Profiles stored in `~/.config/youtrack-cli/config.json`:

```json
{
  "defaultServer": "cloud",
  "servers": {
    "cloud": {
      "url": "https://myteam.youtrack.cloud",
      "token": "perm:..."
    },
    "onprem": {
      "url": "https://youtrack.internal.corp",
      "token": "perm:..."
    }
  }
}
```

**Precedence:** env vars > `--server` flag > `defaultServer` in config.

**Migration:** On first run, if `~/.youtrack-cli-env` exists, auto-import it as the `default` profile.

---

## 6. Command Reference

### 6.1 `issue` — Issue tracking

Built-in issue properties such as `project`, `created`, and `updated` are selected via `issue view --fields`.
Mutable custom fields live under `issue custom-field ...`, with the field schema discovered through `project schema`.

| Command | Positional | Flags | Description |
|---|---|---|---|
| `issue list` | | `--query`, `--limit` (default 25) | Search issues |
| `issue view <id>` | issue ID or draft ID | `--comments <n>` | View issue or draft details; optionally inline the `n` most recent comments |
| `issue create` | | `--project` (req), `--summary` (req), `--description`, `--description-file`, `--custom-fields`, `--custom-fields-file`, `--parent`, `--as-user` | Create issue |
| `issue draft create` | | `--project` (req), `--summary` (req) | Create issue draft |
| `issue update <id>` | issue ID or draft ID | `--summary`, `--description`, `--description-file`, `--custom-fields`, `--custom-fields-file` | Update issue or draft |
| `issue delete <id>` | issue ID or draft ID | `--yes` | Delete issue or draft |
| `issue command <ids> <query>` | comma-separated IDs, command query | | Apply YouTrack command (e.g., `"State Fixed"`) |
| `issue suggest <ids> <query>` | comma-separated IDs, partial query | | Get command auto-complete suggestions |
| `issue custom-field get <id> <field>` | issue ID or draft ID, field name | | Get custom field value |
| `issue custom-field set <id> <field>` | issue ID or draft ID, field name | `--value` (req), `--type` (req: `enum`/`user`/`period`/`text`/`int`/`string`) | Set custom field value |
| `issue assign <id> <user>` | issue ID, user login | | Change assignee |
| `issue unassign <id>` | issue ID | | Clear assignee |
| `issue visibility <id>` | issue ID | `--groups`, `--users` | Set issue visibility (empty = visible to all) |
| `issue watch <id>` | issue ID | | Subscribe to issue updates |
| `issue unwatch <id>` | issue ID | | Remove issue subscription |
| `issue vote <id>` | issue ID | | Vote for issue |
| `issue unvote <id>` | issue ID | | Remove vote |

### 6.2 `link` — Issue links & link types

| Command | Positional | Flags | Description |
|---|---|---|---|
| `link list <issueId>` | issue ID | `--type` | List linked issues |
| `link add <issueId> <targetId>` | issue ID, target issue ID | `--type` (req, e.g. `"Subtask"`) | Link two issues |
| `link remove <issueId> <targetId>` | issue ID, target issue ID | `--type` (req) | Remove issue link |
| `link types` | | | List all available link types |
| `link create <name> <sourceToTarget>` | name, source-to-target label | `--target-to-source`, `--directed`, `--aggregation` | Create link type |

### 6.3 `comment` — Comments on issues and articles

| Command | Positional | Flags | Description |
|---|---|---|---|
| `comment list <parentId>` | issue or article ID | `--on` (`issue`\|`article`, default `issue`), `--offset`, `--limit` (default 10) | List comments |
| `comment add <parentId>` | issue or article ID | `--text`, `--text-file`, `--on`, `--as-user` | Add comment |
| `comment update <parentId> <commentId>` | parent ID, comment ID | `--text`, `--text-file`, `--on` | Edit comment |
| `comment delete <parentId> <commentId>` | parent ID, comment ID | `--on` | Delete comment |
| `comment pin <parentId> <commentId>` | parent ID, comment ID | `--on` | Pin comment |
| `comment unpin <parentId> <commentId>` | parent ID, comment ID | `--on` | Unpin comment |
| `comment visibility <parentId> <commentId>` | parent ID, comment ID | `--on`, `--groups`, `--users` | Set comment visibility |
| `comment reaction add <parentId> <commentId> <reaction>` | parent ID, comment ID, reaction | `--on` | Add reaction |
| `comment reaction delete <parentId> <commentId> <reactionId>` | parent ID, comment ID, reaction ID | `--on` | Remove reaction |

### 6.4 `article` — Knowledge base

| Command | Positional | Flags | Description |
|---|---|---|---|
| `article list` | | `--query` | List/search articles |
| `article view <id>` | article ID | | View article |
| `article create` | | `--project` (req), `--summary` (req), `--content`, `--content-file`, `--as-user` | Create article |
| `article update <id>` | article ID | `--summary`, `--content`, `--content-file` | Update article |
| `article delete <id>` | article ID | `--yes` | Delete article |
| `article move <id>` | article ID | `--parent` (req) | Set parent article |
| `article visibility <id>` | article ID | `--groups`, `--users` | Set visibility |
| `article tag <id> <tagId>` | article ID, tag ID | | Tag article |

### 6.5 `attachment` — File attachments

| Command | Positional | Flags | Description |
|---|---|---|---|
| `attachment list <parentId>` | issue or article ID | `--on` (`issue`\|`article`, default `issue`) | List attachments |
| `attachment add <parentId> <file>` | parent ID, file path | `--on`, `--name` | Upload attachment |
| `attachment delete <parentId> <attachmentId>` | parent ID, attachment ID | `--on` | Delete attachment |

### 6.6 `project` — Project administration

This group mirrors the main Project Settings areas in the YouTrack UI: overview, fields, team, apps, and time tracking.
The most important machine-readable entrypoint is `project schema`, which exposes the project issue field schema in the same spirit as the YouTrack MCP toolset.

| Command | Positional | Flags | Description |
|---|---|---|---|
| `project list` | | `--query` | List/search projects |
| `project view <key>` | project key | | View project details, including leader, team, archive state, and time tracking basics |
| `project create` | | `--short-name` (req), `--name` (req), `--leader` (req) | Create project |
| `project update <key>` | project key | `--name`, `--leader`, `--archive` | Update general project settings |
| `project delete <id>` | project internal ID | `--yes` | Delete project |
| `project templates` | | | List project templates |
| `project convert-to-template <id>` | project internal ID | | Convert project to template |
| `project schema <key>` | project key | | Return issue custom field JSON schema for the project |
| `project field list <key>` | project key | | List project custom fields |
| `project field attach <key> <fieldId>` | project key, field ID | | Attach custom field to project |
| `project field detach <key> <fieldId>` | project key, field ID | | Remove custom field from project |
| `project team users <key>` | project key | | List project team users |
| `project team add-user <key> <user>` | project key, user login | | Add user to project team |
| `project team remove-user <key> <user>` | project key, user login | | Remove user from project team |
| `project team groups <key>` | project key | | List project team groups |
| `project team add-group <key> <group>` | project key, group name | | Add group to project team |
| `project team remove-group <key> <group>` | project key, group name | | Remove group from project team |
| `project app list <key>` | project key | | List apps attached to project |
| `project app add <key> <appId>` | project key, app ID | | Attach app |
| `project app remove <key> <appId>` | project key, app ID | | Detach app |
| `project time-tracking <key>` | project key | `--enabled`, `--spent-time-field`, `--estimation-field` | Configure project time tracking |
| `project workitem-attributes <key>` | project key | | List work item types and attribute schema for the project |

### 6.7 `user` — User management

| Command | Positional | Flags | Description |
|---|---|---|---|
| `user me` | | | Current authenticated user |
| `user list` | | `--query` | List/search users |
| `user view <login>` | user login | | View user profile |
| `user create` | | `--login` (req), `--name` (req), `--email` (req), `--password`, `--type` (`standard`/`agent`/`reporter`) | Create user |
| `user delete <id>` | user ID | `--successor`, `--yes` | Delete user |
| `user ban <id>` | user ID | | Ban user |
| `user unban <id>` | user ID | | Unban user |
| `user setting <key> <value>` | setting key, value | `--user` (default: me) | Set profile setting |
| `user locale <locale>` | locale string | `--user` | Set locale |
| `user timezone <tz>` | timezone ID | `--user` | Set timezone |
| `user timezone` | | `--user` | Get timezone |
| `user drafts` | | `--user`, `--type` (`issue`\|`article`) | List drafts |
| `user delete-drafts` | | `--user` | Delete all issue drafts |
| `user pin-project <projectId>` | project ID | `--user` | Pin project |

### 6.8 `group` — User groups

| Command | Positional | Flags | Description |
|---|---|---|---|
| `group list` | | `--query` | List groups |
| `group view <name>` | group name | | View group |
| `group create <name>` | group name | | Create group |
| `group delete <name>` | group name | `--successor`, `--yes` | Delete group |

### 6.9 `role` — Roles & permissions

| Command | Positional | Flags | Description |
|---|---|---|---|
| `role list` | | | List roles |
| `role create <name>` | role name | `--description`, `--permissions` (comma-sep) | Create role |
| `role update <id>` | role ID | `--permissions` (replaces full list) | Update role permissions |
| `role delete <id>` | role ID | `--yes` | Delete role |
| `role grant` | | `--role` (req), `--user` or `--group` (req), `--project` (optional; omit for global) | Grant role |
| `role permissions` | | | List all available permissions |

### 6.10 `field` — Custom field prototypes & bundles

| Command | Positional | Flags | Description |
|---|---|---|---|
| `field list` | | | List custom field prototypes |
| `field create <name> <typeId>` | field name, type ID | | Create custom field prototype |
| `field delete <id>` | field ID | `--yes` | Delete field prototype |
| `field privacy <id>` | field ID | `--public` (`true`/`false`) | Set field public/private |
| `field bundle list` | | `--type`, `--offset`, `--limit` | List bundles |
| `field bundle view <bundleId>` | bundle ID | | View bundle metadata |
| `field bundle values <bundleId>` | bundle ID | `--offset`, `--limit` | List bundle values |
| `field bundle create <type>` | bundle type | `--name` | Create bundle |
| `field bundle add-value <bundleId> <value>` | bundle ID, value | | Add bundle value |
| `field bundle remove-value <bundleId> <value>` | bundle ID, value | | Remove bundle value |
| `field bundle attach <fieldId> <bundleId>` | field ID, bundle ID | | Attach bundle to field |

### 6.11 `tag` — Tags

| Command | Positional | Flags | Description |
|---|---|---|---|
| `tag list` | | | List tags |
| `tag create <name>` | tag name | `--owner` (req), `--color`, `--shareable`, `--updatable`, `--untag-on-resolve`, `--pinned` | Create tag |
| `tag delete <id>` | tag ID | `--yes` | Delete tag |
| `tag apply <targetId> <tagId>` | target (issue/article) ID, tag ID | `--on` (`issue`\|`article`, default `issue`) | Apply tag |
| `tag remove <targetId> <tagId>` | target (issue/article) ID, tag ID | `--on` (`issue`\|`article`, default `issue`) | Remove tag |

### 6.12 `board` — Agile boards & sprints

| Command | Positional | Flags | Description |
|---|---|---|---|
| `board list` | | `--offset`, `--limit` | List favorite agile boards |
| `board search` | | `--query`, `--offset`, `--limit` | Search agile boards |
| `board view <id>` | board ID | | View board |
| `board create <name>` | board name | `--template` (`scrum`/`kanban`), `--projects` (comma-sep) | Create board |
| `board delete <id>` | board ID | | Delete board |
| `board sprint list <boardId>` | board ID | `--offset`, `--limit` | List sprints |
| `board sprint add <boardId>` | board ID | `--name` (req), `--start` (req), `--end` (req), `--goal` | Add sprint |
| `board issue add <boardId> <issueId>` | board ID, issue ID | `--sprint` | Add issue to board; defaults to current sprint when available |
| `board issue remove <boardId> <issueId>` | board ID, issue ID | `--sprint` | Remove issue from board or sprint |

### 6.13 `dashboard` — Dashboards

| Command | Positional | Flags | Description |
|---|---|---|---|
| `dashboard list` | | | List dashboards |
| `dashboard create <name>` | name | | Create dashboard |
| `dashboard rename <id> <name>` | dashboard ID, new name | | Rename dashboard |
| `dashboard delete <id>` | dashboard ID | `--yes` | Delete dashboard |

### 6.14 `workitem` — Time tracking & work items

| Command | Positional | Flags | Description |
|---|---|---|---|
| `workitem add <issueId>` | issue ID | `--minutes` (req), `--type`, `--text`, `--author`, `--date`, `--attributes` | Log work item with optional custom attributes |
| `workitem types <projectKey>` | project key | | List project work item types |
| `workitem attributes <projectKey>` | project key | | List project work item attributes and allowed values |
| `workitem settings <projectKey>` | project key | | View project time tracking settings |
| `workitem add-type <projectKey> <name>` | project key, type name | `--auto-attach` | Create work item type |
| `workitem set-hours <n>` | hours per day | | Set hours-in-a-day |

### 6.15 `workflow` — Workflows & saved searches

| Command | Positional | Flags | Description |
|---|---|---|---|
| `workflow list` | | | List workflows |
| `workflow create` | | `--name` (req), `--title` (req) | Create workflow |
| `workflow delete <id>` | workflow ID | `--yes` | Delete workflow |
| `workflow add-rule <id>` | workflow ID | `--name` (req), `--script` (req) | Add rule to workflow |
| `workflow restore-defaults` | | | Restore default workflows |
| `workflow search list` | | | List saved issue searches |
| `workflow search save` | | `--query` (req), `--name` (req) | Create saved search |
| `workflow search share <id>` | search ID | `--users`, `--groups` | Share saved search |

### 6.16 `settings` — Global / admin settings

| Command | Positional | Flags | Description |
|---|---|---|---|
| `settings view` | | | View global settings |
| `settings frontend` | | | View frontend config |
| `settings feature <flag>` | feature flag name | `--enabled` (default `true`) | Toggle feature flag |
| `settings license` | | `--username` (req), `--license` (req) | Set license |
| `settings email` | | `--server` (req), `--port` (req), `--from` (req), `--reply-to` | Configure email |
| `settings email disable` | | | Disable email |
| `settings banner <message>` | banner text | | Set global banner |
| `settings banner disable` | | | Disable banner |
| `settings reactions <on\|off>` | on/off | | Toggle reactions |
| `settings ai <on\|off>` | on/off | | Toggle JetBrains AI |
| `settings ai-completion <on\|off>` | on/off | | Toggle AI text completion |
| `settings ai-grammar <on\|off>` | on/off | | Toggle AI grammar correction |

### 6.17 `apps` — Apps & plugins

| Command | Positional | Flags | Description |
|---|---|---|---|
| `apps list` | | | List apps |
| `apps view <id>` | app ID | | View app details |
| `apps delete <id>` | app ID | `--yes` | Delete app |
| `apps deactivate <id>` | app ID | | Deactivate app |
| `apps plugins` | | | List plugin usages |
| `apps delete-plugin <id>` | plugin ID | `--yes` | Remove plugin usage |
| `apps script list <appId>` | app ID | | List app scripts |
| `apps script add <appId>` | app ID | `--name` (req), `--file` (req) | Add script to app |
| `apps script update <appId> <scriptId>` | app ID, script ID | `--name`, `--file` | Update app script |
| `apps script delete <appId> <scriptId>` | app ID, script ID | | Delete app script |

### 6.18 `gantt` — Gantt charts

| Command | Positional | Flags | Description |
|---|---|---|---|
| `gantt list` | | | List Gantt charts |
| `gantt view <id>` | chart ID | | View chart details |
| `gantt create <name>` | chart name | `--projects` (comma-sep) | Create Gantt chart |
| `gantt issue add <chartId> <issueId>` | chart ID, issue ID | | Add issue to chart |
| `gantt issue remove <chartId> <issueId>` | chart ID, issue ID | | Remove issue from chart |
| `gantt issue move <chartId> <issueId>` | chart ID, issue ID | `--parent`, `--after`, `--before` | Move or reorder issue in chart |
| `gantt link <chartId> <source> <target>` | chart ID, source issue ID, target issue ID | `--type` | Link issues inside chart |
| `gantt unlink <chartId> <source> <target>` | chart ID, source issue ID, target issue ID | | Remove issue dependency from chart |
| `gantt delete <id>` | chart ID | `--yes` | Delete chart |

### 6.19 `server` — Instance information

| Command | Positional | Flags | Description |
|---|---|---|---|
| `server info` | | | Show server version, build, base URL, and installation timestamp |

### 6.20 `api` — Raw REST API

| Command | Positional | Flags | Description |
|---|---|---|---|
| `api <method> <path>` | HTTP method, API path | `--body`, `--api-fields`, `--top`, `--paginate` | Direct API call |

---

## 7. Flag Naming Conventions

| Convention | Examples |
|---|---|
| All flags are `--kebab-case` | `--short-name`, `--base-url`, `--as-user` |
| Boolean flags are bare (no value = true) | `--verbose`, `--yes`, `--paginate` |
| Explicit boolean off: `--no-<flag>` | `--no-color` |
| File variants: `--<flag>-file` | `--description-file ./notes.md`, `--text-file ./msg.md` |
| Comma-separated lists | `--groups "Dev,QA"`, `--permissions "perm1,perm2"` |
| Positional args for primary identifiers | `issue view JT-42`, not `issue view --id JT-42` |

### Migration from v1 flag names

| v1 (camelCase) | v2 (kebab-case) |
|---|---|
| `--projectShortName` | `--project` |
| `--issueId` | positional `<id>` |
| `--parentId` | positional `<parentId>` |
| `--fieldName` | positional `<field>` |
| `--fieldValue` | `--value` |
| `--fieldType` | `--type` |
| `--permittedGroups` | `--groups` |
| `--permittedUsers` | `--users` |
| `--setEnv` | `--save` (in `auth login`) |
| `--baseUrl` | `--url` (in `auth login`) |

---

## 8. Help System

### Group-level help

```
$ youtrack-cli issue --help

Manage YouTrack issues — create, search, update, and track.

Usage:
  youtrack-cli issue <command> [flags]

Commands:
  list          Search issues
  view          View issue details
  create        Create a new issue
  draft         Create issue drafts
  update        Update an issue or draft
  delete        Delete an issue or draft
  command       Apply YouTrack command
  suggest       Get command suggestions
  custom-field  Get or set custom field values
  assign        Change the assignee
  unassign      Clear the assignee
  visibility    Set issue visibility
  watch         Subscribe to issue updates
  unwatch       Remove issue subscription
  vote          Vote for an issue
  unvote        Remove vote from an issue

Run 'youtrack-cli issue <command> --help' for details on a specific command.
```

### Command-level help

```
$ youtrack-cli issue create --help

Create a new issue in a YouTrack project.

Usage:
  youtrack-cli issue create --project <shortName> --summary <text> [flags]

Required flags:
  --project           Project short name (e.g., JT)
  --summary           Issue summary

Optional flags:
  --description       Issue description (Markdown)
  --description-file  Read description from file
  --custom-fields     Inline JSON object with custom field values
  --custom-fields-file Read custom field values from JSON file
  --parent            Parent issue ID
  --as-user           Create on behalf of another user (requires admin token)

Examples:
  youtrack-cli issue create --project JT --summary "Login broken on Safari"
  youtrack-cli issue create --project JT --summary "RFC: Auth" --description-file ./rfc.md
  youtrack-cli issue create --project JT --summary "Incident" --custom-fields-file ./fields.json
```

### Top-level help

```
$ youtrack-cli --help

YouTrack CLI — command-line interface for YouTrack

Usage:
  youtrack-cli <command> [flags]

Core commands:
  issue         Manage issues
  link          Manage issue links and link types
  comment       Manage comments
  article       Manage knowledge base articles
  project       Manage project settings and schema

Planning:
  board         Agile boards and sprints
  dashboard     Manage dashboards
  gantt         Gantt charts

Work tracking:
  workitem      Log work and manage time tracking

Administration:
  user          Manage users
  group         Manage user groups
  role          Manage roles and permissions
  field         Custom field prototypes and bundles
  tag           Manage tags
  workflow      Manage workflows and saved searches
  settings      Global and admin settings

Integrations:
  apps          Apps and plugins

Utilities:
  auth          Authenticate and manage server profiles
  server        Show server version and installation info
  api           Raw REST API access

Global flags:
  --json, -j    Output as JSON
  --plain       Tab-separated output, no colors
  --quiet, -q   Suppress output
  --verbose     Log HTTP traffic to stderr
  --yes, -y     Skip confirmation prompts
  --server, -s  Select server profile
  --fields, -f  Field projection (comma-separated)
  --help, -h    Show help
  --version, -v Show version

Run 'youtrack-cli <command> --help' for more information on a command.
```

---

## 9. Confirmation & Safety

### Destructive commands require confirmation

```
$ youtrack-cli issue delete JT-42
Delete issue JT-42 "Login button broken on Safari"? [y/N]: y
Deleted JT-42.
```

```
$ youtrack-cli issue delete JT-42 --yes
Deleted JT-42.
```

### Read-only mode

When `YOUTRACK_RO=1`, all write commands exit with error before making any HTTP call:

```
$ YOUTRACK_RO=1 youtrack-cli issue create --project JT --summary "test"
Error: Read-only mode is enabled (YOUTRACK_RO=1). Write operations are blocked.
```

---

## 10. Configuration File

Location: `~/.config/youtrack-cli/config.json` (XDG-compliant; respects `XDG_CONFIG_HOME`)

```json
{
  "defaultServer": "cloud",
  "servers": {
    "cloud": {
      "url": "https://myteam.youtrack.cloud",
      "token": "perm:..."
    }
  },
  "defaults": {
    "limit": 25,
    "project": "JT"
  },
  "aliases": {
    "my-issues": "issue list --query \"assignee: me State: Open\""
  }
}
```

### Defaults

The `defaults` section provides default values for common flags:
- `defaults.limit` — default `--limit` for list commands
- `defaults.project` — default `--project` for create commands

### Aliases

User-defined command shortcuts:

```
$ youtrack-cli alias set my-issues "issue list --query \"assignee: me State: Open\""
$ youtrack-cli my-issues
```

---

## 11. AI Agent Integration

The CLI exposes the same high-value building blocks that already power the YouTrack MCP-style toolset:

- `project schema <key>` returns project issue field schema for safe issue creation and updates.
- `project workitem-attributes <key>` exposes work item types and attribute schema for time tracking.
- `issue draft create` gives agents a safe staging flow before creating a real issue.
- `issue view <id> --comments <n>` and `comment list --limit <n>` keep comment reads explicit and bounded.

### Machine-friendly behavior

When `--json` is active (or stdout is piped):
- All output is valid JSON
- Errors are structured: `{ "error": "...", "code": N }`
- No interactive prompts (equivalent to `--yes`)
- No progress spinners or color codes

---

## 12. Installation

### npm (primary — works everywhere Node.js runs)

```bash
npm install -g youtrack-cli
```

### macOS (Homebrew)

```bash
brew install jetbrains/utils/youtrack-cli
```

### Linux

```bash
curl -fsSL https://jb.gg/yt/install | bash
```

### Windows (Winget)

```powershell
winget install JetBrains.YouTrackCLI
```

<details>
<summary>More installation methods</summary>

**npx (no install):**
```bash
npx youtrack-cli issue list --query "State: Open"
```

**Debian/Ubuntu:**
```bash
curl -fsSLO https://github.com/JetBrains/youtrack-cli/releases/latest/download/youtrack-cli_linux_amd64.deb
sudo dpkg -i youtrack-cli_linux_amd64.deb
```

</details>

---

## 13. v1 → v2 Migration

### Backward compatibility

The v1 flat command names (`add-issue`, `get-issues`, etc.) will be supported as hidden aliases during a deprecation period. When used, they print a deprecation warning to stderr:

```
$ youtrack-cli add-issue --projectShortName JT --summary "test"
⚠ Deprecated: 'add-issue' is now 'issue create'. See 'youtrack-cli issue create --help'.
```

This allows existing scripts and AI agent skills to continue working while migrating.

### Command mapping (v1 → v2)

Full mapping of all 131 v1 commands to v2 equivalents is provided in the migration guide (not included here for brevity — derivable from sections 6.1–6.20 above).

---

## 14. Non-Goals (out of scope for v2)

- **Interactive / TUI mode** — no curses-based UI; keep it a simple CLI
- **Shell completions** — desirable but deferred to v2.1
- **Webhooks / real-time streaming** — YouTrack doesn't expose WebSocket endpoints
- **Offline mode** — all commands require network access
- **Plugin system** — `api` command covers arbitrary endpoints

---

## 15. Open Questions

1. **Should `--project` be inferrable from git branch?** (e.g., branch `JT-123-fix-bug` → project `JT`)
2. **Pagination UX**: auto-paginate by default, or require `--all`/`--paginate`?
3. **Should `project view` stay concise while schema-heavy output lives in `project schema` / `project workitem-attributes`, or do we also need a `--full` mode?**

---

## Summary of Key Changes from v1

| Aspect | v1 (current) | v2 (target) |
|---|---|---|
| Command style | Flat: `add-issue`, `get-issues` | Hierarchical: `issue create`, `issue list` |
| Flag names | camelCase: `--projectShortName` | kebab-case: `--project` |
| Primary IDs | Flags: `--issueId JT-42` | Positional: `issue view JT-42` |
| Output | JSON always | Human tables (default), `--json`, `--plain` |
| Auth | Single env file | Multi-server profiles + env var override |
| Destructive ops | No confirmation | Confirmation prompt (bypass with `--yes`) |
| Help | Flat `--help` dumps all | Hierarchical per-group and per-command |
| Installation | `npm link` only | npm, brew, apt, winget, npx |
| Drafts | Implicit / manual | First-class `issue draft create`; draft IDs work in `view`, `update`, and `delete` |
| Project schema | Ad hoc | `project schema` and `project workitem-attributes` expose structured metadata |
| Links | Mixed into issue commands | Dedicated `link` group with link-type discovery |
| Server metadata | Not surfaced | `server info` exposes version and installation details |
| AI integration | Manual skill files | Structured schema discovery, bounded comment fetches, and draft-safe flows |
| Aliases | None | User-defined command shortcuts |
