# Maintainer handoff

## Product and scope

Agent Dashboard is a terminal-independent presentation capability for Claude Code.
The caller owns layout, role assignment and workflow; the dashboard supplies
composable Panels and last-observed telemetry. Missing values remain unknown.
There is no workflow executor, automatic agent router or inferred parent graph.

End users receive self-contained platform executables. Go is a build-time
dependency; Python is used only by optional developer tests/demo tooling.
macOS ARM64 has local runtime verification. macOS x64, Linux and Windows targets
remain cross-build-only unless new runtime evidence is recorded.

## Current implementation

- `agent-dashboard/internal/dashboard/layout.go`: caller-owned row/column trees
  with session, agent/agents, tasks, events, text and native DAG leaves.
- `dag.go`: validated declared graphs, deterministic layout, external return
  rails, observed actor/task binding, cell-aware viewports and decorative motion.
- `link_focus.go`: viewer-local directional reachability and optional in-leaf
  relation details. Selection never executes work or edits the caller graph.
- `echo.go`: observed field differences mapped to existing rows, static change
  summaries, and history anchors for bounded newest-first event buffers.
- `tui.go`: independent motion/snapshot clocks, keyboard navigation and
  epoch-tagged reads so pre-pause snapshots cannot overwrite a resumed baseline.
- `state.go` / `ingest.go`: isolated opt-in state, bounded real hook normalization,
  cross-process locking and atomic writes.
- `install.go`: backed-up/idempotent hook configuration and narrow removal that
  preserve unrelated settings and skills.

The selected visual direction is in-place graph focus with optional relation
details, plus local row-change echoes. Panel zoom, timelines and event-waterfall
controls are not implemented.

## Read before changing behavior

- `agent-dashboard/docs/PANELS.md`: composition, bindings, keys, change echoes,
  pause/reopen/history behavior.
- `agent-dashboard/docs/DAG.md`: graph validation, returns/cycles, relationship
  focus, viewport and sidebar rules.
- `agent-dashboard/docs/ARCHITECTURE.md`: telemetry, privacy and installer contract.
- `agent-dashboard/docs/VALIDATION.md`: exact test coverage and runtime boundaries.

Keep actual task/lifecycle data intact. Tool errors do not fail the whole agent;
ordinary Stop/returned states do not complete the issue. Decorative packets are
not measured traffic. Viewer operations do not control agents or collection.

## Development and distribution

The source of truth is this repository; installed skill paths are links, not
development copies. Preserve other skills/hooks. Never commit credentials,
private telemetry, user transcripts or machine-specific account paths.

Build/test from `agent-dashboard/`; use the available Go toolchain, then
`go test -race -cover ./...`, `go vet ./...` and `go run ./cmd/package`.
The optional `tests/pty_smoke.py` is developer-only, not a product dependency.

The packager builds six OS/architecture combinations and complete skill ZIPs.
Publish ZIPs plus `target/release/SHA256SUMS` through GitHub Releases. Keep
`bin/`, `target/` and build toolchains out of Git. Unsigned distribution does not
imply notarization, Windows signing or runtime verification on cross-built targets.

Planning records live under `.scratch/dashboard-control-effects/`.
Throwaway HTML prototypes/screenshots are archived on
`prototype/dashboard-control-effects`, outside the product branch and packages.
The separate native demo launcher uses isolated synthetic temporary state and
cleans it on exit. Historical `agent-dashboard/legacy/rex/` is uninstalled reference
material; do not reinstall it or modify terminal configuration to use this skill.

## Privacy and publishing

Both published branches have sanitized history. Use a maintainer alias and GitHub
noreply identity for new commits, and keep examples independent of machine-account
paths. Preserve third-party notices verbatim and retain the public repository URL
where needed for downloads. Publish only the intended validated branches/tags;
local history-recovery bundles/backup refs stay private and are never mirrored.
Old external clones or caches cannot be assumed erased by a history rewrite.

The distribution tag is `agent-dashboard-v0.1.0`. Upload the six complete skill
ZIPs and `target/release/SHA256SUMS`, verify asset names/hashes in draft, then
publish and run `tests/release_smoke.py` against the public URLs. Record the actual
post-publication result in Release notes. Keep source tag provenance aligned with
the code/docs used to build packages; never claim cross-builds as runtime tests.
