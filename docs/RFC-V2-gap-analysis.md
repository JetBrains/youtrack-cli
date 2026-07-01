# RFC v2 Gap Analysis

> Compared against `RFC-V2.md` — documents what is specified but not yet implemented.

---

## 1. Global Flags (§3) — Partially wrong

| RFC Flag | Current State |
|---|---|
| `--json` / `-j` | Missing — exists as `--format json` instead |
| `--plain` | Missing entirely |
| `--yes` / `-y` (global skip-confirm) | Missing — each command has its own `--force` |
| `--server` / `-s` (named profile) | Missing entirely |
| `--fields` / `-f` (field projection) | Missing entirely |
| TTY auto-detection → JSON on pipe | Missing — JSON is always default |

---

## 2. Output Format (§4) — Fundamentally backwards

- **RFC**: tables by default, `--json` for machines
- **Current**: JSON by default, `--format table` to get tables
- `--plain` (tab-separated, no headers) is missing
- Structured JSON error format (`{ "error": "...", "code": N, "suggestion": "..." }`) not fully implemented

---

## 3. Authentication & Multi-Server (§5) — Single-server only

| RFC Command | Current State |
|---|---|
| `auth login` (interactive + non-interactive) | Exists as `auth get-token` (different name, saves to `.youtrack-cli-env`) |
| `auth list` | Missing |
| `auth switch <name>` | Missing |
| Named profiles in `~/.config/youtrack-cli/config.json` | Missing — only `~/.youtrack-cli-env` |
| `--server` / `-s` flag to select profile | Missing |
| Env var precedence: env > `--server` > `defaultServer` | Missing |
| Auto-migration from old `.youtrack-cli-env` | Missing |

---

## 4. Command Naming — Wrong verbs throughout (§1, Design Principle 5)

RFC requires `list`, `view`, `create`, `update`, `delete`. Current uses `get`, `add`:

| Current | RFC Target |
|---|---|
| `issue get` | `issue view` |
| `issue add` | `issue create` |
| `user current` | `user me` |
| `user get` | `user view` |
| `user add` | `user create` |
| `project get` | `project view` |
| `project add` | `project create` |
| `group get` | `group view` |
| `board get` | `board view` |
| `board add` | `board create` |
| `dashboard add` | `dashboard create` |
| Flag `--asUser` | `--as-user` (kebab-case) |

---

## 5. Missing Issue Subcommands (§6.1)

- `issue draft create` — draft-safe staging flow for AI agents
- `issue suggest <ids> <query>` — command autocomplete
- `issue assign <id> <user>` / `issue unassign <id>`
- `issue watch <id>` / `issue unwatch <id>`
- `issue vote <id>` / `issue unvote <id>`
- `issue create` missing flags: `--description-file`, `--custom-fields`, `--custom-fields-file`, `--parent`
- `issue view` missing `--comments <n>` inline flag
- `issue delete` uses `--force` instead of `--yes`

---

## 6. Missing Top-Level Command Groups

| RFC Group | Current State |
|---|---|
| `link` (§6.2) | Nested under `issue link` — RFC wants it top-level with `link types` and `link create` |
| `comment` (§6.3) | Nested under `issue comment` — RFC wants top-level with `--on issue\|article` |
| `attachment` (§6.5) | Nested under `issue attachment` — RFC wants top-level with `--on` |
| `workitem` (§6.14) | Nested under `issue workitem` — RFC wants top-level with `types`, `attributes`, `settings`, `add-type`, `set-hours` |
| `server info` (§6.19) | Completely missing |
| `gantt` (§6.18) | Partial: `report gantt-list` / `report gantt-delete` — not a proper `gantt` group |

---

## 7. Missing Project Subcommands (§6.6)

- `project schema <key>` — **key AI agent feature**, exposes field schema for safe issue creation
- `project workitem-attributes <key>`
- `project update <key>` — general settings update
- `project convert-to-template <id>`
- `project app list/add/remove`
- `project team users`, `project team groups`, `project team remove-user`, `project team remove-group` — only `add-user`/`add-group` exist

---

## 8. Missing Safety & Read-Only Mode (§9)

- `YOUTRACK_RO=1` — read-only mode blocking all write commands is not implemented
- Global `--yes`/`-y` flag — currently each command rolls its own `--force`

---

## 9. Configuration File System (§10)

The entire `~/.config/youtrack-cli/config.json` system is missing:

- `defaults` section (`limit`, `project`)
- `aliases` section + `youtrack-cli alias set ...` command
- XDG `$XDG_CONFIG_HOME` compliance
- Named server profiles

---

## 10. Other Missing Features

- `user setting <key> <value>` / `user delete-drafts`
- `comment pin` / `comment unpin` / `comment visibility`
- `comment reaction add` / `comment reaction delete`
- `board search`, `board sprint add` (flags: `--name`, `--start`, `--end`, `--goal`), `board issue add/remove`
- `role grant` / `role permissions`
- `tag apply` and `tag remove` at top level with `--on issue|article`
- v1 backward-compatibility hidden aliases with deprecation warnings (§13)

---

## Priority Order

| Priority | Area | Notes |
|---|---|---|
| 1 | **Output system** | Invert defaults (table by default, JSON on pipe/`--json`), add `--plain` |
| 2 | **Multi-server auth** | Config.json profiles, `auth login`/`auth list`/`auth switch`, `--server` flag |
| 3 | **Verb renames** | `get`→`view`, `add`→`create`, `current`→`me`; `--force`→`--yes` |
| 4 | **Top-level group promotions** | `link`, `comment`, `attachment`, `workitem`, `server` |
| 5 | **`project schema`** | AI agent integration cornerstone |
| 6 | **Safety** | `YOUTRACK_RO=1`, global `--yes`, `--fields` |
| 7 | **Config system** | Defaults, aliases, XDG compliance |
