# Workflows

## YouTrack query syntax

Used by `issue list`, `issue count`, and `issue command`:

```
assignee: me State: Open
assignee: me State: {In Progress}    # braces for values with spaces
project: MYPROJ Priority: Critical
created: today
#MYPROJ-42                           # specific issue
tag: bug
```

---

## Common workflows

**Find and update an issue:**
```bash
youtrack issue list --query "assignee: me State: Open"
youtrack issue command MYPROJ-42 "State Fixed"
```

**Create an issue with a file description:**
```bash
youtrack issue add --project MYPROJ --summary "RFC: Auth" --description-file ./rfc.md
```

**Bulk update via query:**
```bash
youtrack issue list --query "project: MYPROJ State: Open" --json | jq -r '.[].id' \
  | xargs -I{} youtrack issue command {} "Priority Critical"
```

**Link issues:**
```bash
youtrack issue link type list                          # find valid link type IDs
youtrack issue link add MYPROJ-42 --target MYPROJ-10 --type <linkTypeId>
```

**Upload attachments:**
```bash
youtrack issue attachment add MYPROJ-1 ./screenshot.png
youtrack article attachment add MYPROJ-A-1 ./report.pdf
```

**Grant a role:**
```bash
youtrack role list                                     # find roleId
youtrack user get john.doe --json | jq .id            # get opaque userId

# Project scope
youtrack role grant --user <userId> --role <roleId> --project MYPROJ

# Global scope
youtrack role grant --user <userId> --role <roleId>
```

**Add user to project team:**
```bash
youtrack user get alice --json | jq .id               # get opaque userId
youtrack project team add-user MYPROJ <userId>
```

**Add group to project team:**
```bash
youtrack group get Developers --json | jq .id         # get opaque groupId
youtrack project team add-group MYPROJ <groupId>
```

**Create issue and set custom field:**
```bash
youtrack issue add --project MYPROJ --summary "New feature"
youtrack issue command MYPROJ-1 "Type Feature"
```

**Work with whiteboards:**
```bash
youtrack whiteboard add --name "Sprint Planning"
youtrack whiteboard node add <wbId> --x 100 --y 100 --label "Start here"
youtrack whiteboard link <wbId> --source <nodeA> --target <nodeB> --link-type TWO_WAY
```
