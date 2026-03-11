# AGENTS.md

Zero-dependency ESM Node.js CLI for interacting with the YouTrack REST API. Designed for use by AI agents and developers.

## Agent Skill

A full YouTrack interaction guide is available as an agent skill — load it when the user asks to manage YouTrack entities (issues, articles, projects, users, sprints, etc.):

**`.agents/skills/youtrack-cli/SKILL.md`** — command reference, common workflows, output formats, and YouTrack-specific gotchas.

## Development Commands

```bash
npm start -- <command> [--param value ...]         # run locally
node youtrack-cli.js list-commands                 # list all commands (table in TTY, JSON when piped)
npm test                                           # integration tests against a real YouTrack instance
npm run setup                                      # npm link → exposes global `youtrack-cli`
node --test --test-name-pattern="get-issue" test.js  # run a single test by name
```

## Environment

```bash
export YOUTRACK_BASE_URL=https://your-instance.youtrack.cloud
export YOUTRACK_TOKEN=perm:...
```

Or run `get-token --login ... --password ... --setEnv true` to write `~/.youtrack-cli-env` (auto-loaded on each invocation).

## Project Structure

| File | Role |
|---|---|
| `youtrack-cli.js` | Command registry and `main()` entrypoint |
| `http-helpers.js` | HTTP primitives, data-parsing utilities, verbose/error logic |
| `api-paths.js` | REST endpoint constants (`API.*`) |
| `test.js` | Integration tests — no mocking, real YouTrack instance |

Put new command logic in `youtrack-cli.js`. Move reusable request/parse helpers to `http-helpers.js`.

## Command Registration

```js
cmd("name", "description", ["requiredParam", "?optionalParam"], async (ctx, args) => {
  // ctx = { base, token }
  return doGet(ctx.base, `${API.SOMETHING}?fields=id,name`, ctx.token);
});
```

- `?` prefix marks an optional param.
- Handlers must return JSON-serializable data — `main()` prints it.
- Flags are `--key value`; kebab-case is camelCase'd (`--base-url` → `baseUrl`).
- `list-commands` and `get-token` are the only commands that skip token validation.

## When Adding a Command

- Add a usage example and table row to `README.md`.
- Add to `.agents/skills/youtrack-cli/SKILL.md` if part of a non-trivial workflow.

## Coding Conventions

- 2-space indentation, double quotes, semicolons — match surrounding code.
- Command names: kebab-case. JS identifiers: camelCase.

## Testing

Tests run against a **real YouTrack instance — no mocking**. Set `YOUTRACK_BASE_URL` and `YOUTRACK_TOKEN` before running. Clean up all created resources in `after()` hooks.

## ID Conventions

| Entity | Identifier | Example |
|---|---|---|
| Issue | `<PROJECT>-<n>` | `FOO-42` |
| Project | short name | `FOO` |
| User | login | `john.doe` |
| Group / Role | name | `Developers` / `Developer` |
| Everything else | opaque `id` from API responses | `2-7` |

## API Conventions

- `permittedGroups` / `permittedUsers`: comma-separated names/logins. Empty = visible to all.
- Long-text params (`?description`, `?content`, `?text`) accept a `...File` variant (e.g. `--descriptionFile ./notes.md`).
- Custom field bundles: create the bundle and attach it to the field prototype as separate steps.

## Gotchas

- **IMPORTANT — `add-project` requires `--leader`**: the YouTrack API rejects project creation without it. Use `get-current-user` to get the current user's login if none is specified.
- **`apply-command` logins**: the CLI auto-wraps logins containing `.`, `-`, or `@` in braces for YouTrack's command parser (`alice.qa` → `{alice.qa}`). Write the login as-is.
- **`--asUser <login>`**: the target user must have at least a Contributor role in the project.

## Commits

Use Conventional Commit prefixes: `feat:`, `fix:`, `chore:`, `docs:`. Keep commits focused; don't mix refactors with behavior changes.
