# AI Hero: native DAG workspaces chosen by the caller

These are **optional examples**, not a built-in AI Hero workflow. The dashboard
supplies presentation; the caller keeps its existing skills and chooses work,
agents, layout, bindings and completion policy. No workflow skill is invoked by
loading an example. No AI Hero logic is compiled into the executable.

The examples follow the locally inspected skill versions:

- `/tdd`: red → green at agreed test seams, repeating for the next slice.
  Refactoring belongs to review, not this skill's red-green loop.
- `/code-review`: Standards and Spec axes in parallel, with Spec skipped only
  when the caller confirms there is no spec. Verdict aggregation is caller-owned.

Read your own installed skills before using these examples: versions may differ.
The dashboard never infers stages, test outcomes or review roles from tool names.

## Choose a workspace

| Caller context | Live template | Explicit synthetic preview |
| --- | --- | --- |
| TDD | `tdd.layout.json` | `tdd.demo.layout.json` |
| Two-axis review | `review.layout.json` | `review.demo.layout.json` |
| Review with no spec available | `review-no-spec.layout.json` | `review-no-spec.demo.layout.json` |

The primary layouts now combine the **native DAG and detail Panels in one TUI**:

- TDD: session header; red-green graph beside lead/worker/task Panels; observed
  events and caller context below. Red/green nodes bind reported tasks, not tools.
- Review: session header; fan-out/merge graph beside Standards/Spec actor cards
  and reported review items; exact-ID-filtered events and caller context below.
  Each axis has two independent communication arrows: lead → worker delegates
  the pinned check; worker → lead reports results on an explicit return rail.
  Worker → verdict findings aggregation remains a separate declaration.
  The verdict node is unbound caller annotation, never invented execution state.
- No-spec review: removes the Spec graph node, actor card, edges and event
  binding together, including the Spec reporting return edge. The layout does
  not decide whether a spec exists.

For these full compositions, **140 columns ×56 rows or larger is recommended**.
The main row stacks below 104 columns; the review-card row stacks below 56.
Smaller viewports clip rather than silently change the graph. Tab selects the
DAG; up/down/PgUp/PgDn scroll it, left/right pan, Home resets both axes. Closing
with q/Ctrl-C affects only the viewer. Focus has a heavy highlighted frame;
`a` toggles decorative edge motion, and `p` freezes the displayed snapshot and
animation without stopping hooks. Packets are not measured data traffic.

`tdd.dag.layout.json` and `review.dag.layout.json` remain compact **live**
alternatives (graph + events), useful in shorter windows. The `.dag.json` files
are graph objects for embedding in other caller layouts. Tests ensure their
bindings match the graph in the corresponding primary live workspace. No Python
viewer or additional runtime is needed.

## Live use: bind observed IDs in a private copy

All live templates use explicit `replace-with-...` placeholders, except the
known root actor `lead`. They intentionally show unknown until bound. They do
not contain demo IDs, create workers or prescribe task creation.

1. Choose a live template and copy it to a private working file **outside git**.
2. Read `status` / actual task results for this session's real IDs.
3. Bind the same ID everywhere it is used:

   | Value | Locations |
   | --- | --- |
   | Standards actor ID | DAG node `standards.bind.actor`, actor card `standards.actor`, events `actors` filter |
   | Spec actor ID (only when present) | DAG node `spec.bind.actor`, actor card `spec.actor`, events `actors` filter |
   | Red task ID | DAG node `red.bind.task` |
   | Green task ID | DAG node `green.bind.task` |

4. Remove nodes/Panels that do not describe the caller's actual work. If native
   tasks do not represent red/green steps, omit their bindings or replace the
   graph with the caller's actual structure. Do not fabricate phase tasks just
   to fill the graph. The optional `plan` adapter mirrors real reported metadata.
5. Publish the private file using the existing interface. The viewer hot-reloads
   it without changing execution or hook collection.

From the skill directory, choose the executable for the host where Claude runs:

```sh
D=./bin/darwin-arm64/agent-dashboard
"$D" start --session '<Claude-ID>' --title 'Actual current work'
"$D" status --session '<Claude-ID>'
# After binding the selected private copy:
"$D" layout --session '<Claude-ID>' --json @my-private-layout.json
"$D" tui --session '<Claude-ID>'
```

PowerShell:

```powershell
$D = '.\bin\windows-amd64\agent-dashboard.exe'
& $D start --session '<Claude-ID>' --title 'Actual current work'
& $D status --session '<Claude-ID>'
& $D layout --session '<Claude-ID>' --json '@my-private-layout.json'
& $D tui --session '<Claude-ID>'
```

If the caller switches to review or drops the Spec axis, publish its new private
layout rather than launching a different renderer. Titles/edges declare intended
structure; they are not proof of measured parentage, traffic or successful tests.
Reply rails keep an upward direction marker and animate toward the parent;
`⇄ paired arrows` identifies explicit opposite endpoint pairs. These are declared
communication channels, not measured message traffic or a guarantee that a
worker has reported. Do not infer reverse edges for tasks/agents without a
caller-declared reporting contract.

A completed red/green task means only the reported task is completed. A returned
review worker is not a successful verdict. `finish` remains an explicit caller
declaration; this example defines no commit, PR or completion gate.

## Isolated synthetic previews

Use **only `.demo.layout.json` files** for the fully bound fixture previews:

```sh
"$D" demo --json @examples/ai-hero/tdd.demo.layout.json
"$D" tui --session DEMO-layout

# In another terminal, replace the composition without reseeding or restarting:
"$D" layout --session DEMO-layout --json @examples/ai-hero/review.demo.layout.json
# Or show the no-spec variant:
"$D" layout --session DEMO-layout --json @examples/ai-hero/review-no-spec.demo.layout.json
```

PowerShell uses `& $D` and quotes every `'@file'` argument. `demo` seeds isolated
fixture data under `DEMO-layout`, with an outer **DEMO · SYNTHETIC** label, a
DEMO-named layout and explicit synthetic context text. It never runs AI Hero,
Claude or a review. No settings change is needed for this preview.

Demo mappings are deliberate: TDD red/green nodes use fixture tasks `1`/`2`;
review Standards/Spec use fixture actors `demo-scout`/`demo-tests` consistently
in graph, cards and event filters. Their underlying roles remain `sample-A` /
`sample-C`; labels do not reclassify them. Task titles/states are generic fixture
data, not real test results. The generic `SeedDemo` is unchanged. Verdict stays
unbound in previews too. Never use these files as live bindings.

Session-filtered events may initially contain only the root event; no fake
review activity is generated just to fill the buffer. Optional explicit preview
annotations (still synthetic):

```sh
"$D" note --session DEMO-layout --actor demo-scout --text 'SYNTHETIC standards annotation'
"$D" note --session DEMO-layout --actor demo-tests --text 'SYNTHETIC spec annotation'
```

See [Panel contract](../../docs/PANELS.md), [DAG contract](../../docs/DAG.md)
and [validation scope](../../docs/VALIDATION.md). Static/PTY rendering of these
fixtures is not an end-to-end AI Hero issue execution.
