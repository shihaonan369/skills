---
name: agent-dashboard
description: "Compose a live terminal TUI when a user or another skill requests a dashboard, agent/task panels, or a live status board. Supplies panels and telemetry; the caller controls layout and keeps its existing workflow."
argument-hint: "[caller layout / task context]"
---

# Agent Dashboard · presentation capability

Keep the caller's existing workflow and skills. This skill supplies TUI panels,
not a workflow, orchestrator, agent router, plan template or completion policy.
The screenshot is visual inspiration only. AI Hero is an optional example, not a
default or required dependency. Do not spawn agents just to populate a panel.

## Locate the executable and check readiness

The skill directory is `${CLAUDE_SKILL_DIR}`. Select its bundled binary using the
**host running Claude**, not a remote viewer's OS:

| Host | Relative executable |
| --- | --- |
| macOS arm64 / x86-64 | `bin/darwin-arm64/agent-dashboard` / `bin/darwin-amd64/agent-dashboard` |
| Linux arm64 / x86-64 | `bin/linux-arm64/agent-dashboard` / `bin/linux-amd64/agent-dashboard` |
| Native Windows arm64 / x64 | `bin/windows-arm64/agent-dashboard.exe` / `bin/windows-amd64/agent-dashboard.exe` |

In commands below, replace `<D>` with that quoted absolute executable path;
PowerShell uses `& '<path>'`. Run `<D> doctor`. If the binary is absent, report a
missing platform package and direct the user to the ready-to-use ZIPs in
[GitHub Releases](https://github.com/shihaonan369/skills/releases/latest), as
explained in `README.md` → Download. A source checkout alone is not a runtime
package; do not install an end-user runtime or revive the legacy implementation. If hooks are not configured, explain the one-time
`<D> install --skill-dir "${CLAUDE_SKILL_DIR}"` step, get permission before changing
settings, and request a fresh Claude session after installation. See `README.md`
for Windows linking/permissions and platform verification constraints.

## Attach and publish the caller's composition

1. Run `<D> start --session "${CLAUDE_SESSION_ID}" --title "<actual work title>"`.
   Use the substituted **Claude session ID**, not a terminal ID or guessed UUID.
   The result prints the command for a second terminal on this host. Show it to
   the user; do not claim a window opened without an actual terminal operation.
2. Read `docs/PANELS.md` to construct the caller's layout. Accept an existing
   layout file, or compose panels according to the **calling skill's instructions
   and current real work**. The caller chooses rectangles, buffers, sizes, data
   bindings and responsive rules. Do not infer a workflow from tools/screens.
   If no composition/context is provided, use the explicit session inspector or
   propose a minimal presentation, clearly not an inferred workflow.
3. Store any live composition in a private working file outside git, then run
   `<D> layout --session "${CLAUDE_SESSION_ID}" --json @<absolute-layout-file>`.
   It atomically updates the canvas. The caller can replace it at any time.
   For an AI Hero example **only when relevant**, read `examples/ai-hero/README.md`.
   Its live templates combine native DAG/detail Panels and contain binding
   placeholders; `.demo.layout.json` files are separate synthetic fixtures.
   These are caller illustrations, not prescribed phases.
4. After actual Claude tool activity, inspect `<D> status --session
   "${CLAUDE_SESSION_ID}"`. A positive/increasing `hook_count` confirms observed
   events. Configuration alone does not prove hooks are loaded/permitted. If
   zero, state telemetry is unverified and follow `docs/ARCHITECTURE.md`; do not
   represent missing data as idle/success or use screen/process proxies.

## Supply data without taking over work

Native hooks collect real lifecycle/tools and recognized successful native task
updates. An agent panel binds a real ID from observed data; labels do not create
or classify agents. Missing actors, models, parentage and progress stay unknown.
Panel selection/color has no workflow meaning. Text panels are labelled
caller-authored and are not telemetry.

Only if the existing workflow needs them, use these optional adapters:

- `<D> plan --session "${CLAUDE_SESSION_ID}" --json @<reported-plan-file>` mirrors
  its **actual** task metadata; it does not generate or require plan stages.
- `<D> note --session "${CLAUDE_SESSION_ID}" --text "<concise factual note>"`
  records an operational annotation, not hidden reasoning, secrets or source.
- `<D> finish --session "${CLAUDE_SESSION_ID}" --summary "<caller declaration>"`
  reports completion when the **caller** says it is complete. This skill defines
  no prerequisite loop. Ordinary Claude Stop is never issue completion.

Continue the caller's task normally; do not insert a new process or force use of
other skills. Closing/pausing the viewer affects rendering only. Collection can
be disabled explicitly with `<D> deactivate --session "${CLAUDE_SESSION_ID}"`.
Use `demo` only for a requested synthetic preview and label it as such; never
mix demo data with the user's active session or claim it proves real execution.

## Native DAG Panel (presentation only)

When the caller requests a graph, read `docs/DAG.md` and add a `kind: dag`
leaf to its existing row/column composition. The Panel's `dag` field contains
the version-1 nodes/edges object. Publish the complete layout through the same
`<D> layout --session "${CLAUDE_SESSION_ID}" --json @<private-layout-file>`
interface, then use the normal `tui` viewer. No Python or separate graph process
is needed.

- The caller explicitly declares nodes, edges and bindings. Real state comes
  only from exact observed actor/task IDs; missing bindings stay unknown and
  unbound nodes show `declared only`. Never infer a workflow or agent parentage.
- Ordinary edges must be acyclic; `style: return` annotations close declared
  loops without affecting ranking. Cross-rank/return edges use external rails.
  The graph never executes or schedules work. For a caller-declared delegation
  and report contract, add separate parent → child and child → parent edges;
  mark the report `style: return`. Each animates in its own direction; never
  infer a reverse edge or message delivery from chronology or a returned state.
- Up/down scroll; left/right pan the focused DAG; Home resets both axes.
  Focus has a heavy highlighted border. Edge packets are decorative, never proof
  of traffic; `a` toggles motion and `p` freezes snapshot/animation, not collection.
- `examples/dag/fanout.layout.json` is a synthetic preview. Optional
  `examples/ai-hero/{tdd,review,review-no-spec}.layout.json` (full workspaces),
  `examples/ai-hero/*.dag.layout.json` (compact) and
  `examples/issue-loop/loop.dag.layout.json` illustrate caller compositions,
  not product defaults. Keep live copies private and replace bindings with
  actual IDs consistently in graph, actor cards and filters. For a requested
  AI Hero synthetic preview, use its separate `.demo.layout.json` files with
  `demo`; never carry their fixture IDs into live layouts.
