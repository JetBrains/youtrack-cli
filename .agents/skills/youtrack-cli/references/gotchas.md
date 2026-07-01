# Gotchas & Tips

## CLI gotchas

- **Opaque IDs for team commands** — `project team add-user` / `project team add-group` expect opaque IDs (`2-9`, `4-1`), not logins/names. Resolve with `youtrack user get <login> --json | jq .id`.
- **`project add` requires `--leader`** — the YouTrack API rejects creation without it. Use `user me` to get the current login if none is specified.
- **`issue command` logins** — the CLI auto-wraps logins with `.`, `-`, or `@` in braces. Write the login as-is.
- **`issue count` may return -1** — the count is computed async; retry manually if needed.
- **`issue link add` HTTP 405** — known bug (F-01); use `issue command` as a workaround.
- **`user pin-project` needs opaque userId** — not login (F-06).
- **`user locale set` HTTP 500** — known bug (F-03).
- **Output format** — table on TTY, JSON when piped. Use `--json` / `--plain` to override.

## Known defaults

- **Issue Type defaults to "Bug"**: `add-issue` does not accept a `type` param — every new issue gets Type=Bug. Always follow with `set-custom-field --fieldName Type --fieldValue <type> --fieldType enum` when creating non-Bug issues. Valid values: `Bug`, `Feature`, `Task`, `Cosmetics`, `Exception`, `Usability Problem`, `Performance Problem`, `Epic`.
- **`get-issues` defaults to `$top=100`**: pass `--limit <n>` to override; `--limit -1` returns all (may be slow on large projects).

## Project creation

- **Always pass `--leader`**: the CLI param is optional but the YouTrack API rejects creation without it. If the user doesn't specify one, run `get-current-user` silently and use that login.

## `--asUser <login>`

`add-issue`, `add-comment`, and `add-article` accept `--asUser <login>` to set `reporter`/`author`. For article comments: `add-comment --place articles --asUser <login>`.

Requirements:
- Current token must have admin permissions.
- The `--asUser` user must have at least the **Contributor** role in the target project (use `grant-role-to-user` if needed).

## `apply-command` with dotted/hyphenated logins

The CLI auto-wraps `assignee` and `reporter` values containing `.`, `-`, or `@` in braces for YouTrack's command parser. Write `assignee alice.qa` — the CLI sends `assignee {alice.qa}` automatically.

## Assignee field — Hub-backed instances

In Hub-backed YouTrack, the Assignee field only accepts users who are genuine Hub project team members. `add-user-to-project-team` posts to the YouTrack REST endpoint but Hub does **not** persist those changes — the user cannot be set as Assignee afterwards.

**Only reliable way to make a user assignable:** create the project with that user as `--leader`. Project leaders are added to the Hub team at creation time.

`grant-role-to-user` grants project access (enables `--asUser`) but does **not** make the user assignable as Assignee unless they were also set as leader.

## File attachments

Always use `add-attachment --issueId ... --file ./path` for issues and `add-article-attachment --articleId ... --file ./path` for articles. Never use `curl` for uploads.

## Role creation vs reuse

When the user asks to create a role, always create a new one with a unique name — never reuse an existing role silently.

---

## Tips

- Always run a lookup command first when you need an internal ID: `get-user`, `get-projects`, `get-roles`, `get-agile-boards`, etc.
- Use `--descriptionFile` / `--textFile` / `--contentFile` for long content instead of inline strings.
- `permittedGroups` and `permittedUsers` accept comma-separated values.
- All boolean params accept `"true"` or `"false"` as strings.
- Run `youtrack-cli list-commands` to see every registered command with its full parameter list.
- Use `--fields '?'` on any command to discover available top-level response fields.
