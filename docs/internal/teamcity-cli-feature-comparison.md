# TeamCity CLI comparison and feature candidates

This document compares YouTrack CLI with
[JetBrains TeamCity CLI](https://github.com/JetBrains/teamcity-cli) and records
feature candidates that are useful for YouTrack CLI. It is a planning document,
not a compatibility requirement: YouTrack CLI's current commands and output are
the product contract, and TeamCity-specific features should not be copied only
for parity.

The comparison was performed against YouTrack CLI `origin/master` at `76b0226`
and the TeamCity CLI default branch on July 27, 2026.

## Already at parity

YouTrack CLI already has the main foundations expected from an agent-friendly
JetBrains CLI:

- explicit JSON and plain output, field projection, and an authenticated raw API
  escape hatch;
- multiple saved instances, browser authentication, OS-keyring storage, and a
  client-level read-only guard;
- a structured JSON error envelope with initial error codes and suggestions;
- real-server end-to-end coverage and API schema-drift detection;
- GoReleaser builds for Linux, macOS, and Windows on amd64 and arm64 with
  `CGO_ENABLED=0`, injected-version verification, checksums, signing hooks,
  native packages, prerelease detection, and release installation instructions;
- an installable agent skill and documentation for agent-safe operation.

Consequently, release packaging does not need to be redesigned to match
TeamCity CLI. The remaining release work is verification of the produced
artifacts and publication policy, as tracked in
[the EAP checklist](eap-release-checklist.md).

## Recommended before a broad EAP

### 1. Consistent pagination for every collection

**Problem.** Collection requests are bounded at `$top=500`, and `issue list`
supports `--offset`, but many high-level list commands still return only one
page. A successful command can therefore represent a truncated data set without
the caller knowing it.

**TeamCity precedent.** TeamCity CLI centralizes pagination and follows the
server-provided `nextHref` while retaining an explicit truncation signal:
[api/paginate.go](https://github.com/JetBrains/teamcity-cli/blob/main/api/paginate.go).

**Proposed YouTrack contract.**

- Give collection commands consistent `--limit` and `--offset` flags.
- Add `--all` where fetching every page is safe and useful.
- Implement one `$top`/`$skip` paginator in the API layer rather than bespoke
  loops in commands.
- Keep JSON collections as arrays, including `[]` for no results.
- Make a bounded result explicit. Prefer a reliable `hasMore`/truncation signal;
  otherwise document the page semantics and warn on stderr when another page is
  known to exist.
- Preserve the current JSON array shape unless a separately versioned metadata
  envelope is introduced.

**Acceptance criteria.** Tests cover an empty collection, exactly one page, a
second page, `--offset`, `--all`, cancellation between pages, and stable JSON.

### 2. Bounded retries for idempotent reads

**Problem.** Short network interruptions, rate limiting, and transient server
errors currently fail immediately even when repeating a read is safe.

**TeamCity precedent.** TeamCity CLI retries transient GET failures with
context-aware backoff and honors `Retry-After`:
[api/retry.go](https://github.com/JetBrains/teamcity-cli/blob/main/api/retry.go).

**Proposed YouTrack contract.**

- Retry only idempotent reads by default.
- Retry selected network failures, HTTP 429, and explicitly selected 5xx
  statuses with a small attempt and elapsed-time budget.
- Honor `Retry-After`, add jitter, and stop immediately when the command context
  is canceled.
- Do not retry POST, PUT, DELETE, uploads, or other mutations without a proven
  idempotency mechanism.
- Expose the final response through the normal structured error contract.

**Acceptance criteria.** Deterministic tests cover success after retry,
`Retry-After`, exhausted attempts, cancellation during backoff, and the absence
of automatic mutation retries.

### 3. A global `--no-input` mode

**Problem.** CI jobs and agents need a guarantee that no command will wait for a
prompt or interactive authentication flow. `--force` skips individual
destructive confirmations, but it is not a global non-interactive contract.

**TeamCity precedent.** TeamCity CLI exposes `--no-input` at the root and tests
it throughout its black-box suite.

**Proposed YouTrack contract.**

- Add a persistent `--no-input` flag.
- Fail with a classified, actionable error whenever input would be required.
- Keep `--force` as the separate authorization to bypass destructive
  confirmation; `--no-input` alone must not authorize a mutation.
- Document CI behavior and cover browser auth, token prompts, selections, and
  destructive confirmations.

### 4. Acceptance tests for release artifacts

**Problem.** Real-server E2E validates the product surface, and GoReleaser checks
the injected version, but the final archives are not yet smoke-tested across all
release operating systems.

**TeamCity precedent.** TeamCity CLI can run a narrow black-box testscript suite
against the exact GoReleaser-built binary:
[acceptance/README.md](https://github.com/JetBrains/teamcity-cli/blob/main/acceptance/README.md).

**Proposed YouTrack scope.** Retain the real-server E2E suite as the source of
behavioral coverage. Add a small portable artifact suite for `--version`, help,
JSON error shape, `--no-input`, isolated config paths, SIGINT cancellation, and
execution after archive extraction. Run it against Linux, macOS, and Windows
artifacts before publication.

### 5. Complete the structured error taxonomy

YouTrack CLI already emits stable JSON codes for common HTTP errors. Extend the
contract with at least `rate_limited`, `timeout`, `network_error`,
`validation_error`, `conflict`, `read_only`, and `canceled`. Commands and agents
should branch on `error.code`, never on message text. Preserve the existing
envelope and add optional structured details only compatibly.

TeamCity reference:
[api/errors.go](https://github.com/JetBrains/teamcity-cli/blob/main/api/errors.go).

## Candidates during EAP

### Open the resource in YouTrack

Add `--web` to suitable view commands, initially issues, articles, projects,
boards, and dashboards. It should print the resolved URL and open it through the
platform browser. Define its behavior in JSON/non-interactive environments and
make incompatible output flags explicit.

TeamCity implements this through shared command helpers and supports it across
resource commands.

### Bind a repository to a YouTrack project

A committed `youtrack.toml` could map a repository or monorepo subdirectory to a
default project, allowing commands such as:

```text
youtrack issue create --summary "Fix checkout"
youtrack issue list --query "#Unresolved"
```

The file may provide resource context, but it must not contain secrets or cause
credentials to be selected from an untrusted checkout. Server/profile selection
needs a separate explicit trust design. Commands must show the inferred project
in destructive or ambiguous flows and allow an explicit flag to override it.

TeamCity reference:
[internal/link/link.go](https://github.com/JetBrains/teamcity-cli/blob/main/internal/link/link.go).

### Contextual shell completion

Enhance Cobra completion with local instance aliases, project keys known from a
trusted repository binding, and static enums such as link or field types. Avoid
slow network lookups on every shell completion by default, and bound any opted-in
remote lookup with a short timeout.

TeamCity reference:
[internal/completion/completion.go](https://github.com/JetBrains/teamcity-cli/blob/main/internal/completion/completion.go).

### Configuration portability

Adopt `$XDG_CONFIG_HOME` with a documented one-time migration from
`~/.youtrack-cli-config.json`, add `--no-color`/`NO_COLOR`, and bound keyring
operations. Migration must preserve mode `0600`, remain atomic, and never print
tokens.

### Agent-skill evaluation

Keep one authoritative YouTrack skill and add executable evaluations for common
issue, project, membership, read-only, and error-recovery workflows. The goal is
to verify that documented commands execute and their JSON can be parsed, not to
create another mocked API contract.

## Optional or intentionally deferred

- Command-only aliases may help repeated workflows, but shell aliases that run
  arbitrary `!` commands increase the attack surface for an agent-first CLI.
- Custom request headers should be demand-driven. If added, reject control
  characters and redact authorization, cookies, and all configured custom
  headers from diagnostics.
- Telemetry should not be introduced for feature parity. Any future proposal
  requires disclosure, opt-out, a data-minimization schema, and privacy review.
- TeamCity guest authentication, pipeline migration, and build-agent operations
  are product-specific and do not translate into YouTrack CLI requirements.

## Suggested implementation order

1. Consistent pagination and truncation semantics.
2. Idempotent-read retry policy.
3. Global `--no-input`.
4. Release-artifact acceptance tests.
5. Expanded structured errors.
6. `--web`.
7. Trusted repository-to-project binding.
8. XDG configuration, completion, and skill evaluations.

Each feature should be implemented independently with its own compatibility and
security review. The EAP should be used to validate the proposed contracts
before they become stable CLI guarantees.
