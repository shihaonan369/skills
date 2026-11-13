# Authorized Implementation: DAG Link Focus and Change Echo

## Scope

After confirming the recommended rules, the user instructed (translated), "No, start work." Implement the two selected directions in the native Go TUI; introduce no new runtime, change no telemetry/schema, and leave application configuration and real sessions untouched. Do not implement panel zoom, timelines, or event waterfalls.

## Reversible implementation defaults (not yet human-accepted terminal UX)

- Keep arrow keys for DAG panning/scrolling; [ / ] select nodes in declaration order without automatic panning; c clears selection.
- f cycles through direct neighbors/all upstream/all downstream/combined; default to direct neighbors. r toggles return-edge participation (enabled by default; declared edges are not removed).
- i toggles the relationship sidebar inside the leaf panel: side by side when wide enough, or an explicit narrow-width notice without rearranging the caller's layout. Selection and sidebar visibility are viewer-local state.
- x clears local change indicators. Reserve two leading characters per row for transient/static markers so effects never change layout dimensions.
- Compare actual snapshot fields for row differences; use 1.2 seconds of transient emphasis and retain the summary until the next batch of real field changes, a new baseline, or explicit clearing.
- Pause discards in-flight snapshots; resume immediately requests a new baseline without replay. Stale in-flight snapshots must not overwrite the resumed display.
- History/help views suppress emphasis; event history preserves the reading anchor. Clear local node selection when a new layout removes the selected node.

## Implementation and validation

Implemented in native `link_focus.go`, `echo.go`, and the existing DAG/layout/viewer integration. No new dependencies or persisted/schema fields.

- Full Go race/coverage suite: 51 top-level + 15 subtests passed; vet passed. Core coverage 91.3%.
- Nine Mac PTY runs, including keyboard focus/sidebar, real synthetic task row changes, motion-off markers, resume baseline, SIGWINCH and both exit keys.
- Extended parser/viewer fuzz: 5 seconds, 2 workers, 46,387 executions, no failure.
- Six platform archives rebuilt; all 12 binary/archive SHA-256 entries and single-binary package contents verified; extracted native executable starts. Linux/Windows/Intel-Mac are cross-build-only.
- Separate Ghostty-owned TTY opened with the labelled synthetic preview launcher. Temporary state is isolated and cleans on exit; no actual Claude run or settings changes.
- Exact evidence/boundaries are in [VALIDATION.md](../../agent-dashboard/docs/VALIDATION.md). No commit/push/publish, remote host or real-user telemetry mutation.

Terminal-interaction defaults are implemented under the explicit execution authorization but remain available for feedback in the separate acceptance ticket. This does not complete the broader roadmap or authorize remaining controls.
