# Issue loop: caller-owned long-running dashboard example

A caller-authored composition for a continuous loop — issues keep arriving,
each gets triaged, grilled and shipped as a PR, then the loop returns to intake.
**The skill supplies no loop engine**: the loop belongs to an existing workflow;
this example only presents it. No workflow skill is invoked by this example.

## Native DAG Panel

`loop.dag.layout.json` embeds `loop.dag.json` in a native `kind: dag` Panel,
with an event Panel below it. A `style: return` annotation closes the declared
loop without contributing to acyclic ranking or scheduling anything.

- `intake` binds `actor: lead`, the observed root session.
- `triage` / `grill` / `pr` bind reported task IDs `1` / `2` / `3`.
  These are illustrative bindings, not automatic stage detection. Rebind to the
  caller's real IDs; do not create tasks just to populate the picture.

From the skill root, using the executable matching the actual host:

```sh
D=./bin/darwin-arm64/agent-dashboard
"$D" start --session '<Claude-ID>' --title 'Actual issue loop'
"$D" layout --session '<Claude-ID>' --json @my-private-loop-layout.json
"$D" tui --session '<Claude-ID>'
```

Copy `loop.dag.layout.json` to the private layout file and rebind IDs from
`status` before publishing. Missing IDs render unknown. PowerShell uses `& $D`
and quotes `'@my-private-loop-layout.json'`. No Python graph process is needed.

## Rectangle companion

`loop.layout.json` remains an independent rectangle-only alternative (session,
lead, workers, tasks, events and caller text). Neither layout is a product default.
The caller may instead combine a DAG and other Panels in any row/column tree.

## Synthetic preview

```sh
"$D" demo --json @examples/issue-loop/loop.dag.layout.json
"$D" tui --session DEMO-layout
```

This binds the isolated synthetic fixture and stays labelled DEMO / SYNTHETIC.
It is not a running issue loop. Up/down scroll; left/right pan the focused DAG;
Home resets both axes. Editing presentation never starts/stops agents or tasks.
See [`../../docs/DAG.md`](../../docs/DAG.md) for schema and rendering limits.
