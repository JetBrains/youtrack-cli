# CLI Research Findings

> Live-instance research against local Docker YouTrack (http://127.0.0.1:8090).
> Date: 2026-06-05. Researcher: sub-agent (automated command sweep).

---

## Summary

17 findings across all command groups. 3 critical (broken API calls), 1 high (silent config override), 6 medium/low UX gaps, 6 documentation gaps, 1 security fix already applied.

| Severity | Count |
|---|---|
| Critical (broken API) | 3 |
| High (UX regression) | 1 |
| Medium | 3 |
| Low | 6 |
| Doc gap | 4 |
| Fixed | 1 |

---

## Critical — Broken API Calls

### F-01: `issue link add` returns HTTP 405

**Command:** `youtrack issue link add <issueId> --target <targetId> --type <typeId>`

**Symptom:** HTTP 405 Method Not Allowed from YouTrack.

**Root cause:** The CLI POSTs to the link type collection endpoint (read-only). YouTrack's REST API for linking issues requires a POST to `/api/issues/{id}/links` with a body specifying the link direction and target issue, not a POST to the link type list.

**Fix:** Rewrite `api/issues.go` `AddIssueLink` to POST to `/api/issues/{id}/links` with body `{"direction": "...", "issues": [{"id": "..."}]}`. The link type should be looked up first to resolve the direction field.

---

### F-02: `field bundle` commands return HTTP 404

**Commands:** Any `youtrack field bundle ...` subcommand.

**Symptom:** HTTP 404 on all bundle endpoints.

**Root cause:** The API path uses the Go struct type name `EnumBundle` (PascalCase) where the YouTrack REST API expects lowercase bundle type strings: `enum`, `state`, `version`, `build`, `ownedField`, `user`.

**Fix:** Change bundle type path segments from PascalCase (`EnumBundle`) to lowercase (`enum`) in `api/paths.go` or wherever the bundle path format strings are defined.

---

### F-03: `user locale set` returns HTTP 500

**Command:** `youtrack user locale set <userId> <locale>`

**Symptom:** HTTP 500 Internal Server Error.

**Root cause:** `api/users.go:SetUserLocale` sends `{"locale": {"locale": "en_US"}}`. The YouTrack API for locale expects `{"locale": {"id": "en_US"}}` — the nested key should be `"id"`, not `"locale"`.

**Fix location:** `api/users.go:123` — change `map[string]any{"locale": locale}` to `map[string]any{"id": locale}`.

---

## High — UX Regression

### F-04: Env vars silently overridden by active config instance

**Symptom:** Setting `YOUTRACK_BASE_URL` / `YOUTRACK_TOKEN` has no effect when `~/.youtrack-cli-config.json` has an active instance. No warning is printed.

**Root cause:** Client resolution priority (highest to lowest): `--base-url`/`--token` flags → `--server` flag → active instance in config file → env vars. The config file takes Tier 3, env vars are Tier 4. This is correct behavior but not communicated.

**Fix:** Print a warning to stderr when env vars are present but overridden by an active config instance. Example: `"YOUTRACK_BASE_URL is set but ignored; active instance 'work' takes precedence. Use --server or 'instance use' to change."`.

**Fix location:** `internal/client.go` — in `GetClient()`, detect when both the env var and config instance are populated.

---

## Medium

### F-05: `issue count` returns -1 on first call

**Command:** `youtrack issue count --query "..."`

**Symptom:** Returns `-1` immediately — YouTrack computes issue counts asynchronously. The first poll often returns the sentinel value before the count is ready.

**Fix:** Add a retry loop in `api/issues.go` `CountIssues()` that polls until the result is ≥ 0 or a timeout is hit (e.g. 5 s, 200 ms interval).

---

### F-06: `user pin-project` requires opaque user ID, not login

**Command:** `youtrack user pin-project <userId> <projectId>`

**Symptom:** Must pass opaque `userId` (e.g. `1-1`), not login. Inconsistent with all other `user` commands that accept login.

**Fix:** Resolve `userId` via `GetUser()` in `api/users.go:PinProjectForUser()` (same pattern as `DeleteUser`, `BanUser`).

---

### F-07: `issue link remove` requires opaque `linkId`

**Command:** `youtrack issue link remove <issueId> <linkId>`

**Symptom:** `linkId` is an opaque ID (e.g. `100-1`) returned by `issue link list --json`. Not documented anywhere, and the table output of `issue link list` does not show it.

**Fix:** Ensure `issue link list` table output includes the link ID column, and document the two-step workflow in help text: `issue link list --json | jq '.[].id'` → `issue link remove <id> <linkId>`.

---

## Low

### F-08: Missing success output on delete commands

**Commands:** `youtrack issue tag`, `youtrack issue visibility`, `youtrack dashboard delete`, `youtrack canvas delete`, `youtrack search delete` (if implemented).

**Symptom:** No output after successful execution — user cannot tell if the command worked.

**Fix:** Print a single-line confirmation like `"deleted tag <id>"` / `"updated visibility on <issueId>"` to stdout on success.

---

### F-09: `user delete` / `group delete` — `--successor` not marked required

**Command:** `youtrack user delete <userId>` (missing `--successor`)

**Symptom:** Help text shows `--successor` as optional. YouTrack requires a successor for user deletion; without it the API returns an error. The flag should be marked `Required: true` in cobra, so the CLI rejects the call before making any HTTP request.

**Fix:** Add `cmd.MarkFlagRequired("successor")` in `cmd/user/delete.go`.

---

### F-10: `issue comment visibility` / `issue visibility` output unclear

**Commands:** `youtrack issue comment visibility <issueId> <commentId>`, `youtrack issue visibility <issueId>`

**Symptom:** No output on success; error output on failure may be opaque.

**Fix:** Print `"set visibility on <issueId>"` on success.

---

### F-11: `issue workitem add` — duration format undocumented

**Command:** `youtrack issue workitem add <issueId> --duration <value>`

**Symptom:** `--duration` accepts ISO-8601 durations (`PT1H30M`, `P1DT2H`) but this is not mentioned in help text or README.

**Fix:** Update the flag's `Usage` string in `cmd/issue/workitem.go` and add an example in README.

---

### F-12: `project archive` — no success output

**Command:** `youtrack project archive <shortName>`

**Symptom:** Exits 0 silently. No confirmation printed.

**Fix:** Print `"archived project <shortName>"` on success.

---

## Documentation Gaps

### F-13: `user profile` subcommands — complex JSON body, no examples

**Commands:** `youtrack user profile general|appearance|articles|notifications|issues-list|grazie|tips <userId> [flags]`

**Issue:** Flags accept JSON body blobs with no schema in help text or docs. Agents cannot use these without examples.

**Fix:** Add examples for each profile subcommand in `SKILL.md`.

---

### F-14: `project timetrack set` — flags undocumented

**Command:** `youtrack project timetrack set <shortName> [flags]`

**Issue:** Unclear what flags are accepted and what each does.

**Fix:** Document available flags in README and SKILL.md.

---

### F-15: `issue assist` — functionality unclear

**Command:** `youtrack issue assist <query> <issueId>...`

**Issue:** Help text does not explain what "assist" does vs `suggest`. Based on code, it applies YouTrack AI-assisted command suggestions.

**Fix:** Clarify help text and add an example.

---

### F-16: Env var precedence undocumented

**Issue:** The 4-tier precedence (flags > `--server` > config instance > env vars) is not described in README or SKILL.md. Users assume env vars always apply.

**Fix:** Add a "Credential Precedence" section to README.

---

## Fixed

### F-17: `user delete` — self-deletion guard (fixed in 490bb62)

**Command:** `youtrack user delete me --successor <other>`

**Previous behavior:** `GetUser("me")` resolves via `/api/users/me` to the currently authenticated user. Calling `user delete me` would silently delete the admin running the command.

**Fix applied:** `api/users.go:DeleteUser()` now calls `GetCurrentUser()` after resolving the target user and rejects if `user.ID == me.ID`, regardless of what identifier was passed. Error: `"cannot delete the currently authenticated user '<login>'"`.
