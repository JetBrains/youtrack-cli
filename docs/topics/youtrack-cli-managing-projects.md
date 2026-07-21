[//]: # (title: Managing Projects)
<show-structure for="chapter" depth="2"/>

A project groups issues under a short name, such as `DEMO`. YouTrack CLI lets you create projects, inspect them, manage their team, and configure their custom fields from the command line. Most project commands identify a project by its short name.

## Listing projects

List every project you can access with `youtrack project list`:

```Shell
youtrack project list
```

Each row shows the project short name and full name. Use the short name to address the project in other commands.

## Viewing a project

Show the details of a single project with `youtrack project view` and its short name:

```Shell
youtrack project view DEMO
```

> Add `--json` to get the full project record, including the opaque `id` that some commands require.
>
{style="tip"}

## Creating a project {id="create"}

Create a project with `youtrack project create`. Provide a short name and a full name:

```Shell
youtrack project create --shortName DEMO --name "Demo Project"
```

YouTrack returns the new project. The short name becomes the prefix of every issue ID in the project, for example `DEMO-42`.

Every project must have a leader. The YouTrack API rejects a project that has no leader. If you omit `--leader`, YouTrack CLI sets the leader to the current authenticated user. To assign a different leader, pass a login:

```Shell
youtrack project create --shortName DEMO --name "Demo Project" --leader jane.doe
```

> Run `youtrack user current` to see the login of the account YouTrack CLI is authenticated as.
>
{style="tip"}

To start from a built-in template, add the `--template` flag:

```Shell
youtrack project create --shortName DEMO --name "Demo Project" --template scrum
```

### project create flags

<table>
<tr><td>Flag</td><td>Description</td></tr>
<tr><td>`--shortName`

</td><td>Project short name, used as the issue ID prefix. Required.

</td></tr>
<tr><td>`--name`

</td><td>Project full name. Required.

</td></tr>
<tr><td>`--leader`

</td><td>Login of the project leader. Defaults to the current authenticated user.

</td></tr>
<tr><td>`--template`

</td><td>Built-in template: `helpdesk`, `taskManagement`, `scrum`, `kanban`, `projectManagement`, `demo`, or `marketing`. Omit for an empty project.

</td></tr>
</table>

## Archiving a project

Archive a project to retire it from active use without deleting its data:

```Shell
youtrack project archive DEMO
```

`project archive` asks for confirmation. Add `--force` to skip the prompt in scripts.

## Deleting a project

Delete a project with `youtrack project delete`. This is irreversible.

```Shell
youtrack project delete DEMO
```

YouTrack CLI asks for confirmation first. To skip the prompt in a script, add `--force`:

```Shell
youtrack project delete DEMO --force
```

> Deleting a project removes every issue it contains. Archive the project instead if you only need to retire it.
>
{style="warning"}

## Managing the project team {id="team"}

Add users and groups to a project team so they can work on its issues.

Both team commands take opaque IDs, not logins or group names. A user ID looks like `2-9`; a group ID looks like `4-1`. Look up the ID first, then pass it.

To add a user, read the user's `id`, then add it:

```Shell
youtrack user view jane.doe --json
youtrack project team add-user DEMO 2-9
```

To add a group, read the group's `id`, then add it:

```Shell
youtrack group view Developers --json
youtrack project team add-group DEMO 4-1
```

> Extract the ID in one step with a JSON processor, for example `youtrack user view jane.doe --json | jq -r .id`.
>
{style="tip"}

## Managing custom fields

List the custom fields attached to a project:

```Shell
youtrack project field list DEMO
```

Attach an existing custom field to a project with `youtrack project field add`. The `--field` flag takes the custom field's ID, which you look up at the instance level:

```Shell
youtrack field list --json
youtrack project field add DEMO --field 58-3
```

> Custom fields are defined once for the whole instance, then attached to individual projects. See <a href="youtrack-cli-administration.md">Administration</a> for managing the instance-level field catalog.
>
{style="note"}

## Configuring other project settings

Several subcommands adjust individual project settings. Each takes the project short name.

<table>
<tr><td>Command</td><td>Description</td></tr>
<tr><td>`youtrack project timetrack set DEMO`

</td><td>Enable time tracking for the project. Add `--enabled=false` to disable it.

</td></tr>
<tr><td>`youtrack project workflow add DEMO --workflow <id>`

</td><td>Attach a workflow to the project by its ID.

</td></tr>
<tr><td>`youtrack project template list`

</td><td>List the projects that are marked as templates.

</td></tr>
<tr><td>`youtrack project template convert DEMO`

</td><td>Convert an existing project into a template.

</td></tr>
</table>

<seealso>
<category ref="user-guide">
<a href="youtrack-cli-managing-issues.md">Managing Issues</a>
<a href="youtrack-cli-managing-users-and-access.md">Managing Users and Access</a>
<a href="youtrack-cli-administration.md">Administration</a>
</category>
<category ref="reference">
<a href="youtrack-cli-get-started.md">Getting Started</a>
<a href="youtrack-cli-scripting.md">Scripting and Automation</a>
</category>
</seealso>
