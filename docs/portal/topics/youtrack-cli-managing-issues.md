# Managing Issues
<show-structure for="chapter" depth="2"/>

Issues are the core work items in YouTrack. This page shows how to search, view, create, update, and enrich issues with the `youtrack issue` commands. Sign in first (see [Authentication](youtrack-cli-authentication.md)).

Every issue has an issue ID in the form `PROJECT-123`, for example `DEMO-42`. Most commands take that ID as their first argument. To find an ID, run `youtrack issue list`.

## Find issues with a search query

List issues with `youtrack issue list`. The `--query` flag is required. It takes a YouTrack search query, in the same syntax as the web interface:

```Shell
youtrack issue list --query "assignee: me #Unresolved"
youtrack issue list --query "project: DEMO State: Open"
```

The command returns up to 100 issues by default. Use `--limit` to change the number, and `--fields` to choose the fields to show:

```Shell
youtrack issue list --query "project: DEMO" --limit 20 --fields idReadable,summary,project.shortName
```

Use `--offset` to skip issues when you page through results. `--limit` and `--offset` must not be negative. `--limit 0` returns no issues.

Custom fields also work as columns, like the visible columns of the issue list in the web interface. A `--fields` entry that is not an issue attribute (`idReadable`, `summary`, `reporter`, `created`, `project.shortName`, and so on) is the name of a custom field in the issue's project. Names are case-insensitive. Put a name with spaces in quotes:

```Shell
youtrack issue list --query "project: DEMO #Unresolved" --fields "idReadable,summary,Priority,State,Assignee,Fix versions"
```

```
ID       SUMMARY                         PRIORITY  STATE        ASSIGNEE  FIX VERSIONS
DEMO-42  Login button unresponsive       Critical  In Progress  jane.doe  2.1
DEMO-41  Crash when exporting to PDF     Major     Submitted
```

If no project of the listed issues has a requested custom field, the command prints a warning to stderr and leaves that column empty. With `--json`, the output contains the full issue objects, and `customFields` holds only the fields you asked for.

To count matches without listing them, use `youtrack issue count`:

```Shell
youtrack issue count --query "project: DEMO #Unresolved"
```

## View one issue by its ID

View a single issue by its ID. Add `--comments N` to include the N most recent comments, and `--fields` to select fields:

```Shell
youtrack issue view DEMO-42
youtrack issue view DEMO-42 --comments 5 --fields idReadable,summary,reporter.login
```

`--fields` adds fields to the default projection. To fetch only the fields you name, add `--only-fields`. To keep the default projection without a long description, add `--no-description`:

```Shell
youtrack issue view DEMO-42 --only-fields --fields idReadable,summary,reporter --json
youtrack issue view DEMO-42 --no-description --json
```

To see the change history of an issue, list its activity items:

```Shell
youtrack issue activity list DEMO-42
```

## Create an issue in one request

Create an issue with `youtrack issue create`. Specify the project by its short name and give a summary:

```Shell
youtrack issue create --project DEMO --summary "Login button misaligned"
```

> Use `youtrack project list` to look up a project's short name.

YouTrack returns the new issue ID, for example `DEMO-42`. Add a description inline or from a file. Set custom fields with a JSON object of field names and values:

```Shell
youtrack issue create --project DEMO --summary "Release notes" --description-file notes.md
youtrack issue create --project DEMO --summary "Crash on startup" --custom-fields '{"Type":"Bug","Priority":"Critical"}'
youtrack issue create --project DEMO --summary "Prepare release" --type Task --tag release --custom-fields '{"Priority":"Major","Assignee":"me"}'
```

`--project` and `--summary` are required. All other flags are optional:

| Flag | Effect |
|---|---|
| `--description`, `--description-file` | Set the body. The two flags are mutually exclusive. `--description-file -` reads standard input. |
| `--custom-fields`, `--custom-fields-file` | Set custom fields from a JSON object. |
| `--type` | Set the issue Type, for example `Task`. Without the flag, the project default applies. |
| `--tag` | Add an existing tag by name or ID. Repeat the flag for more tags. |
| `--visibility` | Limit the issue to a group name or user login. Repeat the flag for more audiences. Prefix the value with `group:` or `user:` if a group and a user share the name. |
| `--public` | Make the issue visible to everyone who can see the project. You cannot use `--public` together with `--visibility`. |
| `--parent` | Make the issue a subtask of another issue ID. |
| `--asUser` | Create the issue as another user, by login. |

The CLI sends the summary, description, custom fields, Type, tags, and visibility in the request that creates the issue. Thus workflows that require a field accept the issue. The CLI adds the parent link after the issue exists. If the parent link fails, the CLI deletes the new issue.

Custom fields must be one complete JSON object with string values. The CLI rejects malformed JSON, trailing content, and non-string values (including `null`) before it creates the issue.

## Update only the fields you pass

Update the summary or description of an issue:

```Shell
youtrack issue update DEMO-42 --summary "Login button overlaps header"
youtrack issue update DEMO-42 --description "Reproduced on iOS Safari."
youtrack issue update DEMO-42 --description ""  # clear the description
generate-notes | youtrack issue update DEMO-42 --description-file -
```

The command changes only the fields whose flags you pass. An empty `--description` clears the description. If you omit the flag, the description stays. `--description-file <path>` reads the description from a file, and `--description-file -` reads it from standard input.

To change fields such as State, Priority, or Assignee, apply a command (see [Apply a command](#apply-a-command-to-change-several-fields)) or set a field value (see [Set one custom field](#set-one-custom-field-directly)).

## Assign and delete issues

Assign an issue to a user by login, or clear the assignee:

```Shell
youtrack issue assign DEMO-42 jane.doe
youtrack issue unassign DEMO-42
```

<warning>

Deleting an issue is permanent. `youtrack issue delete` asks for confirmation. Pass `--force` to skip the prompt.

</warning>

```Shell
youtrack issue delete DEMO-42 --force
```

## Work with comments by comment ID

Manage comments with `youtrack issue comment`. The comment ID (for example `4-17`) is in the ID column of `issue comment list`:

```Shell
youtrack issue comment list DEMO-42
youtrack issue comment view DEMO-42 4-17
youtrack issue comment create DEMO-42 --text "Confirmed on the latest build."
youtrack issue comment update DEMO-42 4-17 --text "Confirmed on build 2024.3."
youtrack issue comment delete DEMO-42 4-17
youtrack issue comment pin DEMO-42 4-17
youtrack issue comment pin DEMO-42 4-17 --unpin
youtrack issue comment create DEMO-42 --text "Triaged." --asUser jane.doe
youtrack issue comment create DEMO-42 --text-file report.md
git log -1 --format=%B | youtrack issue comment update DEMO-42 4-17 --text-file -
```

`comment create` and `comment update` take the text inline with `--text`, or from a file with `--text-file`. `--text-file -` reads standard input. Standard input keeps long Markdown and shell-sensitive characters intact.

`issue comment list` leaves out deleted comments. Pass `--include-deleted` to list them too, marked as deleted. Comment JSON includes `deleted`, `pinned`, and `visibility` with its permitted groups and users:

```Shell
youtrack issue comment list DEMO-42 --include-deleted --fields id,deleted,pinned,text
```

### React to a comment

Add a reaction to an issue comment with `youtrack issue reaction add`. The reaction is the third positional argument. Remove a reaction by its reaction ID with `youtrack issue reaction delete`. The delete command asks for confirmation unless you pass `--force`:

```Shell
youtrack issue reaction add DEMO-42 4-17 👍
youtrack issue reaction delete DEMO-42 4-17 <reactionId> --force
```

The CLI has no command that lists the reactions on an issue comment. Thus the CLI cannot show you a reaction ID. Article comments use a different syntax: they take the reaction with `--reaction` and have a `reaction list` command (see [Managing Articles](youtrack-cli-managing-articles.md#react-to-a-comment)).

## Tag issues by tag name or ID

Attach, list, and inspect tags with `youtrack issue tag`. `add` and `remove` accept a tag ID or the exact tag name. `view` takes an ID:

```Shell
youtrack issue tag list DEMO-42
youtrack issue tag add DEMO-42 6-3
youtrack issue tag add DEMO-42 "release blocker"
youtrack issue tag view DEMO-42 6-3
youtrack issue tag remove DEMO-42 "release blocker"
```

A name must match one tag exactly, including case. `add` looks among the tags visible to you. `remove` looks among the tags on the issue. Tags owned by different users can share a name. If no tag or more than one tag has the name, the command fails and lists the IDs of the candidates. `add` never creates a tag. To create one, run `youtrack tag create` first. The CLI always reads an argument with the form of an ID, such as `6-3`, as an ID.

Tags share one namespace across the instance, so filter the list instead of reading all of it. `--query` keeps tags with a word in their name that starts with the text. `--limit` (default 100) caps the result:

```Shell
youtrack tag list --query release --limit 20
```

## Link issues through a link type

Connect issues with `youtrack issue link`. List the available link types first. Then add a link with the target issue ID and the link type ID:

```Shell
youtrack issue link type list
youtrack issue link add DEMO-42 --target DEMO-7 --type 167-0
```

A link type can be directed, for example "depends on / is required for". For a directed type, a plain type ID such as `167-0` creates an outward link: DEMO-42 depends on DEMO-7. To create an inward link, add the suffix `t` to the type ID (`167-0t`). The suffix `s` selects the outward direction explicitly.

To create a link type, give its name and both direction labels:

```Shell
youtrack issue link type create --name "Depends" --source-to-target "depends on" --target-to-source "is required for"
```

`issue link list` shows the links of an issue. Each row is a link slot: one link type in one direction. The ID column holds the slot ID, for example `167-0s`.

<warning>

`issue link remove` removes **all** issues that are linked through the slot, not one link. The command asks for confirmation unless you pass `--force`.

</warning>

```Shell
youtrack issue link list DEMO-42
youtrack issue link remove DEMO-42 167-0s
```

## Attach and download files

Add, list, and delete file attachments with `youtrack issue attachment`:

```Shell
youtrack issue attachment add DEMO-42 --file screenshot.png
youtrack issue attachment list DEMO-42
youtrack issue attachment delete DEMO-42 5-12
```

`issue attachment delete` asks for confirmation unless you pass `--force`.

To restrict the file when you upload it, pass `--visibility` with a group name or a user login. Repeat the flag for more audiences. Prefix the value with `group:` or `user:` if a group and a user share the name. Pass `--public` to make the file visible to everyone who can see the project. You cannot use `--visibility` and `--public` together:

```Shell
youtrack issue attachment add DEMO-42 --file trace.log --visibility "Security Team" --visibility user:jane.doe
```

The CLI sends the file and its restriction in one request:

- If you omit both flags, the CLI uploads the file as a normal multipart form.
- If you pass a visibility flag, the CLI encodes the file in memory and sends it in a JSON body. If the server rejects this format, the command fails. The CLI does not fall back to a public upload.
- If the server returns a different visibility, the CLI tries to delete the attachment and reports an error.

<warning>

A server workflow can change the visibility of an attachment. The cleanup step cannot undo an exposure of the file that occurs before the CLI deletes it.

</warning>

### Download an attachment by ID or name

Save an attachment by ID, or select it by its unique filename with `--name`:

```Shell
youtrack issue attachment download DEMO-42 5-12 --output screenshot.png
youtrack issue attachment download DEMO-42 --name screenshot.png --output screenshot.png
```

Pass `--output -` to write the bytes to standard output. `issue attachment list --json` and `issue attachment add --json` return signed attachment URLs. A signed URL works without a token. Share the downloaded file, and keep these URLs private.

## Apply a command to change several fields

A command is a space-delimited keyword-value instruction. It is the same instruction you type in the Apply Command dialog of YouTrack. Apply a command to one or more issues with `youtrack issue command`. Separate issue IDs with commas and no spaces:

```Shell
youtrack issue command DEMO-42 "State Fixed"
youtrack issue command DEMO-1,DEMO-2 "assignee jane.doe Priority Critical"
```

One command can set several fields, add tags, or change the State.

To preview the suggestions for a command without applying it, use `issue command suggest`:

```Shell
youtrack issue command suggest DEMO-42 "State "
```

If a command names a tag that does not exist, YouTrack creates the tag. Pass `--no-new-tags` to refuse such a command before YouTrack applies it:

```Shell
youtrack issue command DEMO-42 "tag regression" --no-new-tags
```

The `--no-new-tags` check reads the command preview from `issue command suggest`. The preview follows your YouTrack language. If the preview is not in English or the CLI cannot verify it, the CLI rejects the command and applies nothing. The check is a best-effort check, not an atomic server-side restriction: tags or settings can change before execution. To add an existing tag exactly, use `issue tag add`.

<note>

If a command sets `assignee` or `reporter`, the CLI wraps a login that contains a dot, a hyphen, or `@` in braces for the command parser of YouTrack. Thus `assignee jane.doe` becomes `assignee {jane.doe}`. Write the login as-is after these two keywords. For other user fields, such as `Reviewer`, put the braces in yourself: `Reviewer {jane.doe}`.

</note>

## Set one custom field directly

Read or set one custom field value with `youtrack issue field`:

```Shell
youtrack issue field view DEMO-42 State
youtrack issue field set DEMO-42 State Fixed
youtrack issue field set DEMO-42 Estimation 2h --type period
```

If the field is not an enum field, pass `--type` (`period`, `text`, `user`, `int`, or `string`), so the CLI interprets the value correctly. Without `--type`, the CLI treats the value as an enum value.

Repeat `--value` to set several values of a multi-value field. The list replaces the current values. `--clear` empties a field. The CLI sends values as-is, so names with spaces or dots need no escaping:

```Shell
youtrack issue field set DEMO-42 Subsystem --value "Platform. Auth & Licensing" --value UI
youtrack issue field set DEMO-42 Assignee --clear
```

Use `issue field list-values <issueId> <field>` to list the choices that are visible in the context of the issue. The field argument accepts the field name or its opaque ID. Results support `--json`, `--plain`, and `--fields`. Page with `--limit` (default 50) and `--offset`. `--limit` must be greater than 0, unlike `issue list`:

```Shell
youtrack issue field list-values DEMO-42 Priority --limit 50 --offset 50 --json
youtrack issue field list-values DEMO-42 State --fields id,name --plain
```

The command requires an instance that supports issue-field value discovery. It does not support scalar fields or state-machine transitions. If discovery is unavailable or forbidden, the command returns the server error. For user fields, the command shows logins. Pass a login to `issue field set` with `--type user`.

## Watch and vote on issues

Subscribe to notifications or vote for an issue:

```Shell
youtrack issue watch DEMO-42
youtrack issue unwatch DEMO-42
youtrack issue vote DEMO-42
youtrack issue unvote DEMO-42
```

## Prepare an issue as a draft

A draft is an issue stub that the project's issue template fills in. You complete the draft and then submit it as a real issue. Create a draft, update its fields, then submit it:

```Shell
youtrack issue draft create --project DEMO --summary "Sprint retro action items"
youtrack issue draft update 3-45 --custom-fields '{"Type":"Task","Priority":"Normal"}'
youtrack issue draft submit 3-45
youtrack issue draft delete 3-45
```

`issue draft create` returns the draft ID, for example `3-45`. Pass the draft ID to `update`, `submit`, and `delete`. `create` and `update` accept `--description` and `--description-file` (`-` reads standard input). `update` also accepts `--custom-fields` and `--custom-fields-file`. `issue draft delete` discards a draft and asks for confirmation unless you pass `--force`.

## View the sprints of an issue

List the sprints that contain an issue:

```Shell
youtrack issue sprint list DEMO-42
```

To work with the boards and sprints, see [Planning Tools](youtrack-cli-planning-tools.md).

## Log work in minutes

Track the time spent on an issue with `youtrack issue workitem`:

```Shell
youtrack issue workitem create DEMO-42 --minutes 90 --text "Implemented the fix"
youtrack issue workitem list DEMO-42
youtrack issue workitem view DEMO-42 <workItemId>
youtrack issue workitem update DEMO-42 <workItemId> --minutes 120
youtrack issue workitem delete DEMO-42 <workItemId>
```

`--minutes` is required when you create a work item. For example, `90` means 1 h 30 m. Pass `--text` for a description and `--type` with a work type ID. Both flags are optional. `workitem delete` asks for confirmation unless you pass `--force`.

## Link commits to an issue

A version control system (VCS) integration connects a YouTrack project to a repository. List the commits and pull requests linked to an issue, and link a commit, with `youtrack issue vcs`:

```Shell
youtrack issue vcs list DEMO-42
youtrack issue vcs view DEMO-42 <changeId>
youtrack issue vcs attach DEMO-42 1a2b3c4d5e6f7081
youtrack issue vcs detach DEMO-42 1a2b3c4d5e6f7081
```

`attach` takes a commit hash of 12 characters or more. YouTrack looks up the hash in the VCS integrations of the issue's project. Thus an integration must see the commit. If no integration sees the commit, the command reports that no change with that version exists. A Generic VCS integration accepts any hash and takes the commit message from `--message`.

`detach` (aliases `delete`, `remove`) unlinks a change. It accepts the change ID or the commit hash. The CLI selects the mechanism that YouTrack accepts for the change:

- If you attached the commit by hand, the CLI deletes the record.
- If a VCS integration attached the commit, the CLI detaches the record, because the server refuses to delete it.

YouTrack keeps a detached record only if the commit message names the issue. The record stops the integration from attaching the commit again. In all other cases, YouTrack removes the change. `attach` restores a record that YouTrack keeps.

Changes to VCS records require the *Update Issue Private Fields* permission. To suppress notifications about the change, pass the global `--mute-notifications` flag. This flag also requires the *Apply Commands Silently* permission.

## Restrict visibility to groups and users

To restrict a new issue from the start, pass `--visibility` or `--public` to `issue create` (see [Create an issue](#create-an-issue-in-one-request)).

Limit who can see an issue, a comment, or an attachment with permitted groups and users. Groups take a name or an ID. Users take a login or an ID.

<warning>

Each call replaces both lists. A list you leave out is cleared. Thus `--users alice` alone makes the issue visible to alice only, even if a group had access before.

</warning>

```Shell
youtrack issue visibility DEMO-42 --groups "Core Team" --users alice
youtrack issue comment visibility DEMO-42 4-17 --users alice
youtrack issue attachment visibility DEMO-42 12-8 --groups "Core Team"
youtrack issue visibility DEMO-42 --public
```

`--public` removes the restriction. The command reads the visibility back after it writes it. If the server reports a different value, the command fails. If you pass no group, no user, and no `--public`, the command fails: YouTrack treats empty lists as no restriction, which makes the issue public.

To keep a comment private from the moment you post it, pass the audience to `issue comment create`. `--visibility` takes a group name or a user login and you can repeat it. Prefix the value with `group:` or `user:` if a group and a user share the name:

```Shell
youtrack issue comment create DEMO-42 --text "Internal note" --visibility "Core Team" --visibility user:alice
```

<seealso>
<category ref="04">
<a href="youtrack-cli-get-started.md">Getting Started</a>
<a href="youtrack-cli-managing-projects.md">Managing Projects</a>
<a href="youtrack-cli-planning-tools.md">Planning Tools</a>
<a href="youtrack-cli-scripting.md">Scripting and Automation</a>
</category>
<category ref="06">
<a href="youtrack-cli-authentication.md">Authentication</a>
<a href="youtrack-cli-configuration.md">Configuration</a>
</category>
</seealso>
