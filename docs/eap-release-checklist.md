# EAP release readiness checklist

The current CLI behavior is the product contract. Differences from historical
design proposals in command names, nesting, flags, and output schema are not
defects by themselves.

Current top-level coverage: **12 of 18 gates complete (67%)** — P0 4/6, P1 6/6,
and release materials 2/6. This document remains the release gate until the open
verification and publication tasks below are completed. Longer-term feature
candidates discovered by comparison with TeamCity CLI are tracked separately in
[TeamCity CLI comparison and feature candidates](teamcity-cli-feature-comparison.md).

## P0 — required before EAP

- [ ] List commands used by automation cannot silently hide additional pages.
  - [x] No API request uses an unbounded top value; collection reads are capped
    at `$top=500`.
  - [x] `issue list` exposes an offset and documents its limit semantics.
  - [x] `raw` supports following paged collection responses, or documentation states
    an explicit, tested manual pagination contract.
  - [x] Tests cover the second page and zero-result behavior.
  - [ ] Other high-level collection commands either expose consistent pagination
    or explicitly document and signal their bounded single-page result. Capping
    requests at 500 prevents unbounded reads but does not by itself prevent
    silent truncation.
- [x] Authentication diagnostics use the same credential precedence as normal
  commands.
  - Explicit flags > `--server` > environment > active instance.
  - `auth status` has tests for config + environment and partial overrides.
- [x] Machine-readable output never changes format implicitly.
  - `--json` always emits valid JSON.
  - `--plain` emits tab-separated/plain output or a clear non-zero error; it
    never falls back to JSON.
  - Mutating commands do not write human-only success text in JSON mode.
- [x] Core resource lifecycles are reversible without `raw`.
  - Article attachments can be deleted.
  - Work items can be listed and deleted.
  - Tags can be removed from issues and articles.
  - Project team users/groups can be listed and removed.
  - Project custom fields can be detached.
- [x] Self-update cannot hang indefinitely or execute an unbounded response.
  - External upgrade and install-method commands have deadlines.
  - Downloaded install scripts have an explicit size limit and are fetched from
    the selected release rather than a moving `latest` target.
  - Update behavior and opt-out variables are documented.
- [ ] Release verification passes on the release commit.
  - [x] `make fmt-check`, lint, unit tests, build, vulnerability scan, and full E2E.
  - [ ] GoReleaser snapshot builds successfully from the release commit.
  - [ ] Release archives are smoke-tested on Linux, macOS, and Windows, including
    `--version`, help, JSON errors, and execution after extraction.

## P1 — strongly recommended during EAP

- [x] HTTP requests and polling loops propagate `context.Context` from Cobra.
- [x] `GetUser` preserves non-404 failures instead of reporting “not found”.
- [x] Project metadata can be updated.
- [x] Sprints can be updated and deleted.
- [x] Group membership can be added and removed.
- [x] Representative JSON contract tests cover the shared output shapes and
  CLI-composed responses. Per-command decoding is intentionally not required
  because serialization is centralized in `output.Print`.
  - [x] Every E2E invocation using `--json`, `-j`, or `--format json` is checked
    for syntactically valid JSON.
  - [x] Object, list, mutation-success, lifecycle, and the composed
    `issue view --comments` responses are decoded and checked by representative
    tests rather than raw substring matching.

## Release materials

- [ ] Release is marked as EAP/pre-release and follows a documented versioning
  scheme. GoReleaser detects prerelease tags, but the EAP tag/version policy is
  not yet documented.
- [ ] Supported YouTrack versions, operating systems, and architectures are
  listed. Release targets exist for Linux, macOS, and Windows on amd64 and arm64;
  a user-facing compatibility matrix and minimum YouTrack version are missing.
- [ ] Known limitations and the `raw` escape hatch are documented. The `raw`
  escape hatch is documented; publish the EAP known-limitations list.
- [ ] Rollback/downgrade instructions are published, including package-manager
  and direct-download installs.
- [x] Feedback and security-reporting links are present in `CONTRIBUTING.md` and
  `.github/SECURITY.md`.
- [x] The background GitHub update request and all opt-outs are disclosed in the
  configuration documentation.
