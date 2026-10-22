[//]: # (title: YouTrack CLI)
<show-structure for="chapter" depth="2"/>

YouTrack CLI is a command-line interface for YouTrack. It manages issues, projects, articles, and more by talking to the YouTrack REST API, so any task you perform in the web interface you can also script from a terminal.

The CLI is built for developers and for AI coding agents such as Claude Code and Cursor. The binary is `youtrack`. Most commands take an issue ID in the form `PROJECT-123`, for example `DEMO-42`.

## Quick example {id="quick-example"}

Set your instance URL and permanent token, then work with issues. List the issues assigned to you, create one, and view it:

```Shell
export YOUTRACK_BASE_URL=https://example.youtrack.cloud
export YOUTRACK_TOKEN=perm:...
youtrack issue list --query "assignee: me #Unresolved"
youtrack issue create --project DEMO --summary "Login button misaligned"
youtrack issue view DEMO-42
```

YouTrack returns the new issue ID when you create an issue, for example `DEMO-42`.

> To connect the CLI to your instance, see [Authentication](youtrack-cli-authentication.md). To install the binary and run your first command, see [Getting Started](youtrack-cli-get-started.md).
>
{style="note"}

## Command groups {id="command-groups"}

Run `youtrack --help` to see every command group. The groups below map to the areas of YouTrack you can manage.

### Core work items

Everyday work with issues and the entities around them.

<deflist type="medium">
<def title="issue">

Create, search, update, comment on, link, and tag issues. See [Managing Issues](youtrack-cli-managing-issues.md).

<code-block lang="Shell">youtrack issue list --query "project: DEMO"</code-block>
</def>
<def title="project">

Create and configure projects. See [Managing Projects](youtrack-cli-managing-projects.md).
</def>
<def title="article">

Manage Knowledge Base articles. See [Managing Articles](youtrack-cli-managing-articles.md).
</def>
<def title="board">

Manage agile boards and sprints. See [Planning Tools](youtrack-cli-planning-tools.md).
</def>
<def title="tag">

Manage tags used to categorize issues.
</def>
<def title="search">

Manage saved searches.
</def>
</deflist>

### Administration

Manage the people and structures behind your instance. See [Managing Users and Access](youtrack-cli-managing-users-and-access.md) and [Administration](youtrack-cli-administration.md).

<table>
<tr><td>

`user`
</td><td>

Manage user accounts, identified by login.
</td></tr>
<tr><td>

`group`
</td><td>

Manage user groups.
</td></tr>
<tr><td>

`role`
</td><td>

Manage roles and permissions.
</td></tr>
<tr><td>

`org`
</td><td>

Manage organizations.
</td></tr>
<tr><td>

`field`
</td><td>

Manage custom fields.
</td></tr>
<tr><td>

`workflow`
</td><td>

Manage workflows.
</td></tr>
<tr><td>

`settings`
</td><td>

Manage instance settings.
</td></tr>
</table>

### Integrations

Connect YouTrack to external systems.

- `app` — manage apps.

### Planning

Tools for tracking and visualizing work. See [Planning Tools](youtrack-cli-planning-tools.md).

- `dashboard` — manage dashboards.
- `gantt` — manage Gantt charts.
- `whiteboard` — manage whiteboards.

### Access

Authenticate and switch between configured instances. See [Authentication](youtrack-cli-authentication.md) and [Configuration](youtrack-cli-configuration.md).

- `auth` — authenticate with an instance, check status, and manage credentials.
- `instance` — list, switch between, and delete configured YouTrack instances; `instance version` shows the active instance's YouTrack version.

### Utilities

- `raw` — make an authenticated request to any YouTrack REST API endpoint. See [Scripting and Automation](youtrack-cli-scripting.md).
- `skill` — install AI coding agent skills that teach agents to drive the CLI. See [AI Agent Integration](youtrack-cli-ai-agent-integration.md).

## Output for scripts {id="output"}

Every command prints a human-readable table by default and switches to JSON automatically when output is not a terminal. Pass `--json` to force JSON, or `--plain` for tab-separated output suited to `grep` and `awk`. For details, see [Scripting and Automation](youtrack-cli-scripting.md).

## Next steps {id="next-steps"}

- Install the CLI and run your first command in [Getting Started](youtrack-cli-get-started.md).
- Connect to your instance in [Authentication](youtrack-cli-authentication.md).
- Learn the day-to-day workflow in [Managing Issues](youtrack-cli-managing-issues.md).

<seealso>
<category ref="user-guide">
<a href="youtrack-cli-get-started.md">Getting Started</a>
<a href="youtrack-cli-authentication.md">Authentication</a>
<a href="youtrack-cli-managing-issues.md">Managing Issues</a>
<a href="youtrack-cli-ai-agent-integration.md">AI Agent Integration</a>
</category>
<category ref="reference">
<a href="youtrack-cli-configuration.md">Configuration</a>
<a href="youtrack-cli-scripting.md">Scripting and Automation</a>
</category>
</seealso>
