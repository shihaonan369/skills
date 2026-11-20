# Optional Rex adapters

Rex is an **optional transport and presentation adapter**, not the dashboard's
runtime, workflow, or telemetry model. Claude lifecycle collection and `tui`
continue to work without it. Nothing under `legacy/rex/` is loaded.

## Availability gate

Run `<D> capabilities` before using these commands. Require `rex-local-open-v1`
for `open` and `rex-observer-v1` for `rex-sources` / `rex-watch` / the
`programs` and `sources` panels. An unknown command means an older binary, not
permission to guess an executable path or silently change observation modes.

These adapters are in **source/development builds, version 0.2.0-dev**. The
published, immutable **0.1.0** downloads and bootstrap pins do **not** include
these capabilities. Bootstrap deliberately reuses existing binaries and does
not upgrade them. A matching locally built binary works; public first-use Rex
support requires a separately authorized release and reviewed pin update.
Do not replace published assets, install a build toolchain for an end user, or
claim that a stable download contains unshipped code.

An installed, reachable Rex CLI/server is needed only for these adapters. The
Lua observer is embedded in the Go executable and runs **locally inside the
existing official Rex CLI**. No Lua/Python/Node runtime, remote agent, persistent
script, daemon, or remote installation is added. Unsupported CLI/server APIs
surface as source errors; settings are left unchanged.

## Agent procedure: open an existing Claude canvas

After publishing the caller's normal Claude layout, run:

```sh
"$D" open --session '<actual-Claude-session-ID>'
```

`open` defaults to `--presentation auto`:

- With verified local Rex context, create a right-hand pane at 42% width using
  `focus=false` and direct executable arguments, or reuse the matching viewer.
- Without usable/verified Rex context, return `mode: terminal`, the exact `args`
  array, and a concrete `reason`. Use an already authorized terminal-control
  capability or show the corresponding normal `tui` command. No window was
  opened by this fallback.
- `--presentation terminal` skips Rex entirely. `--presentation rex` reports
  verification failure instead of pretending that a pane opened. Ownership
  conflicts, busy/corrupt receipts, and uncertain mutations remain explicit
  safety errors even in auto mode; they are not permission to create duplicates.

The operation checks live inherited session/block IDs, local IPC peer PID,
anchor host/placement, owned label, running executable, and window. A saved
label alone never authorizes sending input or replacing a foreign program.
Reusing a viewer preserves its placement and focus; it does not reset a layout
the user moved. Existing zoom can end when Rex creates a split.

An OS file lock serializes equivalent launches. Private atomic launch receipts
record IDs/signatures, **not observations or transcripts**. An ambiguous split
response is reconciled before retry. If creation is still unconfirmed, a durable
pending receipt blocks further splits: inspect the server and obtain explicit
recovery authorization rather than deleting the receipt or guessing that an
in-flight request is dead. A late successful pane is adopted on the next open.
Only an exited viewer with an exact owned receipt can be closed/replaced.
Different private state roots, source/layout content or executable builds are
different launches; existing viewers for other compositions remain untouched.
Receipt/lock metadata is retained, not globally garbage-collected. Clean old
receipts only with their viewers closed and any pending operation investigated.
This cooperative local check is not code attestation against concurrent
same-account binary/configuration replacement.

## Agent procedure: observe selected hosts/programs

Use this branch when the request is for basic cross-host or cross-agent program
status, including Pi in Rex. It needs **no Claude session ID, hooks, settings
edit, or remote deployment**. Full Pi tools/tasks/lifecycle remain unsupported.

1. Establish the scope from the request: current host/session and explicitly
   selected saved hosts. Ask when remote selection is ambiguous; do not connect
   to every saved host as an implicit default.
2. Resolve and pin sources. `--current` defaults to true and retains the current
   concrete Rex session if one exists; otherwise it selects this host's sessions.
   A stale inherited session/block context is an error, never permission to
   widen the current source to every session. Every `--host` explicitly opts
   into that saved host's sessions. Use
   `--current=false` for remote-only observation. Repeat `--host` for more hosts.
3. Capture the command's JSON as a **private UTF-8 file without a BOM**, using
   the agent's file tools. Keep source/layout files out of Git. On POSIX, this
   equivalent reference uses an existing private directory:

   ```sh
   umask 077
   "$D" rex-sources --host devbox > "$private/sources.json"
   ```

   Windows PowerShell 5.1 redirection defaults to UTF-16; do not use it for JSON
   input. Use file tools or `.NET UTF8Encoding(false)` instead. In PowerShell,
   invoke the binary as `& $D` and quote `@file` arguments.
   Reuse the existing private source/layout paths for the same attachment;
   recreating equivalent files at new paths describes a different launch.
4. Read [PANELS.md](PANELS.md) and compose the caller's layout with `programs`
   and/or `sources` panels. [The small example](../examples/rex/README.md) is
   presentation only, not a default pipeline. Supply real source-qualified
   record keys from `rex-watch --status` for exact `program_ids` bindings.
5. Open or reuse the local viewer, then verify its returned ID, rendered output,
   and source health before reporting success:

   ```sh
   "$D" open --sources "@$private/sources.json" --layout "@$private/layout.json"
   ```

   A standalone terminal uses exactly the same data/layout:

   ```sh
   "$D" rex-watch --sources "@$private/sources.json" --layout "@$private/layout.json"
   "$D" rex-watch --sources "@$private/sources.json" --status
   "$D" rex-watch --sources "@$private/sources.json" --plain --width 100 --height 32
   ```

`--status` and `--plain` wait up to 25 seconds for all sources to produce a
baseline or report a failure. They print explicitly partial data and return
nonzero if a source is unavailable or initialization times out. A TUI starts
immediately and shows connection progress. `q`/Ctrl-C closes the viewer and
cancels/reaps its subscription processes, not the observed programs. Pausing
freezes rendering; subscriptions continue. `--rex-bin` can select an absolute
existing local CLI path; the launcher carries that path to avoid PATH assumptions.

## Source and observation contracts

Source configuration is version 1: `sources` contains `name`, explicit `endpoint`,
pinned `host_id`, and optional `sessions` containing exact Rex session IDs.
Aliases are caller-owned labels, not agent roles. Duplicate aliases or pinned
host IDs are rejected: this version selects one endpoint per file-view host;
combine session filters instead of adding the same host twice. Hand-editing a
scope is an explicit caller operation; identity is checked again on connection.

Each source uses a persistent CLI connection, registers structured handlers
before the baseline, and reads current `program_status` for terminal blocks
whose `host_id` matches the pinned host. Foreign/unverified panes are counted as
excluded and are **not queried**. Selected-session sources do not subscribe
with `--all`. No screen text, process list/count, foreground command, or shell
history is used as program telemetry. Process metadata is consulted only by the
local launcher to verify its own existing viewer.

`program_status_changed` and `program_status_removed` invalidate the affected
pane. The observer re-reads its **whole current record list** on the same
connection. An event queued before the first snapshot cannot replay an older
state over it; missing records/fields are removed, not merged. Structural events
rebuild topology. Discovery progress keeps transport contact alive during a
slow staged baseline without declaring it online/ready. A 10-second identity
heartbeat and 60-second full reconciliation cover quiet connections and undocumented CLI startup/subscription gaps. This is
not a claim of raw API snapshot sequence watermarks, gap-free CLI delivery,
instantaneous freshness, or a fixed refresh SLA.

Disconnects, malformed/oversized messages, identity changes, missing sequences,
or unsupported methods trigger full-baseline reconnects with 1..30 second
backoff. A 45-second no-message watchdog cancels a stalled child. The old records
remain visibly **last known**, retaining their old instance label until a fresh
baseline is accepted. One source's failure never marks a program failed or
changes another source. Graceful viewer close cancels children immediately;
pipe-write/flush errors terminate the Lua observer if its consumer disappears.

Reports are untrusted plain data. `app`/`effective_app`, record ID, `state`,
`kind`, decoded `msg`/title, optional percentage, and server report timestamp
are displayed literally with terminal controls stripped. Known zero progress
is `0%`; absent progress is unknown (`?` in compact panels). Report age is not
connection health. A pane can have multiple records; `done` is not Issue
completion, and an ID hierarchy is not agent delegation. Full program keys hash
`[host_id, session_id, block_id, record_id]`, including the empty root record ID.
Use full keys from JSON, not shortened displayed block IDs, for bindings.

Observation snapshots stay in memory; source errors retain last-known data.
Caller-created configuration and local launch receipts remain private files.
Limits: 16 sources; per source 64 selected/discovered sessions, 256 local
terminal panes, 4096 reports; 8 MiB per wire message; 256 KiB source/layout files.
Exceeding a limit produces a source error, not fabricated replacement data.
Native Linux/Windows execution and Windows IPC peer support remain unverified;
an unavailable local peer proof causes a conservative terminal fallback.

## Primary references

- [Program status](https://www.superlogical.com/rex/docs/build/program-status)
- [Lua scripts](https://www.superlogical.com/rex/docs/automate/lua-scripts)
- [Lua events](https://www.superlogical.com/rex/docs/automate/lua-events)
- [Lua API](https://www.superlogical.com/rex/docs/reference/lua)
- [Events](https://www.superlogical.com/rex/docs/reference/events)

Use the installed CLI's `--help` and `api describe` when versions differ. No
private Rex socket protocol is implemented by this adapter.
