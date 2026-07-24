# EAP release readiness checklist

The current CLI behavior is the product contract. `RFC-V2.md` is used only as a
coverage checklist; differences in command names, nesting, flags, and output
schema are not defects by themselves.

## P0 — required before EAP

- [x] List commands used by automation cannot silently hide additional pages.
  - [x] No API request uses an unbounded top value; collection reads are capped
    at `$top=500`.
  - [x] `issue list` exposes an offset and documents its limit semantics.
  - [x] `raw` supports following paged collection responses, or documentation states
    an explicit, tested manual pagination contract.
  - [x] Tests cover the second page and zero-result behavior.
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
  - GoReleaser snapshot builds successfully.
  - Release archives are smoke-tested on Linux, macOS, and Windows.

## P1 — strongly recommended during EAP

- [ ] HTTP requests and polling loops propagate `context.Context` from Cobra.
- [ ] `GetUser` preserves non-404 failures instead of reporting “not found”.
- [ ] Project metadata can be updated.
- [ ] Sprints can be updated and deleted.
- [ ] Group membership can be added and removed.
- [ ] Every supported command has JSON contract tests using decoded fields rather
  than substring assertions.

## Release materials

- [ ] Release is marked as EAP/pre-release and follows a documented versioning
  scheme.
- [ ] Supported YouTrack versions, operating systems, and architectures are
  listed.
- [ ] Known limitations and the `raw` escape hatch are documented.
- [ ] Rollback/downgrade instructions are published.
- [ ] Feedback and security-reporting links are present.
- [ ] The background GitHub update request and all opt-outs are disclosed.
