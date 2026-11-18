[//]: # (title: Planning Tools)

<show-structure for="chapter" depth="2"/>

YouTrack CLI works with the planning tools of YouTrack: agile boards, sprints, Gantt charts, and dashboards. The CLI supports fewer operations here than for issues, and each section states what the CLI can and cannot do. Each of these entities has an opaque `id`, for example `126-0`. To find an `id`, run the matching `list` command.

## Manage agile boards {id="boards"}

An agile board organizes issues from one or more projects into a sprint-based view. The `board` command supports full create, update, and delete.

### List and view boards {id="board-list"}

List every board you can see, then view one by its `id`:

```Shell
youtrack board list
youtrack board view 126-0
```

### Create a board in a project {id="board-create"}

Create a board with `youtrack board create`. Both flags are required: name the board and attach it to a project by its short name.

```Shell
youtrack board create --name "Team Sprint Board" --project DEMO
```

> Use `youtrack project list` to look up a project's short name.
>
{style="tip"}

### Rename or delete a board {id="board-update-delete"}

Rename a board with `board update`. Remove a board with `board delete`, which asks for confirmation. Add `--force` to skip the prompt in scripts:

```Shell
youtrack board update 126-0 --name "Q3 Sprint Board"
youtrack board delete 126-0
```

> The CLI does not move issues between board columns. A board column or swimlane shows the value of a field, such as State. To move an issue, change that field with `youtrack issue command`. The board then shows the new field value. See [Managing Issues](youtrack-cli-managing-issues.md).
>
{style="note"}

## Manage sprints on a board {id="sprints"}

A sprint is a time-boxed iteration on a board. The CLI can list, view, create, update, and delete sprints.

### List and view sprints of a board {id="sprint-list"}

Both commands require the board's `id` through the `--board` flag:

```Shell
youtrack board sprint list --board 126-0
youtrack board sprint view 127-3 --board 126-0
```

### Create a sprint with a date range {id="sprint-create"}

Create a sprint on a board. The `--board` `id` and `--name` are required. `--goal` sets a free-text goal. `--start` and `--end` set the date range.

```Shell
youtrack board sprint create --board 126-0 --name "Sprint 14" --goal "Ship the search rewrite"
```

> The `--start` and `--end` flags take a Unix epoch timestamp in milliseconds, not a calendar date. Convert a date on the command line:
>
> ```Shell
> youtrack board sprint create --board 126-0 --name "Sprint 14" \
>   --start "$(date -j -f %Y-%m-%d 2026-08-01 +%s)000"
> ```
> {ignore-vars="true"}
>
> The `-j -f` form is for macOS `date`. On Linux, use `$(date -d 2026-08-01 +%s)000`.
>
{style="tip"}

### Update or delete a sprint {id="sprint-update-delete"}

`board sprint update` changes only the fields you pass. An empty goal clears the goal. `--archived=false` restores an archived sprint. `sprint delete` asks for confirmation unless you pass `--force`:

```Shell
youtrack board sprint update 127-3 --board 126-0 --name "Sprint 14B" --goal "Revised scope"
youtrack board sprint update 127-3 --board 126-0 --archived=true
youtrack board sprint delete 127-3 --board 126-0 --force
```

## Manage existing Gantt charts {id="gantt"}

A Gantt chart schedules issues on a timeline. The CLI manages charts that exist: it lists, renames, retimes, and deletes them, and it manages their member issues and one display setting. The CLI cannot create a Gantt chart or show its timeline. Create charts in the YouTrack web interface.

### List, update, and delete charts {id="gantt-list"}

```Shell
youtrack gantt list
youtrack gantt update 96-2 --name "Release 2.0 Plan" --start-date 1754006400000
youtrack gantt delete 96-2
```

The `--start-date` flag of `gantt update` takes a Unix epoch timestamp in milliseconds. `--name` and `--start-date` are optional. Pass the flags you want to change.

`gantt delete` asks for confirmation. Add `--force` to skip the prompt in scripts.

### Manage chart members {id="gantt-members"}

A chart member is an issue placed on the timeline. Add an issue, list the current members, and set a member's start date:

```Shell
youtrack gantt member add 96-2 --issue DEMO-42
youtrack gantt member list 96-2
youtrack gantt member set-start 96-2 174-9 --start-date 1754006400000
```

`set-start` takes the chart `id` and the resource `id` of the member. The resource `id` is the `id` that `gantt member list` shows, not the issue ID. The `--start-date` flag is required. It takes a Unix epoch timestamp in milliseconds.

### Show or hide weekends {id="gantt-settings"}

`gantt settings update` always sets the weekend display. If you want to show weekends, pass `--show-weekends`. If you want to hide them, pass `--show-weekends=false`. A call without the flag also hides weekends:

```Shell
youtrack gantt settings update 96-2 --show-weekends
youtrack gantt settings update 96-2 --show-weekends=false
```

## Manage dashboard names only {id="dashboards"}

A dashboard is a named container for report widgets. The CLI manages only the dashboard record, that is, its name. The CLI cannot read or edit the widgets on a dashboard.

List, create, rename, and delete dashboards:

```Shell
youtrack dashboard list
youtrack dashboard create --name "Team Overview"
youtrack dashboard update 152-5 --name "Team Metrics"
youtrack dashboard delete 152-5
```

The `--name` flag is required for `create` and `update`. The CLI has no `dashboard view` command. Use `dashboard list` to find the `id` and name of a dashboard.

`dashboard delete` asks for confirmation. Add `--force` to skip the prompt in scripts.

## All planning commands support script output {id="output"}

Every command in this topic supports the global output flags: `--json` for machine-readable output, `--plain` for tab-separated rows, and `--fields` to select fields. See [Scripting and Automation](youtrack-cli-scripting.md).

<seealso>
<category ref="04">
<a href="youtrack-cli-managing-issues.md">Managing Issues</a>
<a href="youtrack-cli-managing-projects.md">Managing Projects</a>
<a href="youtrack-cli-scripting.md">Scripting and Automation</a>
</category>
<category ref="06">
<a href="youtrack-cli.md">YouTrack CLI</a>
<a href="youtrack-cli-configuration.md">Configuration</a>
</category>
</seealso>
