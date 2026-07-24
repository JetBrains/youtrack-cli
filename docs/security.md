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

## Token Storage

Tokens saved via `auth login`, `instance add`, or `auth refresh-token` are stored in the OS keyring when one is available:

- **macOS** — Keychain
- **Windows** — Credential Manager
- **Linux** — Secret Service (GNOME Keyring, KWallet, etc. via D-Bus)

Each token is stored under the service name `youtrack-cli` with the instance alias as the account. The config file (`~/.youtrack-cli-config.json`) then contains only the instance URL — no secret.

When no keyring is available (headless Linux, CI containers, locked keychain), the CLI automatically falls back to storing the token in `~/.youtrack-cli-config.json` (mode 0600), exactly as before, and prints a one-line warning to stderr. Existing file-stored tokens are migrated to the keyring transparently the next time the config is written; the token is then removed from the file.

`instance delete` / `auth logout` remove the keyring entry along with the config entry.

To opt out of keyring storage entirely (e.g. for deterministic CI setups), set:

```bash
export YOUTRACK_DISABLE_KEYRING=1
```

Environment credentials (`YOUTRACK_TOKEN`) and the `--token` flag bypass stored credentials entirely and are unaffected. Prefer an environment secret over a command-line token in automation, because command-line arguments can appear in shell history and process listings.

## Read-Only Mode

Set `YOUTRACK_RO=1` to prevent any write operations at the client level:

```bash
export YOUTRACK_RO=1
youtrack issue list --query "assignee: me"      # works
youtrack issue create --project MYPROJ --summary "..."   # error: read-only mode is enabled (YOUTRACK_RO=1): write operations are blocked
```

This flag blocks all `POST`, `PUT`, `DELETE`, and multipart upload requests at the HTTP client layer — no command-level checks required, so it protects against accidental writes regardless of which command is invoked.

This is particularly useful for:
- AI agent integrations where you want to allow read-only introspection of YouTrack state.
- CI jobs that only query issues (e.g. generating release notes) and should never mutate data.
- Developer exploration sessions where you want a safety net against typos.

## Command Mutability Reference

Destructive commands (deletes, bans, archives, and similar) prompt for confirmation before acting; pass `--force` to skip the prompt in scripts.

| Command | Mutates data? |
|---|---|
| `issue view` / `issue list` / `issue count` / `issue command suggest` | No |
| `issue create` / `issue update` / `issue delete` | **Yes** |
| `issue command` / `issue assign` / `issue unassign` | **Yes** |
| `issue watch` / `issue unwatch` / `issue vote` / `issue unvote` | **Yes** |
| `issue tag` / `issue visibility` | **Yes** |
| `issue comment list` | No |
| `issue comment create` / `issue comment update` / `issue comment delete` | **Yes** |
| `issue comment pin` / `issue comment visibility` | **Yes** |
| `issue link list` | No |
| `issue link add` / `issue link remove` | **Yes** |
| `issue link type list` | No |
| `issue link type create` | **Yes** |
| `issue attachment list` | No |
| `issue attachment add` / `issue attachment delete` | **Yes** |
| `issue field view` | No |
| `issue field set` | **Yes** |
| `issue reaction add` / `issue reaction delete` | **Yes** |
| `issue draft create` / `issue draft update` / `issue draft submit` / `issue draft delete` | **Yes** |
| `issue workitem list` / `issue workitem view` | No |
| `issue workitem create` / `issue workitem update` / `issue workitem delete` | **Yes** |
| `article list` / `article view` | No |
| `article create` / `article update` / `article delete` | **Yes** |
| `article tag` / `article untag` / `article visibility` / `article set-parent` | **Yes** |
| `article comment list` / `article comment view` | No |
| `article comment create` / `article comment update` / `article comment delete` | **Yes** |
| `article comment pin` / `article comment visibility` | **Yes** |
| `article comment reaction list` | No |
| `article comment reaction add` / `article comment reaction delete` | **Yes** |
| `article attachment list` | No |
| `article attachment add` / `article attachment delete` | **Yes** |
| `article child list` / `article parent view` | No |
| `article child add` | **Yes** |
| `project list` / `project view` | No |
| `project create` / `project delete` / `project archive` | **Yes** |
| `project team list-users` / `project team list-groups` | No |
| `project team add-user` / `project team add-group` / `project team remove-user` / `project team remove-group` | **Yes** |
| `project field list` | No |
| `project field add` / `project field remove` | **Yes** |
| `project timetrack set` / `project workflow add` | **Yes** |
| `project template list` | No |
| `board list` / `board view` | No |
| `board create` / `board update` / `board delete` | **Yes** |
| `board sprint list` / `board sprint view` | No |
| `board sprint create` | **Yes** |
| `user list` / `user view` / `user current` | No |
| `user create` / `user delete` / `user ban` / `user unban` | **Yes** |
| `user locale set` / `user timezone set` | **Yes** |
| `user timezone view` | No |
| `user profile *` | No (reads only) |
| `user drafts issues` / `user drafts articles` | No |
| `user drafts delete` | **Yes** |
| `auth status` | No |
| `auth login` / `auth refresh-token` / `auth logout` | **Yes** (local config only) |
| `instance add` / `instance use` / `instance delete` / `instance refresh-token` / `instance login` | **Yes** (local config only) |
| `instance list` / `instance version` | No |
