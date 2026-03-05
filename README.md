# youtrack-cli

A command-line tool for interacting with YouTrack via its REST API. Designed for use by developers and AI agents — create issues, manage projects, control users, and automate anything YouTrack exposes through its API.

## Prerequisites

- **Node.js 18+** — run `node --version` to check
- A YouTrack instance URL and either a permanent token or your login credentials

## Setup

### 1. Get your credentials ready

If you already have a permanent token (available in YouTrack under your profile → **Authentication**), export these two variables in your terminal:

```bash
export YOUTRACK_BASE_URL=https://your-instance.youtrack.cloud
export YOUTRACK_TOKEN=perm:...
```

If you only have a login and password, the tool can generate a token for you and save it automatically:

```bash
node youtrack-cli.js get-token \
  --base-url https://your-instance.youtrack.cloud \
  --login your-login \
  --password your-password \
  --setEnv true
```

This writes your URL and token to `~/.youtrack-cli-env`, which the tool loads automatically on every run — no need to set environment variables each time.

### 2. Install the global command (optional)

Run once to make `youtrack-cli` available as a global command:

```bash
npm run setup
```

After that you can drop the `node youtrack-cli.js` prefix and use the shorter form:

```bash
youtrack-cli <command> --param1 value1 --param2 value2
```

### 3. Run a command

```bash
# Short form (after npm run setup)
youtrack-cli <command> --param1 value1 --param2 value2

# Without global install
node youtrack-cli.js <command> --param1 value1 --param2 value2
```

All output is JSON. Errors are printed to stderr with a non-zero exit code.

## Examples

```bash
# Create an issue
youtrack-cli add-issue \
  --projectShortName MYPROJ \
  --summary "Login button unresponsive on Safari" \
  --description "Reproducible on Safari 17, works fine on Chrome."

# Search issues
youtrack-cli get-issues --query "assignee: me State: Open"

# Fetch a specific issue
youtrack-cli get-issue --issueId MYPROJ-42

# Comment on an issue
youtrack-cli add-comment --issueId MYPROJ-42 --text "Reproducing locally now."

# Change issue state
youtrack-cli apply-command --query "State In Progress" --issueIds "MYPROJ-42"

# Load description from a file instead of typing it inline
youtrack-cli add-issue \
  --projectShortName MYPROJ \
  --summary "RFC: Auth redesign" \
  --descriptionFile ./rfc.md

# Update role permissions
youtrack-cli update-role \
  --roleId 17-6 \
  --permissions "jetbrains.jetpass.organization-read,jetbrains.jetpass.organization-delete"
```

Run `youtrack-cli --help` to see every available command with its parameters.

## Available commands

| Area | Commands |
|---|---|
| Issues | Create, search, view, update, delete; manage custom fields and visibility |
| Comments | Add, edit, delete comments on issues and articles; control visibility |
| Issue links | Link issues with typed relationships (depends on, duplicates, etc.) |
| Articles | Full CRUD for knowledge base articles; parent/child structure; visibility |
| Projects | Create and configure projects; manage teams, time tracking, workflows |
| Users | Create users; manage bans and profile settings (locale, timezone, notifications) |
| Groups & roles | Create, list, delete groups and roles; manage permissions; role assignments |
| Agile boards & sprints | Create boards and sprints |
| Dashboards | Create and manage dashboards |
| Work items | Log time on issues |
| Tags | Tag issues and articles |
| Workflows | View, create, and manage automation workflows |
| Global settings | Email, feature flags, license, AI features, banners |
| Custom fields | Manage field prototypes and bundle values |
| VCS & mailbox | Configure VCS integrations and mailbox rules |
| Raw API | `raw-get`, `raw-post`, `raw-delete` for any endpoint not covered above |

## License

Apache-2.0
