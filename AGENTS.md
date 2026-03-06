# Repository Guidelines

## Project Structure & Module Organization
This repository is a flat, ESM-based Node.js CLI project.

- `youtrack-cli.js`: command registry and main entrypoint (`youtrack-cli <command>`)
- `http-helpers.js`: HTTP and parsing helpers shared by commands
- `api-paths.js`: centralized REST endpoint constants
- `test.js`: Node test runner integration tests against a real YouTrack instance
- `README.md`: user-facing command examples and supported feature areas

Keep new command logic in `youtrack-cli.js`, and move reusable request/parsing code into `http-helpers.js`.

## Build, Test, and Development Commands
- `npm install`: install dependencies
- `npm run start -- <command> [--param value]`: run CLI locally
- `npm run setup`: run `npm link` to expose global `youtrack-cli`
- `npm test`: run integration tests (`node --test test.js`)
- `npm run test:watch`: watch mode for iterative test runs

Example:
```bash
npm run start -- list-commands
node --test --test-name-pattern="get-issue" test.js
```

## Coding Style & Naming Conventions
- Use modern ESM (`import`/`export`) and Node 18+ compatible APIs.
- Match existing style: 2-space indentation, double quotes, semicolons.
- Command names use kebab-case (e.g., `get-current-user`); JS identifiers use camelCase.
- Prefer small helpers for repeated lookup/validation logic.
- Keep output JSON-serializable and errors actionable.

No dedicated formatter/linter is configured; keep changes style-consistent with surrounding code.

## Testing Guidelines
Tests use Node’s built-in `node:test` framework and run against a live YouTrack instance.

- Configure `YOUTRACK_BASE_URL` and `YOUTRACK_TOKEN` (or generate via `get-token --setEnv true`).
- Add integration coverage for every new command or branch.
- Follow current naming style: `describe("area")` and `it("command-name")`.
- Clean up created resources in `after()` hooks or inside test blocks.

## Commit & Pull Request Guidelines
Git history follows Conventional Commit prefixes: `feat:`, `fix:`, `chore:`, `docs:`.

- Write imperative, scoped commit messages (e.g., `fix: handle missing issue link type`).
- Keep commits focused; avoid mixing refactors with behavior changes.
- PRs should include: summary, rationale, test evidence (`npm test` output), and linked issue/task.
- Update `README.md` examples and command tables when adding or changing commands.
