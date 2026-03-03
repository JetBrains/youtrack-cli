# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Running the CLI

```bash
# Run directly
node youtrack-cli.js <command> [--param value ...]

# Or via npm
npm start -- <command> [--param value ...]

# List all available commands
node youtrack-cli.js list-commands

# Show full help
node youtrack-cli.js --help
```

## Environment Setup

Two env vars are required for most commands:

```bash
export YOUTRACK_BASE_URL=https://your-instance.youtrack.cloud
export YOUTRACK_TOKEN=perm:...
```

Alternatively, run `get-token --login ... --password ... --setEnv true` which writes `~/.youtrack-cli-env` (loaded automatically on each invocation).

## YouTrack Concepts

**YouTrack** is a JetBrains project management tool. Key domain concepts:

- **Issue** — the primary work item. Identified by `<PROJECT>-<number>` (e.g. `FOO-42`). Has a summary, description, state, priority, assignee, and arbitrary custom fields.
- **Project** — a container for issues and articles. Identified by a short name (e.g. `FOO`). Has teams, custom fields, workflows, and time tracking settings.
- **Article** — knowledge base document, scoped to a project. Can be nested (parent/child). Supports visibility, tags, attachments, comments, and drafts.
- **Custom Field** — typed metadata on issues (e.g. state, priority, fix versions). Field types: `state`, `enum`, `user`, `date`, `integer`, `float`, `period`, `string`, `text`, `version`, `build`. Bundles hold the allowed values for enum/version/build/state fields.
- **Command** — YouTrack's text-based shorthand to update issues in bulk (e.g. `State Fixed assignee me`). Use `apply-command` to run; `get-command-assist` to validate/autocomplete.
- **Tag** — label attached to issues or articles for filtering.
- **Work Item** — a time log entry on an issue. Requires `minutes`, optional `workType` and `text`.
- **Saved Search** — a named query stored per user or shared with groups.
- **Agile Board** — a Kanban/Scrum board backed by a project and optionally organized into **Sprints**.
- **Dashboard** — a customizable home screen with widgets.
- **Workflow** — server-side automation scripts attached to projects.
- **Reaction** — emoji reaction on a comment.
- **Attachment** — file uploaded to an issue or article.
- **Role** — a named permission set that can be granted to users or groups at project or global scope. Permissions (e.g. `jetbrains.jetpass.project-read`) define what operations are allowed on entities.
- **Group** — a user group. Read and deleted via the YouTrack `/api/groups` REST API. Group membership modification is not supported by the YouTrack REST API (it requires Hub-scoped auth). Use `add-group-to-project-team` to associate a group with a project.
- **VCS Integration** — links Git repos (GitHub, GitLab, etc.) to YouTrack projects for commit-to-issue tracking.
- **Mailbox** — email-to-issue integration (create issues from incoming email).
- **Canvas** — a freeform planning board with card nodes and links.
- **Gantt Chart** — a timeline view of issues for scheduling.
- **Organization** — top-level grouping above projects (YouTrack Cloud feature).

### ID conventions

| Entity | Identifier | Example |
|---|---|---|
| Issue | `<PROJECT>-<n>` | `FOO-42` |
| Project | short name | `FOO` |
| User | login | `john.doe` |
| Group | name | `Developers` |
| Role | name | `Developer` |
| Everything else | opaque `id` field from GET responses | `2-7` |

### Visibility

`permittedGroups` and `permittedUsers` are comma-separated names/logins. An empty value means "visible to all". Visibility applies to issues, comments, and articles.

### Text content

Commands that accept `?description`, `?content`, or `?text` also accept a `...File` variant (e.g. `--descriptionFile ./notes.md`) for passing multi-line content from a file.

## Architecture

The project is a **zero-dependency, ESM Node.js CLI** designed for use by AI agents and developers interacting with the YouTrack REST API. There are three source files:

| File | Role |
|---|---|
| `youtrack-cli.js` | Command registry and `main()` entrypoint |
| `http-helpers.js` | HTTP primitives (`doGet`, `doPost`, `doPut`, `doDelete`, `httpReqBasicAuth`) and data-parsing utilities |
| `api-paths.js` | Named constants for all YouTrack REST API paths (`API.*`) |

### Command registration pattern

Every command is registered with `cmd(name, description, params[], handler)`:

- `params` entries prefixed with `?` are optional (e.g., `"?description"`).
- The handler receives `(ctx, args)` where `ctx = { base, token }` and `args` maps param names to CLI-supplied values.
- All handlers are `async` and return JSON-serializable data, which `main()` prints as pretty JSON.

### Argument parsing

CLI arguments are `--key value` pairs. Kebab-case flags are converted to camelCase before lookup (e.g., `--base-url` → `baseUrl`). Required params are validated before the handler runs; missing ones exit with a usage message.

### Auth

- Most commands use Bearer token auth via the `Authorization: Bearer <token>` header.
- `get-token` uses HTTP Basic auth (`httpReqBasicAuth`) to exchange login/password for a permanent token.
- `list-commands` and `get-token` are the only commands that skip token validation.

### Command domains

Commands are grouped in `youtrack-cli.js` by section comments:
- **Issues** — CRUD, custom fields, visibility, commands, links
- **Comments / Article Comments** — add, update, delete, visibility
- **Articles** — CRUD, parent, visibility, tagging, drafts, trash/restore
- **Projects** — CRUD, templates, teams, custom fields, time tracking, workflows
- **Users / User Profile** — CRUD, ban, profile settings (appearance, locale, timezone, notifications, Grazie), drafts, pinned projects
- **Groups** — list, lookup, add/remove members
- **Roles** — CRUD, grant to user/group
- **Dashboards / Agile Boards / Sprints** — CRUD
- **Tags / Work Items / Reactions / Attachments** — targeted CRUD
- **Workflows** — list, create, add rules, restore defaults
- **Global Settings** — feature flags, license, email, banners, AI toggles
- **Saved Searches / Reports / Apps / Plugins** — CRUD
- **VCS / Mailbox / Organizations** — integration management
- **Gantt / Canvas** — charts and planning canvases
- **Custom Fields / Bundles** — field prototypes and enum values
- **Raw API** — `raw-get`, `raw-post`, `raw-delete` for arbitrary calls
