# Native DAG + task/event Panels

This is a **synthetic presentation fixture**, not a running workflow. The caller
may use the same `kind: dag` leaf inside any row/column composition. There is no
DAG scheduler and no automatic inference of the graph from observed tools.

From the skill root, choose the binary for your host:

```sh
D=./bin/darwin-arm64/agent-dashboard
"$D" demo --json @examples/dag/fanout.layout.json
"$D" tui --session DEMO-layout
```

PowerShell uses `& $D` and `'@examples/dag/fanout.layout.json'`.

The fixture declares two forward branches, a merge and a cross-rank shortcut. Nodes bind `lead` and the three seeded demo task IDs. The task and
event Panels display the same isolated synthetic snapshot. No agents are spawned
and no settings are changed. Keep the DEMO / SYNTHETIC label visible.

For live work, copy the layout to a private file outside git, bind real IDs from
`status`, then publish with `layout --session '<Claude-ID>' --json @<file>` after
opting in. Missing bindings intentionally show unknown. Do not interpret the
fixture as the user's workflow or expose live IDs in committed examples.

Tab focuses the DAG; up/down scroll it, left/right pan it, Home resets both axes.
The scrollable legend below the graph lists all edge IDs/labels/styles. Edges
are caller declarations, not measured traffic or parentage. See
[`../../docs/DAG.md`](../../docs/DAG.md) for the complete contract.

## Forward delegation

`delegation.layout.json` is a small, explicitly synthetic alternative:

```sh
"$D" demo --json @examples/dag/delegation.layout.json
"$D" tui --session DEMO-layout
```

It declares only parent → child (`delegate work`). Both nodes display their own
observed state; no reply route or reporting completion is inferred. For live
work copy the layout privately, bind the child to its actual observed ID and
publish to that real session. Do not use the demo child ID for a live agent.

All bundled graphs are forward-only. Older `style: return` edges must be removed
from private layouts before publication; the renderer reports invalid old graphs
without modifying private files. See the [current DAG contract](../../docs/DAG.md).
