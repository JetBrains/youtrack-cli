[//]: # (title: Managing Issues)
<show-structure for="chapter" depth="2"/>

Issues are the core work items in YouTrack. This page shows you how to search, view, create, update, and enrich issues from the command line with the `youtrack issue` commands.

Every issue has an issue ID in the form `PROJECT-123`, for example `DEMO-42`. Most commands take that ID as their first argument. To find one, run `youtrack issue list`. The examples below assume you have set `YOUTRACK_BASE_URL` and `YOUTRACK_TOKEN` (see [Authentication](youtrack-cli-authentication.md)).

## Listing and searching issues {id="list"}

List issues with `youtrack issue list`. The `--query` flag is required and takes a YouTrack search query, the same syntax you use in the web interface:

```Shell
youtrack issue list --query "assignee: me #Unresolved"
youtrack issue list --query "project: DEMO State: Open"
```

By default the command returns up to 100 issues. Use `--limit` to change that and `-f` to choose which fields to show:

```Shell
youtrack issue list --query "project: DEMO" --limit 20 -f idReadable,summary,project.shortName
```

Use `--offset` to skip issues when paging. Both `--limit` and `--offset` must be non-negative; `--limit 0` returns no issues.

Custom fields work as columns too, like the visible columns of the issue list in the web interface. Any `--fields` entry that is not an issue attribute (`idReadable`, `summary`, `reporter`, `created`, `project.shortName`, and so on) is read as the name of a custom field in the issue's project. Names are case-insensitive, and a name with spaces needs quoting:

```Shell
youtrack issue list --query "project: DEMO #Unresolved" --fields "idReadable,summary,Priority,State,Assignee,Fix versions"
```

```
ID       SUMMARY                         PRIORITY  STATE        ASSIGNEE  FIX VERSIONS
DEMO-42  Login button unresponsive       Critical  In Progress  jane.doe  2.1
DEMO-41  Crash when exporting to PDF     Major     Submitted
```

When a requested custom field exists in none of the listed issues' projects, the command prints a warning to stderr and leaves that column empty. With `--json`, the output is still the full issue objects, and `customFields` holds only the fields you asked for.

To count matches without listing them, use `youtrack issue count`:

```Shell
youtrack issue count --query "project: DEMO #Unresolved"
```

## Viewing an issue {id="view"}

View a single issue by its ID. Add `--comments N` to include the N most recent comments, and `-f` to select fields:

```Shell
youtrack issue view DEMO-42
youtrack issue view DEMO-42 --comments 5 -f idReadable,summary,reporter.login
```

`-f` adds fields to the default projection. To fetch only the fields you name, add `--only-fields`. To keep the default projection but leave out a long description, add `--no-description`:

```Shell
youtrack issue view DEMO-42 --only-fields -f idReadable,summary,reporter --json
youtrack issue view DEMO-42 --no-description --json
```

## Creating an issue {id="create"}

Create an issue with `youtrack issue create`. Specify the project by its short name and provide a summary:

```Shell
youtrack issue create --project DEMO --summary "Login button misaligned"
```

YouTrack returns the new issue ID, for example `DEMO-42`. Add a description inline or from a file, and set custom fields with a JSON object of field names to values:

```Shell
youtrack issue create --project DEMO --summary "Release notes" --description-file notes.md
youtrack issue create --project DEMO --summary "Crash on startup" --custom-fields '{"Type":"Bug","Priority":"Critical"}'
```

`--project` and `--summary` are required. The other flags are optional: `--description` and `--description-file` set the body (mutually exclusive; `--description-file -` reads standard input); `--custom-fields` and `--custom-fields-file` take a JSON object of field values; `--parent` nests the issue under another issue ID; and `--asUser` creates it as another login.

Custom fields must be one complete JSON object with string values. Malformed JSON, trailing content, and non-string values (including `null`) are rejected before an issue is created.

> Use `youtrack project list` to look up a project's short name.
>
{style="tip"}

## Updating an issue {id="update"}

Update the summary or description of an existing issue:

```Shell
youtrack issue update DEMO-42 --summary "Login button overlaps header"
youtrack issue update DEMO-42 --description "Reproduced on iOS Safari."
youtrack issue update DEMO-42 --description ""  # clear the description
generate-notes | youtrack issue update DEMO-42 --description-file -
```

Only fields whose flags are passed are changed. An explicit empty `--description` clears the description; omitting the flag preserves it. `--description-file <path>` reads the description from a file, and `--description-file -` from standard input.

To change fields such as State, Priority, or Assignee, apply a command (see [Applying a command](#command)) or set a field value directly (see [Setting custom fields](#fields)).

## Assigning and deleting issues {id="assign-delete"}

Assign an issue to a user by login, or clear the assignee:

```Shell
youtrack issue assign DEMO-42 jane.doe
youtrack issue unassign DEMO-42
```

Delete an issue with `youtrack issue delete`. The command prompts for confirmation unless you pass `--force`:

```Shell
youtrack issue delete DEMO-42 --force
```

## Working with comments {id="comments"}

Manage comments through `youtrack issue comment`. The comment ID (for example `4-17`) comes from the ID column of `issue comment list`:

```Shell
youtrack issue comment list DEMO-42
youtrack issue comment create DEMO-42 --text "Confirmed on the latest build."
youtrack issue comment update DEMO-42 4-17 --text "Confirmed on build 2024.3."
youtrack issue comment delete DEMO-42 4-17
youtrack issue comment pin DEMO-42 4-17
youtrack issue comment pin DEMO-42 4-17 --unpin
youtrack issue comment create DEMO-42 --text "Triaged." --asUser jane.doe
youtrack issue comment create DEMO-42 --text-file report.md
git log -1 --format=%B | youtrack issue comment update DEMO-42 4-17 --text-file -
```

`comment create` and `comment update` take the text either inline with `--text` or from a file with `--text-file`; `--text-file -` reads standard input, which keeps long markdown and shell-sensitive characters intact.

## Tagging issues {id="tags"}

Attach, list, and inspect tags with `youtrack issue tag`. `add` and `remove` accept a tag ID or the tag's exact name; `view` takes an ID:

```Shell
youtrack issue tag list DEMO-42
youtrack issue tag add DEMO-42 6-3
youtrack issue tag add DEMO-42 "release blocker"
youtrack issue tag view DEMO-42 6-3
youtrack issue tag remove DEMO-42 "release blocker"
```

A name must match one tag exactly, including case. `add` looks among the tags visible to you, and `remove` looks among the tags on the issue. When no tag has that name, or several tags do (tags owned by different users can share a name), the command fails and lists the candidates' IDs. `add` never creates a tag; create one with `youtrack tag create` first. An argument shaped like an ID, such as `6-3`, is always treated as an ID.

Tags share one namespace across the instance, so filter the list instead of reading all of it. `--query` keeps tags with a word in their name that starts with the text, and `--limit` (default 100) caps the result:

```Shell
youtrack tag list --query release --limit 20
```

## Linking issues {id="links"}

Connect issues with `youtrack issue link`. List the available link types first, then add a link by target issue ID and link type ID:

```Shell
youtrack issue link type list
youtrack issue link add DEMO-42 --target DEMO-7 --type 100-0
```

List the links on an issue and remove one by its opaque link ID, shown in the ID column of `issue link list`:

```Shell
youtrack issue link list DEMO-42
youtrack issue link remove DEMO-42 100-1
```

`issue link remove` prompts for confirmation unless you pass `--force`.

## Attaching files {id="attachments"}

Add, list, and delete file attachments with `youtrack issue attachment`:

```Shell
youtrack issue attachment add DEMO-42 --file screenshot.png
youtrack issue attachment list DEMO-42
youtrack issue attachment delete DEMO-42 5-12
```

`issue attachment delete` prompts for confirmation unless you pass `--force`.

## Applying a command {id="command"}

A command is a space-delimited keyword-value instruction, the same one you type in YouTrack's Apply Command dialog. Apply one to one or more issues with `youtrack issue command`. Separate multiple issue IDs with commas and no spaces:

```Shell
youtrack issue command DEMO-42 "State Fixed"
youtrack issue command DEMO-1,DEMO-2 "assignee jane.doe Priority Critical"
```

A single command can set several fields at once, add tags, or change the State.

YouTrack creates a tag that does not exist yet when a command names it. Pass `--no-new-tags` to refuse such a command before anything is applied:

```Shell
youtrack issue command DEMO-42 "tag regression" --no-new-tags
```

The check reads the command preview from `youtrack issue command suggest`, which follows your YouTrack language; non-English or unverifiable previews are rejected without applying the command. The preview is a best-effort check, not an atomic server-side restriction: tags or settings can change before execution. For exact existing-tag association, use `issue tag add`.

> When a command sets the assignee or reporter through `issue command`, YouTrack CLI automatically wraps a login containing a dot, hyphen, or `@` in braces for YouTrack's command parser, so `assignee jane.doe` becomes `assignee {jane.doe}`. Write the login as-is.
>
{style="note"}

## Setting custom fields {id="fields"}

Read or set a single custom field value with `youtrack issue field`:

```Shell
youtrack issue field view DEMO-42 State
youtrack issue field set DEMO-42 State Fixed
youtrack issue field set DEMO-42 Estimation 2h --type period
```

For non-enum fields, pass `--type` (`period`, `text`, `user`, `int`, or `string`) so the CLI interprets the value correctly. When omitted, the value is treated as an enum.

Repeat `--value` to set several values of a multi-value field; the list replaces the current values. `--clear` empties a field. Values are sent as-is, so names with spaces or dots need no escaping:

```Shell
youtrack issue field set DEMO-42 Subsystem --value "Platform. Auth & Licensing" --value UI
youtrack issue field set DEMO-42 Assignee --clear
```

## Watching and voting {id="watch-vote"}

Subscribe to notifications or register a vote on an issue:

```Shell
youtrack issue watch DEMO-42
youtrack issue unwatch DEMO-42
youtrack issue vote DEMO-42
youtrack issue unvote DEMO-42
```

## Working with drafts {id="drafts"}

A draft is a template-prefilled issue stub you can build up before submitting it as a real issue. Create a draft from a project's issue template, update its fields, then submit it:

```Shell
youtrack issue draft create --project DEMO --summary "Sprint retro action items"
youtrack issue draft update 3-45 --custom-fields '{"Type":"Task","Priority":"Normal"}'
youtrack issue draft submit 3-45
youtrack issue draft delete 3-45
```

`issue draft create` returns the draft ID (for example `3-45`). Pass it to `update`, `submit`, and `delete`. Both `create` and `update` accept `--description` and `--description-file` (`-` reads standard input); `update` additionally accepts `--custom-fields` and `--custom-fields-file`. `issue draft delete` discards a draft you no longer need and prompts for confirmation unless you pass `--force`.

## Viewing sprints {id="sprints"}

List the sprints that contain an issue:

```Shell
youtrack issue sprint list DEMO-42
```

To work with the boards and sprints themselves, see [Planning Tools](youtrack-cli-planning-tools.md).

## Logging work {id="work-items"}

Track time spent on an issue with `youtrack issue workitem`:

```Shell
youtrack issue workitem create DEMO-42 --minutes 90 --text "Implemented the fix"
youtrack issue workitem list DEMO-42
youtrack issue workitem view DEMO-42 <workItemId>
youtrack issue workitem update DEMO-42 <workItemId> --minutes 120
youtrack issue workitem delete DEMO-42 <workItemId>
```

`--minutes` is required when creating a work item (for example, `90` means 1 h 30 m). Optionally pass `--text` for a description and `--type` with a work type ID. `workitem delete` prompts for confirmation unless you pass `--force`.

## Linking commits {id="vcs-changes"}

List the commits and pull requests linked to an issue, and link a commit yourself, with `youtrack issue vcs`:

```Shell
youtrack issue vcs list DEMO-42
youtrack issue vcs view DEMO-42 <changeId>
youtrack issue vcs attach DEMO-42 1a2b3c4d5e6f7081
youtrack issue vcs detach DEMO-42 1a2b3c4d5e6f7081
```

`attach` takes a commit hash, abbreviated to 12 characters or more. YouTrack resolves the hash against the VCS integrations configured for the issue's project, so the commit must be one an integration can see — otherwise the call reports that no change with that version was found. A Generic VCS integration accepts any hash and takes the commit message from `--message`.

`detach` (aliases `delete`, `remove`) unlinks a change and accepts either the change ID or the commit hash. YouTrack offers two mechanisms and accepts only one of them per change, so the command picks it: a commit you attached by hand is deleted outright, while one that a VCS integration attached is detached instead, because the server refuses to delete those. YouTrack keeps a detached record only where it needs one to stop the integration re-attaching the commit — that is, when the commit message names the issue. Otherwise the change goes for good, and `attach` restores a record that survived.

Writing to VCS changes requires the *Update Issue Private Fields* permission. Pass `--mute-notifications` to suppress notifications about the change, which additionally needs *Apply Commands Silently*.

## Restricting visibility {id="visibility"}

Limit who can see an issue by passing permitted group or user IDs to `youtrack issue visibility`:

```Shell
youtrack issue visibility DEMO-42 --groups 4-1
youtrack issue visibility DEMO-42 --users 2-9
```

> `--groups` and `--users` expect opaque IDs, not group names or logins. Retrieve them first with `youtrack group view <name>` or `youtrack user view <login>`.
>
{style="warning"}

<seealso>
<category ref="user-guide">
<a href="youtrack-cli-get-started.md">Getting Started</a>
<a href="youtrack-cli-managing-projects.md">Managing Projects</a>
<a href="youtrack-cli-planning-tools.md">Planning Tools</a>
<a href="youtrack-cli-scripting.md">Scripting and Automation</a>
</category>
<category ref="reference">
<a href="youtrack-cli-authentication.md">Authentication</a>
<a href="youtrack-cli-configuration.md">Configuration</a>
</category>
</seealso>
