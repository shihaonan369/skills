# Shared observation views and bindings

Version 0.2.0 advertises `observation-bindings-v1` in `capabilities`. Published
0.1.0 assets do not support it. The ordinary Claude layout schema and existing
`actor`, `actors`, `task_ids` and DAG actor/task bindings remain compatible.

## One binding interface, independent of panel shape

A binding contains **exactly one** typed reference:

- `{"actor":"lead"}` or another exact observed Claude actor ID;
- `{"task":"1"}` for an exact reported Claude task ID;
- `{"program":"rex:<64 lowercase hexadecimal digits>"}` for a full key from
  `rex-watch --status` (`sources[].programs[].key`). Never guess or shorten it.

`session` and `agent` accept `bind`. `agents`, `tasks` and `events` accept an
ordered `bindings` array. DAG nodes use the **same** `bind` object. Single and
list binding fields cannot be combined, and new bindings cannot be mixed with
legacy actor/task selectors on the same panel. Duplicate list bindings and
ambiguous/malformed references are rejected before publication. There are at
most 4096 list bindings per panel, within the existing 256 KiB layout limit.

Examples of equivalent projections, after substituting a real private key:

```json
{"id":"one","kind":"agent","bind":{"program":"<full-program-key>"}}
{"id":"many","kind":"agents","bindings":[{"program":"<full-program-key>"}]}
{"id":"work","kind":"tasks","program_ids":["<full-program-key>"]}
```

These are separate JSON fragments, not executable templates until substitution.
The last form is a shorthand for selecting program records in a list.
`program_ids` is also available on `programs` and `events`. `sources` filters
`session`, typed `agent`, `agents`, `tasks`, `events`, `programs` and `sources`
by exact configured source alias. Source/program filters intersect with explicit
bindings; a filter never widens the scope or authorizes another connection.
Legacy actor/task filters remain in their own namespace and cannot be mixed
with source filters. DAG identity is already pinned by its full program key;
filter its individual node bindings rather than adding panel-level source filters.

Unknown or removed bindings show **Not observed / unknown**. A record deleted
from a successful snapshot does not remain a stale live entity. Disconnected
sources retain their last accepted records, clearly **LAST KNOWN**. The same
resolver supplies card rows, DAG state/animation, relationship inspection and
change-echo keys. No widget uses a separate Rex status interpretation.

## What each shape means with Rex data

| Panel | Rex observation behavior |
| --- | --- |
| `session` | Without `bind`, shows selected source host/instance, transport health, report counts and sessions **with observed reports**. It does not aggregate program lifecycle/progress into a session lifecycle. With `bind`, shows the selected typed record |
| `agent` | Shows one explicitly bound program record, labelled **Rex program report (not native agent/task telemetry)** |
| `agents` | Shows explicitly selected typed records as cards, preserving caller binding order. It does not discover or classify agent roles |
| `tasks` | Shows explicitly selected typed records as work-report rows. The records are not native tasks; owner/dependencies/tools/models remain unreported |
| `events` | Shows bounded local report/source observations and baseline boundaries, not native agent lifecycle or a lossless remote event log |
| `dag` | A node can bind the same program key. Literal `working` animates only on a live source; last-known nodes stop activity animation. `done` stays literal, without a native task/Issue success check. Edges are caller-declared, never inferred from report ID paths |
| `programs` / `sources` | Existing complete report/source inspectors, with source/key filters and optional compact program rows |
| `text`, `row`, `column` | Existing caller-authored annotations and composition; no data-source-specific rendering |

Rex-only `agents`/`tasks` panels **without explicit program selection** explain
that Rex supplies no native agents/tasks. Select `bindings` or `program_ids`
rather than interpreting every program as a worker/task. `actor:"rex:…"` is an
actor lookup, not a program alias. A title such as "reviewer" never changes an
entity's type. Binding a program in several panels is several projections of
one record, not several agents or tasks.

Rex does not report models, tool counts, task ownership, delegation or Issue
completion. Missing progress stays unknown; an explicit zero stays `0%`.
`blocked` retains its reported permission/question/auth kind. Connection errors
never rewrite a program's state to `error`. Report age is displayed literally,
not used to infer disconnection or activity. Messages/titles remain untrusted
plain text with terminal controls stripped.

## Local observation history

The monitor records differences **after** accepting a validated full pane or
snapshot. Records are compared as complete replacements; omitted fields and
removed records clear. Clock-only changes (`updated_at`, contact time, report
age) do not generate entries or echoes. `added`, `changed`, and `removed` entries
use source-qualified keys and local observation time. Removal says outcome
unknown, not finished/failed. Source entries describe transport only.

Initial and reconnect snapshots add a `baseline` entry saying earlier
activity/reconnect gaps are unknown. They do not replay their records as new
activity or invent changes across disconnection. Periodic reconciliation after
an established baseline can record real differences; this still does not make
transport lossless or reconstruct intermediate transitions.

History lives only in monitor memory, retains the latest **500 observations**,
and exposes `changes` plus `dropped_changes` in `rex-watch --status`. The events
panel labels its origin and discarded-entry count. Source/program filters also
apply to events; an exact program filter excludes unbound source/baseline entries.
Use a separate unfiltered/source-filtered history panel when those boundaries
must remain visible. A scrolled event panel retains its reading anchor as
matching entries arrive. Diagnostic page `3` is still the source inspector,
not the events panel. Closing/reopening the monitor starts a new local history.

## Implementation seam

`observations.go` contains the typed binding resolver and normalized
`observationView` / `observationEntity` view model. The Claude adapter and
`observations_rex.go` adapter supply typed identities, factual rows, literal
states, activity/completion semantics, provenance, missing-data notices and
echo values. A frame builds this view once; record widgets resolve and render
it without branching on the transport. Specialized program/source inspectors
remain report diagnostics, not additional lifecycle adapters.

The view is derived and never serialized into Claude state. `ProgramState`
keeps Claude `Agents`, `Tasks`, `Tools`, `Events`, `Session` and completion empty.
Actual Rex observations/history remain in the separate `program_reports` value.
The module earns its depth by sharing resolution, filtering, row/echo identity
and state semantics across all record shapes, rather than creating one adapter
per widget or inventing a second telemetry schema in every renderer.

The public validation seams are `ParseLayout`, `ProgramState`, `Viewer` and the
real Bubble Tea lifecycle, plus `Monitor.Snapshot` at the official CLI executor
seam. See [VALIDATION.md](VALIDATION.md) for exact executed scope.
