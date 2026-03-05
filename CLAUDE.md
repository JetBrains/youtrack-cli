# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Running the CLI

```bash
npm start -- <command> [--param value ...]
node youtrack-cli.js list-commands  # list all commands
```

## Environment Setup

Two env vars are required for most commands:

```bash
export YOUTRACK_BASE_URL=https://your-instance.youtrack.cloud
export YOUTRACK_TOKEN=perm:...
```

Alternatively, run `get-token --login ... --password ... --setEnv true` which writes `~/.youtrack-cli-env` (loaded automatically on each invocation).

## Architecture

The project is a **zero-dependency, ESM Node.js CLI** designed for use by AI agents and developers interacting with the YouTrack REST API. There are three source files:

| File | Role |
|---|---|
| `youtrack-cli.js` | Command registry and `main()` entrypoint |
| `http-helpers.js` | HTTP primitives (`doGet`, `doPost`, `doPut`, `doDelete`, `httpReqBasicAuth`) and data-parsing utilities |
| `api-paths.js` | Named constants for all YouTrack REST API paths (`API.*`) |

### Command registration pattern

Every command is registered with `cmd(name, description, params[], handler)`:

- `params` entries prefixed with `?` are optional (e.g., `"?description"`).
- The handler receives `(ctx, args)` where `ctx = { base, token }` and `args` maps param names to CLI-supplied values.
- All handlers are `async` and return JSON-serializable data, which `main()` prints as pretty JSON.

### Documentation requirement

When adding a new command:
- Add a usage example to the "Examples" section of `README.md`.
- Ensure the command is listed in the "Available commands" table in `README.md`.
- Add the command to the relevant table in the "Command reference" section of `.claude/skills/youtrack-cli/SKILL.md`.
- If the command is part of a non-trivial process, add it to the "Common workflows" section of `.claude/skills/youtrack-cli/SKILL.md`.

### Argument parsing

CLI arguments are `--key value` pairs. Kebab-case flags are converted to camelCase before lookup (e.g., `--base-url` → `baseUrl`). Required params are validated before the handler runs; missing ones exit with a usage message.

### Auth

- Most commands use Bearer token auth via the `Authorization: Bearer <token>` header.
- `get-token` uses HTTP Basic auth (`httpReqBasicAuth`) to exchange login/password for a permanent token.
- `list-commands` and `get-token` are the only commands that skip token validation.

## ID Conventions

| Entity | Identifier | Example |
|---|---|---|
| Issue | `<PROJECT>-<n>` | `FOO-42` |
| Project | short name | `FOO` |
| User | login | `john.doe` |
| Group | name | `Developers` |
| Role | name | `Developer` |
| Everything else | opaque `id` field from GET responses | `2-7` |

## API Conventions

- **Visibility**: `permittedGroups` and `permittedUsers` are comma-separated names/logins. Empty = visible to all.
- **Long text**: params like `?description`, `?content`, `?text` also accept a `...File` variant (e.g. `--descriptionFile ./notes.md`).
- **Custom field bundles**: enum/state/version/build fields require a bundle to hold allowed values. The bundle is created and linked separately from the field prototype.

## Common Gotchas

- **`add-project` requires `--leader`**: the API rejects project creation without a leader login. Use `get-current-user` to get the current user's login if none is specified.
- **`apply-command` with dotted/hyphenated logins**: the CLI auto-wraps `assignee`/`reporter` values containing `.`, `-`, or `@` in braces (e.g. `alice.qa` → `{alice.qa}`). Write the login as-is; the CLI handles the wrapping.
- **`--asUser <login>`** sets `reporter`/`author` on the created entity. The target user must have at least a Contributor role in the project.

