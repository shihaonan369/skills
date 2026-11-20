# Native DAG Panel v1

**Caller owns the graph; the viewer owns rendering and local observation.**
`dag` is a leaf Panel in the bundled Go executable, composable with all other
Panels through `row` / `column`. No Python or other end-user runtime is required.
There is no graph executor, agent router, automatic dependency inference or
fixed workflow. All declared edges flow forward through one directed acyclic
graph; every edge participates in ranking.

## Publish through the existing layout interface

From the skill directory, using the binary for your host:

```sh
D=./bin/darwin-arm64/agent-dashboard
"$D" start --session '<Claude-ID>' --title 'Actual current work'
"$D" layout --session '<Claude-ID>' --json @my-private-layout.json
"$D" tui --session '<Claude-ID>'
```

PowerShell: use `& $D` and quote `'@my-private-layout.json'`. Store live layouts
outside git. Publication atomically replaces the session layout; existing viewers
pick it up at their next refresh. Invalid graphs leave the prior layout intact.

Example layout (the example declares structure, not actual execution):

```json
{
  "version": 1,
  "name": "Caller workspace",
  "root": {
    "kind": "column",
    "children": [
      {
        "id": "topology",
        "kind": "dag",
        "title": "Declared topology",
        "dag": {
          "version": 1,
          "nodes": [
            {"id": "lead", "label": "Lead", "color": "peach",
             "bind": {"actor": "lead"}},
            {"id": "a", "label": "Reported item", "color": "blue",
             "bind": {"task": "1"}},
            {"id": "note", "label": "Caller annotation",
             "meta": ["Not telemetry"]}
          ],
          "edges": [
            {"from": "lead", "to": "a", "label": "caller declares"},
            {"from": "a", "to": "note", "style": "dashed"}
          ]
        }
      },
      {"id": "events", "kind": "events", "size": 6}
    ]
  }
}
```

## Graph schema

- Graph: required `version: 1`, `nodes`; optional `name`, `edges`.
  The surrounding Panel's `title` controls its heading; `name` is graph metadata.
- Node: required unique printable `id`; optional `label` (defaults to ID),
  `color`, `bind`, `meta` (caller-authored lines, not telemetry).
- Binding: exactly one of `{"actor":"<observed-id>"}`,
  `{"task":"<reported-id>"}`, or (0.2.0+ with `observation-bindings-v1`)
  `{"program":"<full-rex-key>"}`. This is the same typed reference used by
  record Panels; see [shared observations](OBSERVATIONS.md). Use `actor: lead`
  for the Claude root session.
  IDs are exact; role names/labels do not classify or create agents.
- Edge: required `from`, `to` referencing declared nodes; optional `label` and
  `style` (`solid` or `dashed`; omitted means solid).
  Duplicate directed endpoint pairs, self-edges and all cycles are rejected.
- Node colors use the Panel palette (see `PANELS.md`). Earlier graph colors
  `orange`, `green`, `teal`, `amber`, `white` are also supported.

Unknown fields, invalid styles/colors/bindings and cycles are
errors. Limits per graph: 1..64 nodes, at most 256 edges; node ID 80 bytes,
label/name/edge label 160 bytes, binding ID 120 bytes; at most 4 meta lines,
2000 input bytes each (sanitized to 400 runes). Terminal controls/bidi overrides
are removed from display text; IDs containing them are rejected. The whole
layout remains subject to its existing 256 KiB and tree limits.

Program-bound nodes display literal Rex report state, not a fabricated actor or
task. `working` activity requires a live source; disconnection retains a static
`LAST KNOWN` state. Program `done` does not add a task-success check or imply
Issue completion. The relationship inspector and change echoes resolve the same
binding as cards/lists. Use `rex-watch --sources @private.json --layout @layout.json`
or `open --sources ...` for a Rex composition; no Claude session ID is required.

## Forward-only graph and older layouts

Declare only the forward structure the caller wants to inspect, for example
parent → worker → findings. The viewer never identifies or draws a reply path
from chronology, record ID hierarchy or a worker's returned state. Nodes keep
their literal observed status independently of the edge structure.

Reverse/return routes, paired-arrow indicators, upward flow animation and the
`r` return-traversal control have been removed in version 0.2.0. A former
`style: return` edge is now an explicit validation error, including when that
style was placed on an otherwise forward edge. Ordinary opposite-edge pairs
and self-edges are invalid cycles as well. Delete those obsolete edges in the
caller's private composition and republish; there is no silent conversion,
edge reversal, layout rewrite or modification of installed/private state.
An older persisted graph shows an `Invalid DAG` diagnostic until replaced.
[Historical validation](VALIDATION.md) describes the capabilities of its dated
builds, not the current graph contract. Public 0.1.0 assets remain unchanged.

The small synthetic `examples/dag/delegation.layout.json` now declares only
parent → child. AI Hero Review examples retain lead → worker delegation and
worker → verdict findings, without worker → lead reporting edges. Workflow
repetition, when relevant, belongs to the caller outside this graph.

## Observation semantics

- Unbound node: `declared only`.
- Bound but absent actor/task: `unobserved · unknown`.
- Bound actor/task: its literal last observed status. Only `done` / `completed`
  receive a completion check; `waiting`, `returned`, `stopped`, API failure and
  unconfirmed termination never masquerade as successful completion.
- Running actors and in-progress tasks show an activity spinner; running actors
  also show their observed tool count. This is not a heartbeat or data-flow proof.
- The outer viewer keeps its normal OBSERVED / DEMO / collection-off labels.
  Last-observed running state may persist after an unobserved force-kill.
- Edges are caller declarations, never measured parentage or tool traffic.
  Editing any presentation never starts/stops agents, tasks or permissions.

## Visual effects and motion controls

Every declared edge has moving colored heads (`●`) with short fading trails.
Packets follow the same source-to-destination polyline as the stroke, including
dashed and cross-rank routes. Arrowheads and node text remain readable.
The Panel labels this **Forward flow · decorative**: these are presentation
effects, not measured data transfers, scheduling, causal relationships or proof
that any worker is alive. They can move even on unknown/returned nodes.

Observed `running` / `in_progress` nodes have a gentle border-brightness pulse.
Other node states remain still. This reflects only the last observed status,
not a heartbeat. Focused Panels have heavy purple borders and bold headings;
unfocused Panels retain thin rounded borders. Both frames occupy the same cells.

An independent 100 ms motion timer refreshes visual effects; observation keeps
its existing 500 ms polling cadence. Frame ticks do not read state files or
write/increment telemetry. `a` turns motion off/on without changing collection;
off removes packets and freezes spinner/pulse effects. `p` freezes the displayed
snapshot and animation, ignoring in-flight snapshot results until resumed.
Navigation remains available and hooks continue collecting. Motion preferences,
clock and focus stay local to each viewer; no layout/schema change is needed.

## Link focus

Link focus is viewer-local inspection of **declared** directed relationships,
independent of observed lifecycle state. It starts with no selected node. While
the DAG leaf is focused:

- `]` / `[`: next/previous node in declaration order; from no selection choose
  the first/last. Selection never pans, scrolls or rearranges the graph; a node
  outside the viewport is still identified in the heading/details.
- `c`: clear node selection; keep the chosen scope/sidebar preferences.
- `f`: cycle direct neighbors → all upstream → all downstream → both.
  Direct neighbors is the default.
- `i`: toggle a 32-cell relationship sidebar **inside this leaf**, not a new
  caller panel. It shows the binding, literal state and relation/edge summaries.
  It needs a leaf width of at least **82 cells**; narrower leaves retain the
  graph viewport and show a width hint. Expand the terminal/leaf or toggle it
  off. Long details clip; the existing scrollable graph legend retains full
  endpoint IDs/edge labels. Closing the sidebar does not erase saved offsets.

With a selection, unrelated nodes/edges dim and unrelated packets stop. Related
routes keep the existing geometry, and colored/text markers distinguish
`*` selected, `<` upstream/incoming, `>` downstream/outgoing and `.` unrelated. The edge legend preserves each independent directed declaration.
No color interpretation alone is required.

Upstream/downstream focus inspects ancestors and descendants of the existing
forward graph; it does not add a reverse flow. An isolated node selects only itself.
Unknown/absent bindings remain unknown, unbound nodes remain declared-only.
Neither reachability nor particles prove actual messages, causal execution,
critical paths or worker liveness.

These controls do not replace existing arrows, Home or panel focus, and do
nothing in help/diagnostic pages or non-DAG leaves. On a caller hot-swap the
focused leaf is retained by ID where possible; a removed selected node clears
its selection, and a removed/retyped DAG loses its local link preferences.
No schema, task, permission or collection state is changed.

## Layout, routing and keyboard

Ranks follow longest paths through all declared edges, top to bottom. Node order
within a rank follows declaration order. Boxes have fixed 30-cell width, with
labels/state/meta clipped by grapheme cell width. Nodes do not reorder or shrink
into a different graph on small screens: the Panel becomes a viewport.

Adjacent-rank edges use inline lanes. Cross-rank edges have dedicated
right-hand rails and connect above/below boxes, not through intermediate nodes.
Line crossings are not graph nodes. Arrowheads identify destinations; a
scrollable edge legend below the graph preserves all endpoint IDs, labels and
styles when inline labels would not fit. Use that legend to disambiguate
crossings in dense graphs. This is a deterministic terminal layout, not a
crossing-minimizing graph editor.

- `Tab` / `Shift-Tab` or `j` / `k`: focus panels in caller order.
- Up/down and PgUp/PgDn: vertical scroll in the focused Panel.
- Left/right: horizontal pan in the focused DAG Panel (4 cells per key).
- `Home`: reset both axes of the focused Panel.
- Title reports effective x/y offsets and maximums. Offsets are viewer-local.
- `a`: toggle motion; `p`: pause/resume snapshot and animation.
- `q` / Ctrl-C: close only the viewer; normal help/diagnostic pages remain.

## Examples and earlier script

- `examples/dag/delegation.layout.json`: explicitly synthetic parent → child
  delegation only.
- `examples/dag/fanout.layout.json`: explicitly synthetic native DAG + tasks +
  events composition, including a forward cross-rank edge.
- `examples/ai-hero/{tdd,review,review-no-spec}.layout.json`: full native DAG +
  detail workspaces with live binding placeholders; the corresponding
  `.demo.layout.json` files are explicitly synthetic and fully fixture-bound.
  `{tdd,review}.dag.layout.json` are compact live alternatives. Plain `.dag.json`
  files are reusable graph objects. These are optional callers, not a default.
- `examples/issue-loop/loop.dag.layout.json`: one forward iteration of an
  optional caller-owned issue loop, not a graph back edge.

Preview without invoking Claude or changing its settings:

```sh
"$D" demo --json @examples/dag/fanout.layout.json
"$D" tui --session DEMO-layout
```

This uses labelled synthetic fixture data, not a running workflow. For live work,
copy the composition to a private file and bind IDs from the actual `status`.

`dag/dag-view.py` is the earlier standalone Python prototype, retained for
reference, not the native product path and not included in runtime packages.
To migrate an earlier `.dag.json`, embed its object in a `kind: dag` Panel's
`dag` field as above and publish the complete layout. No Python command is needed.
