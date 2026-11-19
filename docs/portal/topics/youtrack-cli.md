# YouTrack CLI
<show-structure for="chapter" depth="2"/>

YouTrack CLI is a command-line interface (CLI) for YouTrack. It calls the YouTrack REST API, the HTTP interface of YouTrack, so you can script from a terminal the tasks that you do in the web interface. To start, install the binary and sign in with `youtrack auth login`.

The CLI is built for developers and for artificial intelligence (AI) coding agents, such as Claude Code and Cursor. The binary is `youtrack`. Most commands take an issue ID in the form `PROJECT-123`, for example `DEMO-42`.

## Sign in once, then work with issues

Sign in through the browser, then list the issues assigned to you, create one, and view it:

```Shell
youtrack auth login https://example.youtrack.cloud
youtrack issue list --query "assignee: me #Unresolved"
youtrack issue create --project DEMO --summary "Login button misaligned"
youtrack issue view DEMO-42
```

`auth login` saves an OAuth session, a sign-in that the CLI renews automatically. YouTrack returns the new issue ID when you create an issue, for example `DEMO-42`. For scripts and continuous integration (CI), use the `YOUTRACK_BASE_URL` and `YOUTRACK_TOKEN` environment variables instead. See [Scripting and Automation](youtrack-cli-scripting.md).

<note>

To install the binary and run your first command, see [Getting Started](youtrack-cli-get-started.md). To choose a sign-in method, see [Authentication](youtrack-cli-authentication.md).

</note>

## Command groups match the areas of YouTrack

Run `youtrack --help` to see every command group. The groups below map to the areas of YouTrack you can manage.

### Core work items

Everyday work with issues and the entities around them.

<table style="none">
<tr><td>

`issue`
</td><td>

Create, search, update, comment on, link, and tag issues. See [Managing Issues](youtrack-cli-managing-issues.md).

```Shell
youtrack issue list --query "project: DEMO"
```
</td></tr>
<tr><td>

`project`
</td><td>

Create and configure projects. See [Managing Projects](youtrack-cli-managing-projects.md).
</td></tr>
<tr><td>

`article`
</td><td>

Manage Knowledge Base articles. See [Managing Articles](youtrack-cli-managing-articles.md).
</td></tr>
<tr><td>

`board`
</td><td>

Manage agile boards and sprints. See [Planning Tools](youtrack-cli-planning-tools.md).
</td></tr>
<tr><td>

`tag`
</td><td>

Manage tags used to categorize issues.
</td></tr>
<tr><td>

`search`
</td><td>

Manage saved searches.
</td></tr>
</table>

### Administration

Manage the people and structures behind your instance. See [Managing Users and Access](youtrack-cli-managing-users-and-access.md) and [Administration](youtrack-cli-administration.md).

<table style="none">
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

### Access

Authenticate and switch between configured instances. See [Authentication](youtrack-cli-authentication.md) and [Configuration](youtrack-cli-configuration.md).

- `auth` — authenticate with an instance, check status, and manage credentials.
- `instance` — list, switch between, and delete configured YouTrack instances; `instance version` shows the active instance's YouTrack version.

### Utilities

- `raw` — make an authenticated request to any YouTrack REST API endpoint. See [Scripting and Automation](youtrack-cli-scripting.md).
- `skill` — install AI coding agent skills that teach agents to drive the CLI. See [AI Agent Integration](youtrack-cli-ai-agent-integration.md).
- `update` — check for a newer YouTrack CLI release and upgrade through the same install method. See [Getting Started](youtrack-cli-get-started.md#upgrade-with-youtrack-update).

## Output switches to JSON in scripts

Every command prints a human-readable table by default and switches to JSON (JavaScript Object Notation) automatically when output is not a terminal. Pass `--json` to force JSON, or `--plain` for tab-separated output suited to `grep` and `awk`. For details, see [Scripting and Automation](youtrack-cli-scripting.md).

## Next steps

- Install the CLI and run your first command in [Getting Started](youtrack-cli-get-started.md).
- Connect to your instance in [Authentication](youtrack-cli-authentication.md).
- Learn the day-to-day workflow in [Managing Issues](youtrack-cli-managing-issues.md).

<seealso>
<category ref="04">
<a href="youtrack-cli-get-started.md">Getting Started</a>
<a href="youtrack-cli-authentication.md">Authentication</a>
<a href="youtrack-cli-managing-issues.md">Managing Issues</a>
<a href="youtrack-cli-ai-agent-integration.md">AI Agent Integration</a>
</category>
<category ref="06">
<a href="youtrack-cli-configuration.md">Configuration</a>
<a href="youtrack-cli-scripting.md">Scripting and Automation</a>
<a href="youtrack-cli-security.md">Security</a>
</category>
</seealso>
