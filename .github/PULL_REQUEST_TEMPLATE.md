<!-- Keep this lean. Every section stays — write "N/A — <reason>" instead of removing one. The diff carries the details; this body explains what you can't read from the diff. -->

## Summary

<!-- One paragraph: what does this PR do, and why -->

## Changes

<!-- Short bullet list of key changes; group by area if useful; terse -->

-

## Design Decisions

<!-- Non-obvious choices and trade-offs; write "Straightforward." if none -->

## Example

<!-- For CLI-visible changes: show command + output in a ```bash block. Write "N/A — not user-visible." otherwise -->

```bash
youtrack <command>
```

## Test Plan

- [ ] `make fmt-check`, `make lint`, `make build` pass
- [ ] Full Dockerized E2E suite passes in CI (`make test`)
- [ ] If adding a new command/flag: E2E test added in `test/`
- [ ] If adding a data-producing command: includes `--json` support
- [ ] If modifying `--json` output: no field removals/renames (additive only)
- [ ] If changing docs-visible behavior: updated `docs/`, `.agents/skills/`, and `README.md`
