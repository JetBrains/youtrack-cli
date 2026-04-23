# Multi-Instance Support Design

**Date:** 2026-04-23
**Branch:** ev/multi-instance-support
**Status:** Approved

## Overview

Add support for managing and switching between multiple YouTrack instances within a single CLI installation. Users can authenticate against several instances and switch the active context with a single command — similar to `kubectl config use-context`. All existing commands continue to work without modification.

---

## Goals

- Allow users to store credentials for multiple YouTrack instances.
- Provide a frictionless context-switching UX (`youtrack instance use <alias>`).
- Preserve full backward compatibility: `~/.youtrack-cli-env` continues to work as a fallback.
- Derive a sensible alias from the instance URL automatically; allow override.

## Non-Goals

- Per-command `--instance` flag (active context is the mechanism).
- Auto-migration of `~/.youtrack-cli-env` (manual opt-in only).
- Credential encryption at rest (same posture as the existing env file).

---

## Config Schema

**File:** `~/.youtrack-cli-config.json` (permissions: `0600`)

```json
{
  "currentInstance": "work",
  "instances": {
    "work": {
      "url": "https://work.youtrack.cloud",
      "token": "perm:abc123"
    },
    "personal": {
      "url": "https://personal.youtrack.cloud",
      "token": "perm:xyz789"
    }
  }
}
```

- `instances` is a **map keyed by alias** — O(1) lookup, clean JSON serialization.
- `currentInstance` is a string pointer to a key in `instances`. Empty string means no active instance (env var fallback applies).
- File is written atomically: write to a `.tmp` sibling, then `os.Rename` to final path.

### Alias Derivation

If no `--alias` is provided during `auth get-token`, the CLI derives the alias from the first label of the URL hostname:

```
https://work.youtrack.cloud  →  work
https://mycompany.myjetbrains.com  →  mycompany
```

On collision with an existing alias, append a numeric suffix: `work` → `work-2` → `work-3`. A notice is printed when this occurs.

---

## New Package: `internal/config`

Owns all config file I/O. Exposes a minimal, focused API:

```go
type Config struct {
    CurrentInstance string              `json:"currentInstance"`
    Instances       map[string]Instance `json:"instances"`
}

type Instance struct {
    URL   string `json:"url"`
    Token string `json:"token"`
}

// Load reads ~/.youtrack-cli-config.json.
// Returns an empty Config (not an error) if the file does not exist.
func Load() (*Config, error)

// Save atomically writes the config to ~/.youtrack-cli-config.json at 0600.
func (c *Config) Save() error

// ActiveInstance returns the active instance and its alias, or nil/"" if none is set.
func (c *Config) ActiveInstance() (*Instance, string)

// AddInstance upserts an instance. Derives alias from URL if alias is empty.
// Returns the final alias used (may differ from input on collision).
func (c *Config) AddInstance(alias, url, token string) string

// RemoveInstance removes the instance by alias.
// Returns an error if the alias does not exist.
func (c *Config) RemoveInstance(alias string) error
```

---

## `GetClient()` Resolution Chain

Priority, highest to lowest:

1. `--base-url` / `--token` flags — explicit per-command override, always wins.
2. Active config instance — `config.Load()` → `ActiveInstance()`.
3. `YOUTRACK_BASE_URL` / `YOUTRACK_TOKEN` env vars — legacy single-instance fallback.
4. **Error** with actionable message.

Error messages by scenario:
- No config, no env vars: `"no instance configured — run: youtrack auth get-token"`
- Config exists but `currentInstance` is empty: `"no active instance — run: youtrack instance use <alias>"` (lists known aliases)
- `currentInstance` points to missing alias: `"active instance \"work\" not found in config — run: youtrack instance use <alias>"`

---

## Command Changes

### `auth get-token` (modified)

- Gains `--alias` flag (optional). If omitted, alias is derived from the URL hostname.
- When `--save` is true (default) and the CLI decides to write credentials:
  1. If `~/.youtrack-cli-env` exists with credentials, offer a one-time import prompt (see Migration below).
  2. Save the new instance to the JSON config under the resolved alias.
  3. If this is the first instance in the config, set it as `currentInstance` automatically.
- Success output:
  ```
  ✓ Authenticated as Jane Doe (jane.doe)
    Saved as: work  (~/.youtrack-cli-config.json)
    Run: youtrack instance use work
  ```
  (The "Run" hint is omitted if the new instance was auto-set as active.)

### `auth status` (modified)

- When the JSON config exists, shows all configured instances with their auth status and marks the active one:
  ```
    work     https://work.youtrack.cloud      ✓ Jane Doe (jane.doe)   [active]
    personal https://personal.youtrack.cloud  ✓ Bob Smith (bob)
  ```
- Falls back to single-instance display (existing behaviour) when no config exists.

### `auth logout` (modified)

- Targets the **active** instance by default: removes it from the config and clears `currentInstance`.
- Gains `--instance <alias>` flag to target a specific non-active instance.
- If only env vars are configured (no JSON config), behaves exactly as today.

### New: `instance` subcommand group (`cmd/instance/`)

| Command | Description |
|---|---|
| `instance list` | Same output as `auth status` — lists all instances and marks active |
| `instance use <alias>` | Sets `currentInstance`; prints: `Switched to instance "work"` |
| `instance remove <alias>` | Removes instance; clears `currentInstance` if it was the active one |

`instance add` does **not** exist — that role belongs to `auth get-token`.

---

## Migration: `~/.youtrack-cli-env` Coexistence

The old env file is **never auto-deleted or auto-migrated**.

### Running `auth get-token --save` when env file exists with credentials

The CLI detects credentials in `~/.youtrack-cli-env` and prompts once:

```
Found existing credentials in ~/.youtrack-cli-env
  https://old.youtrack.cloud (old)

Import them as a named instance before adding the new one? [Y/n]:
```

- **Yes:** imports old credentials as a named instance (alias derived from URL), saves to JSON config, then saves the new instance. Env file is left untouched. A follow-up hint is printed:
  ```
  Tip: ~/.youtrack-cli-env is no longer needed. You can delete it safely.
  ```
- **No:** saves only the new instance to JSON config. Env file remains and continues working as a low-priority fallback (step 3 in the resolution chain).

The env file is **never automatically deleted** regardless of the user's choice.

---

## Error Handling & Edge Cases

| Scenario | Behaviour |
|---|---|
| No config, no env vars | Error: `"no instance configured — run: youtrack auth get-token"` |
| Config exists, `currentInstance` is `""` | Error + list of known aliases + `"run: youtrack instance use <alias>"` |
| `currentInstance` points to missing alias | Error: `"active instance \"work\" not found — run: youtrack instance use <alias>"` |
| Alias collision on `auth get-token` | Auto-suffix: `work` → `work-2`; print notice |
| `instance use` with unknown alias | Error: `"instance \"foo\" not found — run: youtrack instance list"` |
| `instance remove` on the active instance | Clears `currentInstance`; hint: `"Run: youtrack instance use <alias> to set a new one."` |
| `instance remove` last remaining instance | Allowed — config becomes empty; env var fallback applies |
| Env var `YOUTRACK_BASE_URL` + active config both set | Flags win; active config wins over env vars (resolution chain is explicit) |
| Config file corrupted / invalid JSON | `Load()` returns error with path — no silent fallback |
| Concurrent CLI writes | Atomic rename; last writer wins (CLI is single-user, not a daemon) |
| `auth get-token --save` + env file with credentials | One-time import prompt (see Migration section) |

---

## File Layout

```
internal/
  config/
    config.go        # Config struct, Load(), Save(), ActiveInstance(), AddInstance(), RemoveInstance()
  client.go          # modified: updated resolution chain
cmd/
  instance/
    instance.go      # NewCmd(), registers list/use/remove
    list.go
    use.go
    remove.go
  auth/
    get_token.go     # modified: --alias flag, saves to JSON config, import prompt
    status.go        # modified: multi-instance display
    logout.go        # modified: --instance flag, config-aware removal
```

---

## Testing

Following the project's integration-only test policy (`test/` package, real YouTrack instance):

- `TestMultiInstanceAddAndSwitch` — add two instances, verify `instance use` switches active, verify commands hit the correct instance.
- `TestInstanceRemoveActive` — remove active instance, verify subsequent commands return "no active instance" error.
- `TestLegacyEnvFallback` — no config file, only env vars set — verify all commands continue to work.
- `TestEnvFileImportPrompt` — `auth get-token --save` with existing env file: verify import prompt, verify both instances persisted on "yes", verify only new instance on "no".
- `TestAuthStatusMultiInstance` — verify all instances listed, active marked correctly.
