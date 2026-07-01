# Security & Token Permissions

## Token Scope Disclaimer

> **Important:** YouTrack currently does not support granular permission scoping for personal access tokens.

Any token you create via **Profile → Account Security → New Token** receives access to everything your YouTrack account can do — there is no way to issue a token restricted to, say, read-only access or a single project. This means:

- A token used by `youtrack` can create, update, and delete issues, projects, and users — anything your account permits.
- If a token is leaked or misused, the blast radius equals your full YouTrack account permissions.

**Before using `youtrack` in automated pipelines or sharing it with AI agents:**
- Understand which commands mutate data (see table below).
- Use the `YOUTRACK_RO=1` guard when only reads are needed.
- Store tokens in secrets managers (GitHub Actions secrets, Vault, etc.) — never in plain text files committed to version control.
- Rotate tokens regularly and revoke them when a pipeline or integration is decommissioned.

## Read-Only Mode

Set `YOUTRACK_RO=1` to prevent any write operations at the client level:

```bash
export YOUTRACK_RO=1
youtrack issue list --query "assignee: me"   # works
youtrack issue add --project MYPROJ --summary "..."   # error: read-only mode is enabled (YOUTRACK_RO=1): write operations are blocked
```

This flag blocks all `POST`, `PUT`, `DELETE`, and multipart upload requests at the HTTP client layer — no command-level checks required, so it protects against accidental writes regardless of which command is invoked.

This is particularly useful for:
- AI agent integrations where you want to allow read-only introspection of YouTrack state.
- CI jobs that only query issues (e.g. generating release notes) and should never mutate data.
- Developer exploration sessions where you want a safety net against typos.

## Command Mutability Reference

| Command | Mutates data? |
|---|---|
| `issue get` / `issue list` / `issue count` / `issue suggest` | No |
| `issue add` / `issue update` / `issue delete` | **Yes** |
| `issue command` / `issue assign` / `issue unassign` | **Yes** |
| `issue watch` / `issue unwatch` / `issue vote` / `issue unvote` | **Yes** |
| `issue tag` / `issue visibility` | **Yes** |
| `issue comment list` | No |
| `issue comment add` / `issue comment update` / `issue comment delete` | **Yes** |
| `issue comment pin` / `issue comment visibility` | **Yes** |
| `issue link list` | No |
| `issue link add` / `issue link remove` | **Yes** |
| `issue link type list` | No |
| `issue link type add` | **Yes** |
| `issue attachment list` | No |
| `issue attachment add` / `issue attachment delete` | **Yes** |
| `issue field get` | No |
| `issue field set` | **Yes** |
| `issue reaction add` / `issue reaction delete` | **Yes** |
| `issue draft create` | **Yes** |
| `issue workitem add` | **Yes** |
| `project list` / `project get` | No |
| `project add` / `project delete` / `project archive` | **Yes** |
| `project team add-user` / `project team add-group` | **Yes** |
| `project field list` | No |
| `project field add` | **Yes** |
| `project timetrack set` / `project workflow add` | **Yes** |
| `project template list` | No |
| `board list` / `board get` | No |
| `board add` / `board delete` | **Yes** |
| `board sprint list` / `board sprint get` | No |
| `board sprint add` | **Yes** |
| `user list` / `user get` / `user current` | No |
| `user add` / `user delete` / `user ban` / `user unban` | **Yes** |
| `user pin-project` | **Yes** |
| `user locale set` / `user timezone set` | **Yes** |
| `user timezone get` | No |
| `user profile *` | No (reads only) |
| `user drafts list` | No |
| `user drafts delete` | **Yes** |
| `auth status` | No |
| `auth get-token` / `auth refresh-token` / `auth logout` | **Yes** (local config only) |
| `instance add` / `instance use` / `instance remove` / `instance update-token` | **Yes** (local config only) |
| `instance list` | No |
