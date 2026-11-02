[//]: # (title: Planning Tools)

<show-structure for="chapter" depth="2"/>

YouTrack CLI can work with YouTrack's planning surfaces: agile boards, sprints, Gantt charts, and dashboards. This surface is thinner than the issue commands — some entities support full create, update, and delete, others are read-mostly, and a few can only be created in the web interface. Each section states what the CLI can and can't do. These entities are identified by an opaque `id` that YouTrack returns, for example `126-0`; run the matching `list` command to look one up.

## Working with agile boards {id="boards"}

An agile board organizes issues from one or more projects into a sprint-based view. The `board` command supports full create, update, and delete.

### Listing and viewing boards

List every board you can see, then view one by its `id`:

```Shell
youtrack board list
youtrack board view 126-0
```

### Creating a board

Create a board with `youtrack board create`. Both flags are required: name the board and attach it to a project by its short name.

```Shell
youtrack board create --name "Team Sprint Board" --project DEMO
```

> Use `youtrack project list` to look up a project's short name.
>
{style="tip"}

### Updating and deleting a board

Rename a board with `board update`, and remove it with `board delete`:

```Shell
youtrack board update 126-0 --name "Q3 Sprint Board"
youtrack board delete 126-0
```

`board delete` asks for confirmation. Add `--force` to skip the prompt in scripts.

> The CLI does not move issues between board columns. To change the field a board swimlane or column maps to — such as the State field — apply a command to the issue with `youtrack issue command`. The board reflects the new field value. See [Managing Issues](youtrack-cli-managing-issues.md).
>
{style="note"}

## Managing sprints {id="sprints"}

A sprint is a time-boxed iteration on a board. The CLI can list, view, create, update, and delete sprints.

### Listing and viewing sprints

Both commands require the board's `id` through the `--board` flag:

```Shell
youtrack board sprint list --board 126-0
youtrack board sprint view 127-3 --board 126-0
```

### Creating a sprint

Create a sprint on a board. The `--board` `id` and `--name` are required. The `--goal` flag sets a free-text goal, and `--start` and `--end` set the date range.

```Shell
youtrack board sprint create --board 126-0 --name "Sprint 14" --goal "Ship the search rewrite"
```

> The `--start` and `--end` flags take a Unix epoch timestamp in milliseconds, not a calendar date. Convert a date on the command line:
>
> ```Shell
> youtrack board sprint create --board 126-0 --name "Sprint 14" \
>   --start "$(date -j -f %Y-%m-%d 2026-08-01 +%s)000"
> ```
>
> The `-j -f` form is macOS `date`. On Linux, use `$(date -d 2026-08-01 +%s)000`.
>
{style="tip"}

### Updating and deleting a sprint

Change only the fields you pass. An empty goal clears the existing goal, and
`--archived=false` restores an archived sprint:

```Shell
youtrack board sprint update 127-3 --board 126-0 --name "Sprint 14B" --goal "Revised scope"
youtrack board sprint update 127-3 --board 126-0 --archived=true
youtrack board sprint delete 127-3 --board 126-0 --force
```

`sprint delete` asks for confirmation unless `--force` is supplied.

## Working with Gantt charts {id="gantt"}

A Gantt chart schedules issues across a timeline. The CLI manages existing charts: it can list, rename, retime, and delete them, and manage their member issues and one display setting. The CLI cannot create a Gantt chart or render its timeline — create charts in the YouTrack web interface, then manage them here.

### Listing, updating, and deleting charts

```Shell
youtrack gantt list
youtrack gantt update 96-2 --name "Release 2.0 Plan" --start-date 1754006400000
youtrack gantt delete 96-2
```

The `--start-date` flag on `gantt update` is a Unix epoch timestamp in milliseconds. Both `--name` and `--start-date` are optional; pass whichever you want to change.

`gantt delete` asks for confirmation. Add `--force` to skip the prompt in scripts.

### Managing chart members

A chart member is an issue placed on the timeline. Add an issue, list the current members, and set a member's start date:

```Shell
youtrack gantt member add 96-2 --issue DEMO-42
youtrack gantt member list 96-2
youtrack gantt member set-start 96-2 174-9 --start-date 1754006400000
```

The `set-start` command takes the chart `id` and the member's resource `id` — the `id` shown by `gantt member list`, not the issue ID. The `--start-date` flag is required and is a Unix epoch timestamp in milliseconds.

Toggle whether weekends appear on the chart with `settings update`:

```Shell
youtrack gantt settings update 96-2 --show-weekends
```

## Working with dashboards {id="dashboards"}

A dashboard is a named container for reporting widgets. The CLI manages the dashboard record — its name — only. It cannot read or edit the widgets on a dashboard.

List, create, rename, and delete dashboards:

```Shell
youtrack dashboard list
youtrack dashboard create --name "Team Overview"
youtrack dashboard update 152-5 --name "Team Metrics"
youtrack dashboard delete 152-5
```

The `--name` flag is required on both `create` and `update`. There is no `dashboard view` command; use `dashboard list` to find a dashboard's `id` and name.

`dashboard delete` asks for confirmation. Add `--force` to skip the prompt in scripts.

## Formatting output for scripts

Every command in this topic honors the global output flags: add `--json` for machine-readable output, `--plain` for tab-separated rows, or `--fields` to project specific fields. See [Scripting and Automation](youtrack-cli-scripting.md).

<seealso>
<category ref="user-guide">
<a href="youtrack-cli-managing-issues.md">Managing Issues</a>
<a href="youtrack-cli-managing-projects.md">Managing Projects</a>
<a href="youtrack-cli-scripting.md">Scripting and Automation</a>
</category>
<category ref="reference">
<a href="youtrack-cli.md">YouTrack CLI</a>
<a href="youtrack-cli-configuration.md">Configuration</a>
</category>
</seealso>
