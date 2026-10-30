# Command Reference

## Auth

```bash
youtrack auth login [<url-or-alias>] [--alias <name>] [--no-browser] [--scope <ids>] [--token-name <name>] [--session] [--client-id <id>] [--port N] [--timeout <dur>] [--login <l> --password <p>]   # browser sign-in → permanent token; paste fallback
youtrack auth status                        # check credentials for all instances
youtrack auth logout [--instance <alias>]
youtrack auth refresh-token [<alias>]
```

`auth login` signs in through the browser (OAuth 2.0 + PKCE) and saves a permanent token; with `--no-browser`, or when the Hub has no CLI OAuth client, it prompts for a pasted token. `--login`/`--password` use deprecated Hub basic auth.
Saved tokens use the OS keyring with a mode-`0600` config fallback. Single-instance automation can use `YOUTRACK_BASE_URL` + `YOUTRACK_TOKEN`.
Auth commands mask the token in every output format; `--show-token` prints it in full in every format.

```bash
youtrack instance login [<url-or-alias>] [...]   # same command as `auth login`
youtrack instance list
youtrack instance use <alias>
youtrack instance version                        # active instance's YouTrack version + build number
youtrack instance delete [--force] <alias>       # alias: remove
```

## Global Flags

```
--json / -j            Force JSON output (rendering only; still uses each command's default field
                       projection — add --fields for extra data like custom fields, comments, votes, links)
--plain                Force tab-separated output (no headers, pipeline-friendly)
--fields FIELDS / -f   Comma-separated field projection, e.g. id,summary,reporter
                       Sub-field spec: --fields 'comments(id,text)' (shell quotes required)
--server ALIAS / -s    Use a named instance profile for this invocation (does not persist)
--base-url URL         Override YOUTRACK_BASE_URL
--token TOKEN          Override YOUTRACK_TOKEN
--verbose              Print HTTP requests/responses to stderr
--dry-run              Print the first write (method, URL, body) instead of sending it; reads still run,
                       confirmations are skipped, exit 0. Rejected for auth/instance/skill/update
--mute-notifications   Add muteUpdateNotifications=true to every POST/PUT/DELETE
                       (silent: true for issue command); needs Apply Commands Silently or HTTP 403
```

(`--format json|table|plain` still works but is a deprecated, hidden alias for `--json`/`--plain`.)

On `list`-style commands that support `--limit`, pass it to cap the result size on large
projects/queries instead of pulling every record.

Destructive commands (deletes, `user ban`, `project archive`, `settings feature`, link/reaction
removals, etc.) prompt for confirmation and accept `--force` to skip the prompt.
Declining a destructive-action confirmation exits `0` and reports `{"status":"canceled","message":"Canceled."}` in JSON mode; table/plain output remains `Canceled.`.

## issue

```bash
youtrack issue view <issueId> [--comments N] [--no-description] [--only-fields]
                                       # alias: get; --only-fields makes --fields replace the default projection
youtrack issue list --query Q [--limit N] [--offset N]
youtrack issue create --project P --summary S [--description D | --description-file <path|->] [--parent <issueId>]
                                       # alias: add
youtrack issue update <issueId> [--summary S] [--description D | --description-file <path|->]
                                       # --description "" clears it; omitted fields are preserved
youtrack issue delete <issueId>
youtrack issue command <issueId[,issueId...]> <query> [--no-new-tags]   # apply a YouTrack command string
                                       # --no-new-tags refuses a command that would create a tag
youtrack issue command suggest <issueId[,issueId...]> <query>   # preview suggestions without applying
youtrack issue count [--query Q]       # may return -1 on first call (async); retry manually
```

`issue create --custom-fields` and `--custom-fields-file` require one complete JSON object of string values. Invalid JSON, trailing content, and `null` values fail before creating an issue.

**social actions**
```bash
youtrack issue assign <issueId> <login>
youtrack issue unassign <issueId>
youtrack issue watch <issueId>
youtrack issue unwatch <issueId>
youtrack issue vote <issueId>          # cannot vote on your own issues
youtrack issue unvote <issueId>
youtrack issue tag list <issueId>
youtrack issue tag add <issueId> <tagId|name>      # exact name; fails on no match or ambiguity, never creates
youtrack issue tag remove <issueId> <tagId|name>   # name is matched among the issue's tags
youtrack issue visibility <issueId> [flags]
```

**comment**
```bash
youtrack issue comment list <issueId> [--include-deleted]   # JSON includes deleted, pinned, visibility
youtrack issue comment create <issueId> (--text "..." | --text-file <path|->)   # alias: add
youtrack issue comment update <issueId> <commentId> (--text "..." | --text-file <path|->)
youtrack issue comment delete <issueId> <commentId>
youtrack issue comment pin <issueId> <commentId>
youtrack issue comment visibility <issueId> <commentId> [flags]
```

**link**
```bash
youtrack issue link list <issueId>
youtrack issue link add <issueId> --target <targetId> --type <linkTypeId>
youtrack issue link remove <issueId> <linkId>   # linkId from 'issue link list --json | jq .[].id'
youtrack issue link type list
youtrack issue link type create --name N --source-to-target S --target-to-source T [flags]   # alias: add
```

**attachment**
```bash
youtrack issue attachment list <issueId>
youtrack issue attachment add <issueId> --file <filepath>
youtrack issue attachment delete <issueId> <attachmentId>
```

**field / reaction / draft / workitem**
```bash
youtrack issue field view <issueId> <fieldName>   # alias: get
youtrack issue field set <issueId> <fieldName> <value>
youtrack issue field set <issueId> <fieldName> --value V [--value V...]   # multi-value; replaces current values
youtrack issue field set <issueId> <fieldName> --clear
youtrack issue reaction add <issueId> <commentId> <reaction>
youtrack issue reaction delete <issueId> <commentId> <reactionId>
youtrack issue draft create [--project P] [--summary S]
youtrack issue draft delete <draftId>             # prompts; --force to skip
youtrack issue workitem create <issueId> --minutes N [--text T] [--type ID]   # alias: add
youtrack issue workitem list <issueId>
youtrack issue workitem view <issueId> <workItemId>
youtrack issue workitem update <issueId> <workItemId> [--minutes N] [--text T]
youtrack issue workitem delete <issueId> <workItemId>
youtrack issue vcs list <issueId>
youtrack issue vcs view <issueId> <changeId>              # alias: get
youtrack issue vcs attach <issueId> <commitHash> [--message TEXT]
youtrack issue vcs detach <issueId> <changeIdOrHash>      # aliases: delete, remove
                                       # attach/detach take --mute-notifications
```

## project

```bash
youtrack project list
youtrack project view <shortName>       # alias: get
youtrack project create --shortName P --name "..." --leader <login> [--template helpdesk]
                                       # alias: add; --leader required (API rejects without it)
youtrack project update <shortName> [--shortName P] [--name NAME] [--description TEXT] [--leader LOGIN] [--archived[=false]]
youtrack project delete <shortName>
youtrack project archive <shortName>
```

```bash
# ⚠ Both require opaque IDs, not logins/names
youtrack project team add-user <shortName> <userId>    # userId: 'user view <login> --json | jq .id'
youtrack project team add-group <shortName> <groupId>  # groupId: 'group view <name> --json | jq .id'
youtrack project team list-users <shortName>
youtrack project team list-groups <shortName>
youtrack project team remove-user <shortName> <userId>
youtrack project team remove-group <shortName> <groupId>
youtrack project field list <shortName>
youtrack project field add <shortName> [flags]
youtrack project field remove <shortName> <projectFieldId>
youtrack project template list
youtrack project template convert <shortName>
youtrack project timetrack set <shortName> --enabled[=false]
youtrack project workflow add <shortName> [flags]
```

## user

```bash
youtrack user list [--query Q]
youtrack user view <login>              # alias: get
youtrack user current                   # alias: me
youtrack user create --login L --fullName "..." --email E --password P [--type standard|agent|reporter]
                                       # alias: add
youtrack user delete <login> --successor <login>   # --successor required; cannot delete yourself
youtrack user ban <login>
youtrack user unban <login>
```

```bash
youtrack user drafts issues <login>                # draft issues for the given user (login or 'me')
youtrack user drafts delete <login>                # prompts; --force to skip
youtrack user drafts articles <login>
youtrack user locale set <login> <locale>
youtrack user timezone view <login>                # alias: get
youtrack user timezone set <login> <timezoneId>    # e.g. "Europe/Berlin"
youtrack user profile view <login>       # alias: get; general profile snapshot
youtrack user profile general|appearance|notifications|articles|issues-list|grazie|tips|property <login> [flags]
youtrack user profile timetracking view <login>
```

**user profile examples**

| Subcommand | Payload |
|---|---|
| `general` | `'{"timezone":{"id":"Europe/Berlin"}}'` |
| `appearance` | `'{"theme":"DARK","language":"en"}'` |
| `notifications` | `'{"notifyOnOwnChanges":false}'` |
| `articles` | `'{"showSidebar":true}'` |
| `issues-list` | `'{"defaultQuery":"for: me"}'` |

## whiteboard

```bash
youtrack whiteboard list
youtrack whiteboard create --name NAME [--owner LOGIN]   # alias: add
youtrack whiteboard view <id>                             # alias: get
youtrack whiteboard delete <id>
youtrack whiteboard reload <id>
youtrack whiteboard node list <whiteboardId>
youtrack whiteboard node view <whiteboardId> <cardId>      # alias: get
youtrack whiteboard node create <whiteboardId> --x N --y N [--label L] [--description D]  # alias: add
youtrack whiteboard links <whiteboardId>
youtrack whiteboard link <whiteboardId> --source <cardId> --target <cardId> [--link-type ONE_WAY|TWO_WAY|PROTOTYPED] [--link-prototype <id>]
```

## board

```bash
youtrack board list
youtrack board view <id>                       # alias: get
youtrack board create --name NAME --project P  # alias: add
youtrack board update <id> [--name NAME]
youtrack board delete <id>
youtrack board sprint list --board <boardId>
youtrack board sprint view --board <boardId> <sprintId>   # alias: get
youtrack board sprint create --board <boardId> --name NAME [--start <epochMs>] [--end <epochMs>] [--goal G]
                                                   # alias: add
youtrack board sprint update --board <boardId> <sprintId> [--name NAME] [--goal G] [--start <epochMs>] [--end <epochMs>] [--archived[=false]]
youtrack board sprint delete --board <boardId> <sprintId>
```

## article

```bash
youtrack article list [--query Q]
youtrack article view <articleId>       # alias: get
youtrack article create --project P --summary S [--content C] [--asUser login]
                                       # alias: add
youtrack article update <articleId> [--summary S] [--content C]
youtrack article delete <articleId>
youtrack article comment list <articleId>
youtrack article comment view <articleId> <commentId>
youtrack article comment create <articleId> --text "..."   # alias: add
youtrack article comment update <articleId> <commentId> --text "..."
youtrack article comment delete <articleId> <commentId>
youtrack article comment pin <articleId> <commentId> [--unpin]
youtrack article comment visibility <articleId> <commentId> [--groups <ids>] [--users <ids>]
youtrack article comment reaction delete <articleId> <commentId> <reactionId>   # prompts; --force to skip
youtrack article attachment list <articleId>
youtrack article attachment add <articleId> --file <filepath>
youtrack article attachment delete <articleId> <attachmentId>
youtrack article tag <articleId> <tagId>
youtrack article untag <articleId> <tagId>
```

## raw API

```bash
youtrack raw <path> [--method GET|POST|PUT|DELETE] [--body JSON | --body-file PATH|-] [--fields FIELDS]
youtrack raw <path> --output PATH|-        # response body as-is, no JSON parsing
youtrack raw <collection-path> --paginate [--page-size N]
```

`--paginate` returns one combined JSON array, including `[]` for an empty collection.
Without it YouTrack returns at most 42 items; `raw` warns on stderr when a list hits exactly 42.
`--body-file` reads the body from a file (`-` for stdin) — use it for long bodies, which can exceed the Windows command-line limit.
`--output` writes binary or non-JSON responses (images, files) unchanged; `-` writes them to stdout.

## dashboard

```bash
youtrack dashboard list
youtrack dashboard create --name NAME     # alias: add
youtrack dashboard update <id> --name NAME
youtrack dashboard delete <id>
```

## tag

```bash
youtrack tag list [--query Q] [--limit N]  # Q: name word prefix, case-insensitive; default limit 100
youtrack tag create --name NAME [flags]   # alias: add
youtrack tag delete <id>
youtrack tag issue list <tagId>           # view issues for a tag
```

## search

```bash
youtrack search list
youtrack search view <id>                          # alias: get
youtrack search create --name NAME --query Q       # alias: add
youtrack search delete <id>
youtrack search share <id>
```

## group

```bash
youtrack group list [--query Q]
youtrack group view <name>              # alias: get
youtrack group create --name NAME       # alias: add
youtrack group delete <name>
youtrack group member list <groupId>
youtrack group member add <groupId> <userId>
youtrack group member remove <groupId> <userId>
youtrack group role list <groupId>                     # alias of 'role assignments --holder'
youtrack group role add <groupId> --role <roleId> [--project <projectId>|--org <orgId>]  # alias of 'role grant --group'
```

## role

```bash
youtrack role list
youtrack role view <id>                # alias: get
youtrack role grant --user <userId>|--group <groupId> --role <roleId> [--project <projectId>|--org <orgId>]
youtrack role revoke <assignedRoleId>
youtrack role assignments [--holder <userId>]
```

`issue create --type Task --tag release --custom-fields '{"Assignee":"me"}'` sends fields, type and existing tags in the initial create request. `--tag` is repeatable and accepts an exact tag name or ID. Invalid fields, tags and parent references are rejected before creation.
