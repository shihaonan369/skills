# External file data and shared value bindings

## Availability and ownership

Version **0.4.0+** advertises `file-data-bindings-v1`. Older 0.3.0 binaries
do not have this capability. Check the actual executable;
preserve existing installations and report a missing capability instead of
silently replacing a binary or installing a build toolchain.

The caller declares the layout and approved local files. An external script owns
its data and lifecycle; the dashboard neither starts nor supervises it. Ordinary
value updates do not involve a model, scope classifier or background subagent.
They do not create agents, tasks, hooks, native progress or completion facts.
The dashboard's native `data-write` producer is optional; any existing producer
can atomically replace a compatible JSON file.

## Declare once; update only data

```json
{
  "version": 1,
  "data_sources": {
    "timer": {
      "type": "json-file",
      "path": "/private/agent-dashboard/countdown.json",
      "stale_after_ms": 3000
    }
  },
  "root": {
    "id": "countdown",
    "kind": "text",
    "title": "External countdown",
    "value": {
      "source": "timer",
      "path": "/remaining_seconds",
      "format": "mm:ss",
      "prefix": "Remaining: "
    }
  }
}
```

Substitute an actual private **absolute path on the viewer's host** before
publication (for example a drive-qualified path on native Windows). Use the
normal `layout --session ID --json @file` command once. Script updates change
only `countdown.json`:

```json
{"remaining_seconds":92,"updated_at":"2026-11-21T15:00:00Z"}
```

The source map is `data_sources`, not the existing Rex `sources` filter.
`value.source` names this local map; it is not a typed `bind.source` Rex alias.
`value.path` is an RFC 6901 JSON pointer, not JSONPath or a template/expression:
`/remaining_seconds`, `/items/0/value`, `/a~1b/~0key`, or `""` for the root.
`~1` escapes `/`, `~0` escapes `~`. Array indices have no leading zeros; missing
keys, out-of-range indices and null are unknown.

## Shared presentation surfaces

Every surface uses the same discovery, immutable file snapshots, pointer resolver,
freshness checks, formatter and sanitization. Widgets do not load their own files.

- Text panels accept **`value`**. A bound row follows optional caller-authored
  `text` lines, labelled `External data · not native agent/task telemetry`.
- All nine leaf kinds accept **`title_value`** with the same ValueBinding object.
  Display titles carry `[data]`. Row/column splits have no visible title and
  reject `title_value`; their descendants can use it normally.
- DAG nodes and edges accept **`label_value`**. Dynamic `dag.node` templates also
  accept it. Labels carry `[data]`; node/edge IDs, entity bindings, observed state
  and topology are unchanged. Text/label changes can affect geometry but retain
  stable identities. Existing DAG truncation/viewport rules still apply.
- These bindings work inside `repeat` and nested row/column layouts. Data source
  declarations stay global to that caller layout. Value bindings do not infer a
  per-agent JSON path or join external values to entity identity.

Example field decoration:

```json
{"source":"timer","path":"/remaining_seconds","format":"mm:ss","prefix":"Remaining: ","suffix":""}
```

The fields are `source`, `path`, optional `format`, `prefix` and `suffix`.
Prefix/suffix are literal printable strings, at most 160 bytes each. There is no
`${variable}` interpolation, environment expansion, evaluation, shell execution,
HTTP source or model request. Unknown keys and unsupported formats are rejected
at layout publication. Values cannot replace native agent/task identity or state.

### Formats

| Format | Input and behavior |
| --- | --- |
| `text` (default) | String, exact JSON number, or boolean; objects/arrays/null are unknown |
| `number` | A finite JSON number, retaining its written decimal digits; numeric strings are not coerced |
| `mm:ss` | Nonnegative remaining seconds, up to 315360000; rounds fractional seconds upward; minutes may exceed 59 |
| `percent` | Explicit 0..100 number, rendered with `%`; never guesses a 0..1 scale |
| `bar` | Explicit 0..100 number, rendered as a fixed ten-cell ASCII bar plus percentage |
| `deadline` | Explicit RFC 3339 timestamp string; computes nonnegative `mm:ss` from the display clock |

`mm:ss` never subtracts local time: if the producer pauses at 92, the viewer keeps
01:32 while the source remains fresh. Only `deadline` opts into local countdown.
Deadline expiry shows 00:00, not native task/agent completion. An arbitrary
`status` field does not pause a deadline; the producer controls semantics by
publishing remaining seconds, or a revised explicit deadline. Decorative
motion-off does not disable a logical countdown.

## Freshness and failures

`stale_after_ms` defaults to **5000**, with an explicit range **100..3600000**.
The snapshot is fresh only while the file's modification time and optional root
`updated_at` are within that budget. `updated_at`, when present, must be an RFC
3339 timestamp string. The older of the two times is used, so touching a file
cannot freshen an old reported timestamp. Timestamps more than two seconds in
the future are unknown clock-skew diagnostics.

A one-shot deadline producer must choose a freshness budget covering its intended
countdown, or keep publishing a heartbeat. The default five seconds deliberately
does not make an abandoned producer look indefinitely alive. Expired data shows
**stale**, not a still-ticking cached countdown. Missing/unreadable files, invalid
JSON/timestamps, bad numeric types/ranges and missing/null fields show **unknown**
with a short diagnostic. File errors stay local to their bindings; other panels
continue normally. Successful later snapshots recover without a layout rewrite.
No previous apparently-live success is substituted after a failed read.

The normal viewer observation refresh is approximately **500ms**, best effort,
not a hard real-time guarantee. Declared, referenced files are read once per
unique path per refresh, shared by all bindings. Layout parsing/publication, drawing,
animation, focus and scrolling do not read source files. Time-based freshness and
explicit deadlines are evaluated from the cached snapshot during rendering.
Pause freezes displayed values and their clocks. Resume refreshes the sources;
closing the viewer does not stop an independently owned producer. Existing
semantic-scope discovery may continue while paused without updating the display.

## Publish safely from an existing script

```sh
"$D" data-write --file "$DATA" --json '{"remaining_seconds":92}'
"$D" data-write --file "$DATA" --json "@$COMPLETE_JSON_FILE"
printf '%s\n' '{"remaining_seconds":91}' | "$D" data-write --file "$DATA" --json -
```

`D` is the verified executable and `DATA` an approved private absolute file path.
`data-write` validates one complete JSON value and atomically replaces only that
file with mode 0600; new parent directories use 0700. JSON objects without root
`updated_at` get the actual publication time. Explicit timestamps are preserved
and validated, not silently freshened. Scalar/array roots use file modification
time. Invalid writes leave the previous complete file intact. One producer owns
a data file; concurrent full replacements are last-writer-wins, not field merges.
The command does not acquire a session, rewrite a layout or produce telemetry.

Other producers must write a complete temporary file in the same private
directory, close it, and atomically replace the destination using their platform's
replace primitive. In-place partial writes are unsupported. Own that directory
and its permissions; the reader does not repair external file permissions.
Windows mode bits do not establish a private ACL: use a private caller-owned
directory on that host. Final file symlinks, directories and special files are
rejected. Do not select secrets or untrusted/stalled network filesystems.

Bounds: **16 sources**, **1 MiB per file**, **16384 JSON values**, depth **32**,
pointer **1024 bytes / 32 segments**. Duplicate object keys and trailing values
are rejected. Numbers retain JSON precision until numeric formatting requires a
finite number. Terminal controls are stripped and display strings are bounded.
Raw file values are transient and never serialized into session telemetry or
classification mailboxes; the caller layout still persists its paths/bindings.

See [the countdown caller example](../examples/data/README.md) and
[VALIDATION.md](VALIDATION.md) for exact platform/runtime validation scope.
