---
name: agent-dashboard
description: "Compose a live terminal TUI when a user or another skill requests a dashboard, agent/task panels, or a live status board. Supplies panels and telemetry; the caller controls layout and keeps its existing workflow."
argument-hint: "[caller layout / task context]"
---

# Agent Dashboard · presentation capability

Keep the caller's existing workflow and skills. This skill supplies TUI panels,
not a workflow, orchestrator, agent router, plan template or completion policy.
AI Hero is an optional example, not a default or required dependency. Do not spawn
agents just to populate a panel.

## Ensure readiness (agent-managed)

Resolve the absolute skill directory from this loaded file. Claude can supply
`${CLAUDE_SKILL_DIR}`; other harnesses, including Pi, do not necessarily expand
it. Substitute the real directory in the examples below. Run bootstrap on the
**host running this agent**, not on a host selected only for observation.
Perform routine setup yourself rather than asking for downloads or shell work.

macOS/Linux (including WSL):

```sh
sh "${CLAUDE_SKILL_DIR}/bootstrap.sh"
```

Native Windows, in PowerShell:

```powershell
& (Join-Path '${CLAUDE_SKILL_DIR}' 'bootstrap.ps1')
```

On success, the sole stdout line is the absolute executable path. Use it as
quoted `<D>` below; PowerShell uses `& '<path>'`. The bootstrap reuses an existing
binary without network access. If absent, it selects the native platform,
downloads its pinned release, verifies against bundled `bootstrap.sha256` and
the published manifest, and atomically installs only the binary under `bin/`.
It never edits hooks, settings, skill instructions, or layouts. No end-user
runtime/package installation or build toolchain is needed. Prebundled platform
packages use their included executable only; if it is missing, obtain a complete
matching package rather than downloading a fallback. See `README.md` → First
use and Release pin lifecycle for utility, distribution and upgrade boundaries.

A nonzero bootstrap exit is a blocker: report the concrete network, permissions,
policy, utility, or platform failure and stop setup. Do not hand back manual
download steps, invent a binary path, bypass TLS/hash checks or execution policy,
or fall back to the legacy implementation. The agent may retry transient network
failures; integrity failures require investigation, not a different download URL.

## Choose the observation boundary

Run `<D> capabilities` for optional feature discovery. Published 0.1.0 binaries
predate this command; an unknown command leaves normal Claude mode usable, but
is a blocker for Rex features. Version 0.2.0 advertises the capabilities below.
Bootstrap never upgrades an existing binary; obtain approval before replacing
an older installation, preserve integrity pins and verify the actual executable.

- **Claude lifecycle/tools/tasks:** follow the hook readiness and attachment
  steps below. A Rex terminal is optional, never a collector prerequisite.
- **Rex basic program status, selected remote hosts, or Pi status in Rex:** read
  and follow `docs/REX.md` in full. Require its capability gates, resolve only
  the requested hosts, save private UTF-8 source/layout files, then open and
  verify the viewer yourself. This branch skips Claude hook setup and session
  variables. For cards, work rows, source/session scope, observation history or
  program-bound DAGs, require `observation-bindings-v1` and follow
  `docs/OBSERVATIONS.md`. Use its shared typed bindings; keep program-report
  projections distinct from native agents/tasks. This is not full Pi telemetry.
- **Unsupported telemetry:** keep missing lifecycle/tools/tasks unknown and
  explain the actual boundary. A program report is not a substitute for a
  requested full agent adapter without agreement on the narrower scope.

### Live membership and semantic monitoring scope

For new agents/programs joining an existing canvas, role-specific monitoring, or
any dynamically repeated cards/lists/graphs, read **`docs/DYNAMIC.md` in full**.
Require `dynamic-layout-v1`; semantic membership additionally requires
`agent-scope-v1`. These require version 0.3.0 or later; 0.2.0 assets lack them.
Check the actual executable; source pins follow README.md's two-phase release
lifecycle. Report a missing capability; do not silently upgrade.

The initiating agent interprets the user's scope. Use exact reported metadata
filters only when grounded; semantic labels such as “implement agents” are not
an excuse to guess roles or include every general-purpose agent. Initialize a
private scope mailbox and publish shared selectors. Delegate **both initial and
subsequent semantic decisions to the same native background subagent** where
supported; the main agent supplies scope/context and starts/stops the worker,
not an initial classification pass. For Claude Code, prefer a worker-owned
**Monitor** running `scope-watch`; the worker handles existing candidates, later
events and bounded watch renewal in its own context. Follow DYNAMIC.md's actual
capability checks, worker protocol, budgets and stop procedure. Fall back to
background-worker long polling, then foreground subagent batches; use the main
agent only without usable subagents. Disclose gaps and verify real worker
activity. The dashboard never launches Claude/Pi or calls a model API.

### Claude hook readiness

Run `<D> doctor`. If hooks are missing, explain that Claude settings must change,
get permission, then **run** `<D> install --skill-dir "${CLAUDE_SKILL_DIR}"`
yourself and rerun doctor. Request a fresh Claude session afterward; do not claim
that the current session has loaded hooks. If approval is denied, leave settings
untouched and report live lifecycle telemetry as unavailable. Windows
link/permission constraints are in `README.md`.

## Attach and publish the caller's composition

1. Run `<D> start --session "${CLAUDE_SESSION_ID}" --title "<actual work title>"`.
   Use the substituted **Claude session ID**, not a terminal ID or guessed UUID.
   The result prints a terminal-independent viewer command. After composing the
   layout, open it through an available authorized presentation adapter; verify
   the actual returned pane/window before claiming it opened.
2. Read `docs/PANELS.md` to construct the caller's layout. Accept an existing
   layout file, or compose panels according to the **calling skill's instructions
   and current real work**. The caller chooses rectangles, buffers, sizes, data
   bindings and responsive rules. Do not infer a workflow from tools/screens.
   If no composition/context is provided, use the explicit session inspector or
   propose a minimal presentation, clearly not an inferred workflow.
3. Store any live composition in a private working file outside git, then run
   `<D> layout --session "${CLAUDE_SESSION_ID}" --json @<absolute-layout-file>`.
   It atomically updates the canvas. The caller can replace it at any time.
   If capabilities include `rex-local-open-v1`, run `<D> open --session
   "${CLAUDE_SESSION_ID}"`: a verified Rex context creates/reuses the right pane
   without taking focus; `mode: terminal` means no pane opened and returns the
   normal viewer arguments. Read `docs/REX.md` for launch failures/recovery. Older
   binaries use the printed viewer command through existing terminal controls.
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
interface, then use the normal `tui` viewer. For a Rex source composition, use
`rex-watch` / `open --sources` instead, following `docs/OBSERVATIONS.md` and
`docs/REX.md`; bind full program keys, not Claude actor aliases. No Python or
separate graph process is needed.

- The caller explicitly declares nodes, edges and bindings. Real state comes
  only from exact typed actor/task/program bindings; missing bindings stay unknown and
  unbound nodes show `declared only`. Never infer a workflow or agent parentage.
- Every edge must belong to one forward acyclic graph; styles are `solid` or
  `dashed`. Cross-rank edges use external rails. The graph never executes work
  or draws replies from lifecycle state. Older `style: return` edges are rejected:
  remove them from the caller's private composition before republishing. Preserve
  actual actor/task/program bindings and leave installed/private state untouched.
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
