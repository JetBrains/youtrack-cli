# Workflows

## YouTrack query syntax

Used by `get-issues`, `get-issue-count`, and `apply-command`:

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
youtrack-cli get-issues --query "assignee: me State: Open"
youtrack-cli apply-command --query "State Fixed" --issueIds "MYPROJ-42"
```

**Create an issue with a file description:**
```bash
youtrack-cli add-issue --projectShortName MYPROJ --summary "RFC: Auth" --descriptionFile ./rfc.md
```

**Set issue type after creation (Type always defaults to "Bug"):**
```bash
youtrack-cli add-issue --projectShortName MYPROJ --summary "New feature"
youtrack-cli set-custom-field --issueId MYPROJ-1 --fieldName Type --fieldValue Feature --fieldType enum
```

**Link issues:**
```bash
youtrack-cli get-issue-link-types                          # find valid link types
youtrack-cli link-issues --issueId MYPROJ-42 --linkedIssueId MYPROJ-10 --linkType "depends on"
```

**Upload attachments:**
```bash
youtrack-cli add-attachment --issueId MYPROJ-1 --file ./screenshot.png
youtrack-cli add-article-attachment --articleId MYPROJ-A-1 --file ./report.pdf
```

**Grant a role:**
```bash
youtrack-cli get-roles                                     # find roleName
youtrack-cli get-user --login john.doe                     # get userId

# Project scope
youtrack-cli grant-role-to-user --userId <id> --roleName Developer --projectName "My Project"

# Global scope (omit --projectName)
youtrack-cli grant-role-to-user --userId <id> --roleName "System Admin"
```

**Create a role with permissions:**
```bash
# Without permissions
youtrack-cli add-role --name "New Role" --description "Role description"

# With permissions — find IDs first, then create
youtrack-cli get-permissions
youtrack-cli add-role --name "New Role" --permissions "JetBrains.YouTrack.READ_ISSUE,JetBrains.YouTrack.CREATE_ARTICLE"
```

**Update an existing role's permissions (replaces full list):**
```bash
youtrack-cli get-roles                  # find roleId
youtrack-cli get-permissions            # find permission IDs
youtrack-cli update-role --roleId 17-6 --permissions "jetbrains.jetpass.organization-read,jetbrains.jetpass.organization-delete"
```

**Create an issue as user A, comment as user B, assign user C:**
```bash
# 1. Create project — leader becomes a valid assignee
youtrack-cli add-project --shortName PROJ --name "My Project" --leader carol

# 2. Grant Contributor role so --asUser works
youtrack-cli get-user --login alice                          # get userId
youtrack-cli grant-role-to-user --userId <aliceId> --roleName Contributor --projectName "My Project"
youtrack-cli grant-role-to-user --userId <bobId>  --roleName Contributor --projectName "My Project"

# 3. Create / comment / assign
youtrack-cli add-issue   --projectShortName PROJ --summary "Issue title" --asUser alice
youtrack-cli add-comment --parentId PROJ-1 --text "My comment" --asUser bob
youtrack-cli set-custom-field --issueId PROJ-1 --fieldName Assignee --fieldValue carol --fieldType user
```

**Create entities on behalf of another user:**
```bash
youtrack-cli add-issue   --projectShortName MYPROJ --summary "Bug report"    --asUser alice.qa
youtrack-cli add-comment --parentId MYPROJ-1 --text "Investigated."          --asUser bob.lead
youtrack-cli add-article --projectShortName MYPROJ --summary "How-to guide"  --asUser carol.obs
youtrack-cli add-comment --parentId MYPROJ-A-1 --place articles --text "Feedback." --asUser alice.qa
```

**Set assignee via apply-command (dotted logins auto-wrapped):**
```bash
youtrack-cli apply-command --query "assignee alice.qa" --issueIds "MYPROJ-1"
youtrack-cli apply-command --query "Type Bug assignee bob.lead State Open" --issueIds "MYPROJ-2"
```
