# Command Reference

## Auth

```bash
youtrack auth get-token [--alias <name>]   # interactive auth; alias: login
youtrack auth status                        # check credentials for all instances
youtrack auth logout [--instance <alias>]
youtrack auth refresh-token [<alias>]
```

Auth methods: (1) basic login+password, (2) web browser, (3) paste token.
Saves to `~/.youtrack-cli-config.json`. Single-instance: use `YOUTRACK_BASE_URL` + `YOUTRACK_TOKEN`.

```bash
youtrack instance list
youtrack instance use <alias>
youtrack instance remove [--force] <alias>
youtrack instance update-token <alias>
```

## Global Flags

```
--json / -j            Force JSON output
--plain                Force tab-separated output (no headers, pipeline-friendly)
--format json|table|plain  Explicit format override (default: table on TTY, json when piped)
--fields FIELDS / -f   Comma-separated field projection, e.g. id,summary,reporter
                       Sub-field spec: --fields 'comments(id,text)' (shell quotes required)
--server ALIAS / -s    Use a named instance profile for this invocation (does not persist)
--base-url URL         Override YOUTRACK_BASE_URL
--token TOKEN          Override YOUTRACK_TOKEN
--verbose              Print HTTP requests/responses to stderr
```

## issue

```bash
youtrack issue get <issueId>           # alias: view
youtrack issue list [--query Q] [--limit N]
youtrack issue add --project P --summary S [--description D] [--parent <issueId>] [--description-file path]
                                       # alias: create
youtrack issue update <issueId> [--summary S] [--description D]
youtrack issue delete <issueId>
youtrack issue command <issueId[,issueId...]> <query>   # apply a YouTrack command string
youtrack issue count [--query Q]       # may return -1 on first call (async); retry manually
youtrack issue suggest <issueId[,issueId...]> <query>
youtrack issue assist <query> <issueId...>
```

**social actions**
```bash
youtrack issue assign <issueId> <login>
youtrack issue unassign <issueId>
youtrack issue watch <issueId>
youtrack issue unwatch <issueId>
youtrack issue vote <issueId>          # cannot vote on your own issues
youtrack issue unvote <issueId>
youtrack issue tag <issueId> <tagId>
youtrack issue visibility <issueId> [flags]
```

**comment**
```bash
youtrack issue comment list <issueId>
youtrack issue comment add <issueId> --text "..."
youtrack issue comment update <issueId> <commentId> --text "..."
youtrack issue comment delete <issueId> <commentId>
youtrack issue comment pin <issueId> <commentId>
youtrack issue comment visibility <issueId> <commentId> [flags]
```

**link**
```bash
youtrack issue link list <issueId>
youtrack issue link add <issueId> --target <targetId> --type <linkTypeId>   # ⚠ HTTP 405 (F-01)
youtrack issue link remove <issueId> <linkId>   # linkId from 'issue link list --json | jq .[].id'
youtrack issue link type list
youtrack issue link type add [flags]
```

**attachment**
```bash
youtrack issue attachment list <issueId>
youtrack issue attachment add <issueId> <filepath>
youtrack issue attachment delete <issueId> <attachmentId>
```

**field / reaction / draft / workitem**
```bash
youtrack issue field get <issueId> <fieldName>
youtrack issue field set <issueId> <fieldName> <value>
youtrack issue reaction add <issueId> <commentId> <reaction>
youtrack issue reaction delete <issueId> <commentId> <reactionId>
youtrack issue draft create [--project P] [--summary S]
youtrack issue draft delete <draftId>
youtrack issue workitem add <issueId> --duration <ISO8601>   # e.g. PT1H30M
```

## project

```bash
youtrack project list
youtrack project get <shortName>       # alias: view
youtrack project add --shortName P --name "..." --leader <login> [--template helpdesk]
                                       # alias: create; --leader required (API rejects without it)
youtrack project delete <shortName>
youtrack project archive <shortName>
```

```bash
# ⚠ Both require opaque IDs, not logins/names
youtrack project team add-user <shortName> <userId>    # userId: 'user get <login> --json | jq .id'
youtrack project team add-group <shortName> <groupId>  # groupId: 'group get <name> --json | jq .id'
youtrack project field list <shortName>
youtrack project field add <shortName> [flags]
youtrack project template list
youtrack project template convert <shortName>
youtrack project timetrack set <shortName> --enabled[=false]
youtrack project workflow add <shortName> [flags]
```

## user

```bash
youtrack user list [--query Q]
youtrack user get <login>              # alias: view
youtrack user current                  # alias: me
youtrack user add --login L --fullName "..." --email E --password P [--type standard|agent|reporter]
youtrack user delete <login> --successor <login>   # --successor required; cannot delete yourself
youtrack user ban <login>
youtrack user unban <login>
```

```bash
youtrack user drafts issues <login>
youtrack user drafts delete <login>
youtrack user drafts articles <login>
youtrack user locale set <login> <locale>          # ⚠ HTTP 500 (F-03)
youtrack user timezone get <login>
youtrack user timezone set <login> <timezoneId>    # e.g. "Europe/Berlin"
youtrack user profile general|appearance|notifications|articles|issues-list|grazie|tips|property <login> [flags]
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
youtrack whiteboard add --name NAME [--owner LOGIN]
youtrack whiteboard get <id>
youtrack whiteboard delete <id>
youtrack whiteboard reload <id>
youtrack whiteboard node list <whiteboardId>
youtrack whiteboard node get <whiteboardId> <cardId>
youtrack whiteboard node add <whiteboardId> --x N --y N [--label L] [--description D]
youtrack whiteboard links <whiteboardId>
youtrack whiteboard link <whiteboardId> --source <cardId> --target <cardId> [--link-type ONE_WAY|TWO_WAY|PROTOTYPED] [--link-prototype <id>]
```

## board

```bash
youtrack board list
youtrack board get <id>                # alias: view
youtrack board add --name NAME [flags] # alias: create
youtrack board update <id> [flags]
youtrack board delete <id>
youtrack board sprint list <boardId>
youtrack board sprint get <boardId> <sprintId>
youtrack board sprint add <boardId> --name NAME --start DATE --end DATE [--goal G]
```

## article

```bash
youtrack article list [--query Q]
youtrack article get <articleId>
youtrack article add --project P --summary S [--description D] [--description-file path]
youtrack article update <articleId> [--summary S] [--description D]
youtrack article delete <articleId>
youtrack article comment list <articleId>
youtrack article comment add <articleId> --text "..."
youtrack article comment update <articleId> <commentId> --text "..."
youtrack article comment delete <articleId> <commentId>
youtrack article attachment list <articleId>
youtrack article attachment add <articleId> <filepath>
youtrack article tag <articleId> <tagId>
```

## dashboard

```bash
youtrack dashboard list
youtrack dashboard add --name NAME     # alias: create
youtrack dashboard delete <id>
```

## tag

```bash
youtrack tag list
youtrack tag get <id>
youtrack tag add --name NAME [flags]
youtrack tag delete <id>
```

## search

```bash
youtrack search list
youtrack search add --name NAME --query Q
youtrack search delete <id>
```

## group

```bash
youtrack group list [--query Q]
youtrack group get <name>
youtrack group add --name NAME
youtrack group delete <name>
```

## role

```bash
youtrack role list
youtrack role get <id>
youtrack role grant --user <userId>|--group <groupId> --role <roleId> [--project <shortName>]
youtrack role revoke <assignedRoleId>
youtrack role assignments [--holder <userId>]
```
