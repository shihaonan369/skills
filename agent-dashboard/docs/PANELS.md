# Panel interface v1

**Caller owns composition; dashboard owns rendering and local observation.**
A **Panel** is a rectangular TUI surface. A **buffer** is the bounded data shown
inside it. A **layout** composes panels through rows/columns. There is no workflow
engine, stage detector, agent router, role classifier, or automatic workflow inference. Version 0.3.0 adds explicit dynamic
membership templates, not inferred relationships.
Changing a layout never starts/stops agents or changes work items/permissions.

## Publish a layout

After `start --session ID`, send one JSON object with `layout --session ID --json
@file.json` (or an inline JSON argument). The update is atomic and the viewer
picks it up at its next refresh. A rejected layout leaves the previous one intact.
Use a private working file for live IDs or sensitive text; keep it outside git.

Optional Rex mode (0.2.0+) uses `rex-watch --sources @private.json --layout
@layout.json` instead of a Claude session store. Read [REX.md](REX.md) for the
capability gate and explicit source opt-in. That viewer reads configuration at
startup; changed source/layout content is a different launch, not a hot update
to another caller's running viewer. Normal Claude `layout` hot updates are
unchanged.

```json
{
  "version": 1,
  "name": "My caller's workspace",
  "root": {
    "kind": "column",
    "children": [
      {"id": "s", "kind": "session", "title": "Current session", "size": 8},
      {"kind": "row", "stack_below": 72, "children": [
        {"id": "a", "kind": "agents", "title": "Workers", "weight": 2},
        {"id": "p", "kind": "tasks", "title": "Existing plan", "weight": 3}
      ]},
      {"id": "log", "kind": "events", "size": 7}
    ]
  }
}
```

This is an illustrative **composition**, not a prescribed workflow. Swap it for
one text panel, a grid made from nested rows/columns, independent worker boxes,
or a completely different arrangement. See [`../examples/ai-hero/README.md`](../examples/ai-hero/README.md)
for two caller-chosen compositions using the locally inspected AI Hero skills.

## Composition nodes

| Field | Meaning |
| --- | --- |
| `kind` | `row` (left to right) or `column` (top to bottom) |
| `children` | Nonempty ordered array; caller order is preserved |
| `size` | Optional fixed cells along the parent's active axis |
| `weight` | Optional positive relative share of remaining cells; default 1 |
| `stack_below` | Only meaningful on a row: caller explicitly chooses the width below which it becomes a column |

All sizes/weights are integers 0..1000; 0 means omitted. Fixed requests are
honored when they fit; if their sum exceeds available cells, they shrink
proportionally and flexible children may have no space. The renderer never
reorders panels or invents a different process. When a leaf is too small to
render, the footer counts it. Resize or change the caller layout. `stack_below`
is opt-in; there is no hidden automatic rearrangement. Fixed `size` applies to
the new active axis after stacking. All-fixed children can leave blank space.

## Leaf panels

Every leaf needs a unique printable `id` (at most 80 bytes). Optional `title`
(default: ID), `size`, `weight`, and `color`: `peach`, `mint`, `blue`, `purple`,
`red`, `gray`, `teal`, `orange`, `green`, `amber`, `white`. Purple focus
highlighting is a UI state, not an advisor role. Focus uses heavy purple borders
and a bold heading; unfocused Panels use thin rounded borders with the same geometry.

| `kind` | Buffer/data binding | Optional fields |
| --- | --- | --- |
| `session` | Root Claude session, or Rex source/session scope without invented lifecycle | optional typed `bind`; `sources` for Rex scope |
| `agent` | One actual observed typed record; missing bindings stay unknown | `actor` (use `lead` for root agent) **or** typed `bind` |
| `agents` | Observed subagents, or explicitly bound typed record cards; no inferred roles/parent edges | `actors`, or `bindings` / `program_ids`; optional `sources` for Rex |
| `tasks` | Reported task metadata, or explicitly bound work-report records (not native tasks) | `task_ids`, or `bindings` / `program_ids`; optional `sources` for Rex |
| `events` | Newest-first bounded native events or explicitly labelled local Rex observation changes | `actors`, or `bindings` / `program_ids`; optional `sources` for Rex |
| `text` | Caller-authored operational text, explicitly labelled **not hook telemetry** | `text`: array of lines |
| `dag` | Caller-declared directed graph; node states use the shared typed actor/task/program binding | required `dag`: version-1 graph object; see [DAG contract](DAG.md) |
| `programs` | Optional Rex program reports, not agent/task lifecycle | `sources`: exact source aliases; `program_ids`: full source-qualified keys from `rex-watch --status`; `compact: true` for one source-prefixed row/report |
| `sources` | Optional Rex host/instance identity, transport health, last-known/error information, report/unreported/excluded-pane counts | `sources`: exact source aliases |

### Shared typed bindings (0.2.0+)

Require `observation-bindings-v1` for `bind` / `bindings` and program projections
in existing record panels. Every single-record panel and DAG node uses exactly
the same typed reference; lists reuse it without separate Rex implementations.
Read **[OBSERVATIONS.md](OBSERVATIONS.md)** before composing Rex cards, work rows,
source/session scope, events or program-bound DAGs. It owns the exact binding,
filter, missing-data, last-known and history contracts. Existing Claude selectors
and layouts remain compatible. Rex program projections require explicit selection;
an unbound native task/agent list does not silently substitute program reports.

An ID placeholder is **not** an agent. Giving a panel the title “reviewer” never
creates, classifies or assigns that role. Titles are presentation; actor bindings
must use real IDs from `status` or native results. Ownership names are not silently
mapped to IDs. Caller-authored text is never parsed as tool/agent status.

Rex panels require `rex-observer-v1`; stable 0.1.0 rejects these new kinds. A
program key is `rex:` plus 64 lowercase hexadecimal digits, qualified by full
host/session/block/record identity. Record roots may have an empty ID; shortened
block IDs are display only. Source and program filters intersect; missing keys
stay unknown. Source failure preserves reported program state with a last-known
label. Compact `?` means progress was not reported, not zero or idle. Use the
program diagnostic buffer for full message/title, report age and known progress.
A Rex `session` scope explicitly has no Claude lifecycle bound. Typed record
projections never manufacture a merged agent/task collection.

Limits: one object ≤256 KiB, ≤64 total nodes, depth ≤8, text ≤128 lines per leaf,
≤2000 input bytes/line (display/storage sanitize to ≤400 runes), title ≤160 bytes,
actor ID ≤120 bytes. Unknown fields/kinds, duplicate IDs, inappropriate bindings,
invalid sizes and excessive nesting are errors. Terminal controls/bidi overrides
are stripped. Long lines clip by terminal cell width, including CJK/emoji.

## Keyboard and lifetime

- `Tab`/`Shift-Tab` or `j`/`k`: focus leaf panels in caller tree order.
- Up/down/PgUp/PgDn: independent vertical scroll per focused buffer.
- Left/right: horizontal pan in the focused DAG Panel; Home resets both axes.
- DAG only: `[` / `]` select previous/next declared node; `c` clears selection;
  `f` cycles relationship scope, `i` toggles
  the optional in-leaf relationship sidebar. See [link focus](DAG.md#link-focus).
- `1`: caller canvas. `2`/`3`: full-screen task/event **diagnostic buffers**
  (program/source diagnostics in Rex observation mode);
  these do not replace the caller's layout or impose workflow stages.
- `a`: toggle DAG flow/spinner/pulse animation (decorative edges, not traffic).
- `p`: freeze displayed snapshot and animation; collection continues. Resume
  requests a fresh baseline immediately, without replaying paused changes.
- `x`: clear this viewer's change markers/summary, without changing telemetry.
- `?`/Esc: help; `q`/Ctrl-C: close viewer only.

## Observed change echoes

Session, agent/agents and task rows reserve a two-cell change-marker gutter.
A new observed field difference briefly marks only its affected row with `▸`
and a tint for **1.2 seconds**; after that, `·` remains as a static marker.
Tool-error rows use an amber echo, never a fabricated agent-failed state.
DAG state rows echo only changes to the state/count they actually display.

The footer's `Δ N fields` counts changed observed fields, **not events, tool
calls or throughput**. This latest-batch summary and static markers remain until
the next real field-change batch, baseline rebuild, or `x`. Unrelated notes,
layout edits, wall-clock ages and repeated snapshots do not restart the echo.

First load/reopen and resume establish baselines without echoes. Reads issued
before pause/resume are discarded, including late reads delivered after resume.
Read failures retain the old display and do not consume a pending baseline.
Motion-off keeps text/static markers; enabling motion never replays an old echo.
Help, diagnostic pages and a scrolled focused buffer suppress transient echoes.
Returning from those views does not replay them. On pause the short emphasis is
removed while static information remains.

No effect changes focus, scrolls to a changed item, expands a card or writes
telemetry. A newest-first event buffer at a historical offset keeps its reading
anchor when matching new events arrive (including actor filters); if bounded
history evicts that anchor it clamps to the remaining buffer. An offset of zero
continues following the newest entries. No hidden event IDs are inferred.

Claude layout and bounded lifecycle telemetry are local to one opted-in session.
Its viewer can be reopened from another terminal on the **same host/data root**;
several viewers observe the persisted layout independently. There is no renderer
daemon, network port or workflow process. Optional Rex mode keeps observations
in memory and owns cancellable local CLI subscriptions to explicitly selected
sources; closing it stops those subscriptions, not the observed programs. Its bounded events panel records local changes and explicit baseline/gap boundaries.
Its change echoes compare whole report/source-health observations, never report ages
or contact clocks; the footer labels those units `observations`, not events or
throughput. See [REX.md](REX.md) for its separate privacy/lifecycle boundary.

## Dynamic membership (0.3.0+)

Read [DYNAMIC.md](DYNAMIC.md) for the shared `select`, universal `repeat`, scoped
agent-classification mailbox and native-subagent-first procedure. Collections can
follow newly observed members, and any panel or split subtree can repeat an item
binding without caller rewrites on every spawn. Static declarations are unchanged.
Program/source panels keep their types; text stays caller-authored. Classification
changes display membership, never telemetry, task state or relationship facts.
