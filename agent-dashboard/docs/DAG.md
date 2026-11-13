# Native DAG Panel v1

**Caller owns the graph; the viewer owns rendering and local observation.**
`dag` is a leaf Panel in the bundled Go executable, composable with all other
Panels through `row` / `column`. No Python or other end-user runtime is required.
There is no graph executor, agent router, automatic dependency inference or
fixed workflow. A graph with return annotations has an acyclic ordinary-edge
core; the return annotations do not participate in ranking.

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
            {"from": "a", "to": "note", "style": "dashed"},
            {"from": "note", "to": "lead", "style": "return"}
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
- Binding: exactly one of `{"actor":"<observed-id>"}` or
  `{"task":"<reported-id>"}`. Use `actor: lead` for the root session.
  IDs are exact; role names/labels do not classify or create agents.
- Edge: required `from`, `to` referencing declared nodes; optional `label` and
  `style` (`solid`, `dashed`, `return`; omitted means solid).
  Duplicate directed endpoint pairs are rejected. Return self-edges are allowed;
  ordinary self-edges and cycles are rejected.
- Node colors use the Panel palette (see `PANELS.md`). Earlier graph colors
  `orange`, `green`, `teal`, `amber`, `white` are also supported.

Unknown fields, invalid styles/colors/bindings and ordinary-edge cycles are
errors. Limits per graph: 1..64 nodes, at most 256 edges; node ID 80 bytes,
label/name/edge label 160 bytes, binding ID 120 bytes; at most 4 meta lines,
2000 input bytes each (sanitized to 400 runes). Terminal controls/bidi overrides
are removed from display text; IDs containing them are rejected. The whole
layout remains subject to its existing 256 KiB and tree limits.

## Parent/child delegation and reporting

A round trip is **two independent directed declarations**, not a single
undirected line and not a workflow loop inferred by the viewer:

```json
{
  "version": 1,
  "nodes": [
    {"id": "parent", "bind": {"actor": "lead"}, "color": "peach"},
    {"id": "child", "bind": {"actor": "<actual-child-id>"}, "color": "blue"}
  ],
  "edges": [
    {"from": "parent", "to": "child", "label": "delegate work"},
    {"from": "child", "to": "parent", "label": "report results", "style": "return"}
  ]
}
```

The ordinary edge lays out parent above child; the explicit return annotation
has its own external rail and does not change ranks. Packets follow each edge's
own source → destination order, so the child's report travels **up** its rail
toward the parent. The rail keeps a visible `▲` marker (or `▼` for a
downward route); destination arrowheads identify the actual receiver. Colors
follow the declared source node, distinguishing delegation from the reply.
Graphs containing explicit opposite endpoint pairs show `⇄ paired arrows`.

Labels express the caller's communication contract, not measured message
contents or timing. A report edge can coexist with child → verdict findings
aggregation; it does not mean the child is a parent, create another agent or
complete a task. Only the caller declares that work should be reported back.
Last-observed child status is still displayed separately; moving reply packets
do not prove that a result has arrived. Without the explicit reverse edge, the
viewer does not invent one. Two opposite ordinary edges remain an invalid cycle:
mark the reporting edge `return`.

See `examples/dag/delegation.layout.json` for an explicitly synthetic small
parent/child round trip, and the AI Hero Review workspaces for two parallel
workers each reporting back to their lead.

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
dashed, cross-rank and return routes. Arrowheads and node text remain readable.
The Panel labels this **Flow animation · decorative**: these are presentation
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

## Layout, routing and keyboard

Ranks follow longest paths through ordinary edges, top to bottom. Node order
within a rank follows declaration order. Boxes have fixed 30-cell width, with
labels/state/meta clipped by grapheme cell width. Nodes do not reorder or shrink
into a different graph on small screens: the Panel becomes a viewport.

Adjacent-rank edges use inline lanes. Cross-rank and return edges have dedicated
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
  delegation plus child → parent reporting, each with its own direction.
- `examples/dag/fanout.layout.json`: explicitly synthetic native DAG + tasks +
  events composition, including a cross-rank edge and return annotation.
- `examples/ai-hero/{tdd,review,review-no-spec}.layout.json`: full native DAG +
  detail workspaces with live binding placeholders; the corresponding
  `.demo.layout.json` files are explicitly synthetic and fully fixture-bound.
  `{tdd,review}.dag.layout.json` are compact live alternatives. Plain `.dag.json`
  files are reusable graph objects. These are optional callers, not a default.
- `examples/issue-loop/loop.dag.layout.json`: optional caller-declared loop.

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
