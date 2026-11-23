# Architecture and observation contract

## Three independent concerns

1. **Caller/workflow** chooses work, skills, agents and layout. This tool makes
   no AI Hero or screenshot workflow assumptions. AI Hero is an example only.
2. **Observation**: synchronous short-lived Claude settings hooks normalize real
   lifecycle/tool events into a bounded local snapshot after per-session opt-in.
   Optional Rex observation is a separate, in-memory program-report source; it
   does not create lifecycle actors/tasks or require Claude hooks.
3. **Presentation**: a caller-owned split tree binds Panel buffers to that
   snapshot. Bubble Tea + Lip Gloss provide terminal IO, color, resize, Unicode
   cell sizing and Windows support. The pure-Go executable has no runtime install.

Viewer-local `link_focus.go` computes declared reachability and an optional
in-leaf inspector; `echo.go` compares observed field values and decorates rows.
Neither changes hook ingestion or persisted telemetry. The additive typed
binding fields retain version-1 compatibility for existing layouts. Snapshot
reads carry a viewer generation so reads started before pause cannot overwrite
the post-resume baseline. Change/acknowledgement state is not persisted.

The small composition interface is documented in `PANELS.md`. Rendering is
separate from ingestion; layout updates never call agent tools or mutate tasks.
The installed source is this skill directory, not a copy under an app's config.

## Why settings hooks, not frontmatter hooks

Primary sources checked during implementation:
- https://code.claude.com/docs/en/hooks.md#hook-locations
- https://code.claude.com/docs/en/hooks.md#hooks-in-skills-and-agents
- https://code.claude.com/docs/en/hooks.md#exec-form-and-shell-form
- https://code.claude.com/docs/en/skills.md#available-string-substitutions
- https://code.claude.com/docs/en/hooks.md#common-input-fields

Current official docs say skill hooks register on invocation and remain for the
rest of the session; subagent-frontmatter hooks live only for that subagent.
Settings and plugin hooks explicitly cover tool events **inside subagents** with
`agent_id`/`agent_type`. `${CLAUDE_SKILL_DIR}` substitution is documented for skill
markdown and allowed-tools, not frontmatter hook commands. Therefore this build
uses absolute executable paths in **exec form** (`command` plus `args`), without
shells, platform scripts, PATH installation, or undocumented placeholder behavior.
No global collector daemon is installed.

Installation adds only twelve exact handlers (`SessionStart`, `UserPromptSubmit`,
`PreToolUse`, `PostToolUse`, `PostToolUseFailure`, `PermissionRequest`,
`SubagentStart`, `SubagentStop`, `Stop`, `StopFailure`, `SessionEnd`,
`PostModelSwitch`). `args` is exactly `["hook","--source","agent-dashboard-v1"]`.
Hook errors/malformed/oversize input are fail-open, print no decisions, and exit
0. They grant no permission and inject no context. `timeout: 3` and
`onFailure: continue` also make a missing executable nonblocking. The exact
introduction version of exec-form args is not documented; validated Claude
version is recorded in `VALIDATION.md`, not guessed as an older minimum.

Settings watcher docs currently say edits are normally picked up automatically.
For consistent initial coverage, start a **fresh Claude session after install or
uninstall**; do not promise already-running sessions have reloaded. Skill markdown
is live-detected, but that is separate from a session's loaded hook settings.
`doctor` proves configuration presence only. Check `status.hook_count` after
actual tool activity; zero means no observed hooks, not “all agents idle”.

`--bare`, disabled hooks, managed hook restrictions, safe mode, untrusted
interactive workspaces, cancellation/Esc and CLI event availability can suppress
hooks. Interrupted PostToolUse hooks can be cancelled. This is **last observed
state**, not a guaranteed exhaustive event stream, process supervisor, or
heartbeat. Force-killed sessions can retain running states. Root Stop means turn
ended, **not issue complete**; running workers remain visible. SessionEnd marks
unconfirmed workers as such, not falsely successful or failed. A SubagentStop is
an observed turn return, not proof that another hook did not subsequently resume
that worker. StopFailure records an API error without completing the issue.

## Normalization

- Tools are keyed by actor + `tool_use_id`. One completion cannot clear parallel
  tools or another actor's tool. Missing correlation IDs remain uncorrelated.
  PermissionRequest can lack an ID; matching outstanding tools show permission
  waits until subsequent correlated results. No tool failure fails the whole agent.
- Successful TaskCreate/TaskUpdate/TaskList/TodoWrite mirror task metadata when
  recognized. Task response schemas vary; unknown shapes stay unreported. Native
  task tools themselves are model/configuration-dependent. Do not assume every
  workflow exposes them. `plan` is an **optional caller data adapter**, not a
  generated task template. It atomically replaces explicit task metadata.
- Only reported task fractions are displayed. Per-agent progress requires exact
  owner ID matches; requested models, role-based models, costs, confidence,
  hidden reasoning and fork metrics are never inferred.
- Lead model can come from SessionStart/PostModelSwitch, or the bounded tail of
  its own transcript. Worker model is read from its own explicit transcript or
  conventional subagent file path, never the lead transcript or Agent input.
  Transcript writes can lag; unknown stays unknown. No transcript text is copied.
- No parent graph is inferred from chronology. Callers can draw desired structure
  through panel composition, labelled text or explicitly declared DAG edges
  (see `DAG.md`), but presentation is not measured parentage. No role names have privileged behavior.
- `finish` is an optional **explicit caller declaration**. The tool does not define
  prerequisites for it or decide a workflow's completion. A new lead tool/turn
  clears the declaration; the completion command's own PostToolUse does not.

## First-use bootstrap boundary

The skill agent runs `bootstrap.sh` (macOS/Linux/WSL) or `bootstrap.ps1` (native
Windows) before readiness checks. A working existing executable is reused
without network access or automatic update. Missing executables are acquired
from the fixed reviewed release named in the helpers in `shihaonan369/skills`; URLs and
version are not caller-configurable. Both the remote manifest's selected hash
and the actual archive must match the reviewed per-platform pin bundled in
`bootstrap.sha256`. This protects against later replacement of release assets
and checksums, but is not code signing, notarization, or a guarantee against
compromise of the skill source itself. Platform ZIPs contain a generated
`.prebundled` marker: they reuse the included executable or fail before network
access if no matching executable exists. Source main is repinned after publication;
release-tag source pins can describe the preceding release to avoid a circular
archive checksum. See README's release pin lifecycle.

The helper checks supported OS/native architecture and downloads with normal
TLS certificate verification. Download sizes and extracted binary size are
bounded. After verification, only the exact, unique expected executable member
is streamed to a same-filesystem staging file; archive paths are never unpacked
into source directories. A lock serializes first use. Publication is atomic and
no-replace (POSIX hard link, Windows .NET File.Move). The helpers do not execute
the payload and leave existing binaries, layouts, documentation, hooks, and
settings untouched. Temporary downloads are removed on handled exits; SIGKILL
can leave a POSIX lock that requires investigation, not automatic stale-lock
heuristics. Windows sharing locks release on process exit; the empty lock file
is intentionally retained to avoid unlink/reacquisition races.

The agent then runs `doctor`, obtains approval before running `install`, and
requests a fresh Claude session after hook changes. Automatic acquisition is
not permission to change application settings or override host execution policy.
No runtime/package manager is installed: POSIX uses existing HTTPS, SHA-256, and
ZIP utilities, while native Windows uses PowerShell/.NET. Missing tools,
unsupported platforms, read-only locations, and network/policy failures are
explicit blockers. First-use network access is separate from the collector,
which remains local. This does not implement Pi session variables or hooks;
optional Rex program observation has a separate capability boundary below.

## Optional Rex observation and presentation

The full source/version gates, agent procedures and source semantics are in
[REX.md](REX.md). Version 0.2.0 includes these capabilities; unchanged 0.1.0
assets do not. An existing executable is never upgraded implicitly.

`internal/rexwatch` hides official CLI transport, an embedded local Lua observer,
source-qualified identity, full-pane snapshot replacement and independent source
reconnects behind `Monitor.Snapshot` / `Close`. Native process IO is bounded and
cancellable; no private socket protocol or new server is implemented. Lua event
handlers invalidate/re-read a pane rather than applying older queued deltas.
Initial/reconnect baselines are staged until fully validated. Source failures
retain old reports and their old instance label, explicitly last-known, until a
fresh baseline commits. The CLI lacks a documented exposed baseline watermark;
60-second reconciliation bounds startup gaps rather than claiming losslessness.

`dashboard.ProgramState` retains a separate in-memory Rex snapshot without
manufacturing a Claude session, lead, task or completion. `observations.go` and
`observations_rex.go` adapt Claude and Rex into one typed observation view.
Single/list record panels, DAG state, relationship details, filters and change
keys share its binding resolver; they do not each implement Rex. A frame builds
the view once, leaving persisted state untouched. Explicit program projections
stay labelled as program reports. The small interface and exact semantics are
in [OBSERVATIONS.md](OBSERVATIONS.md).

`Monitor` also records bounded local report/source differences after validated
snapshot replacement. Initial/reconnect baselines mark unknown gaps, clock-only
changes stay silent, and discarded-entry counts are explicit. This is not a
lossless server history or reconstructed agent lifecycle. Viewer loading is
injected independently of rendering/motion.
Program values/source health participate in change echoes; report ages and source
contact timestamps do not. Diagnostic buffers are program/source views in this
mode. Source scrolling is independent of history anchoring; both Claude event
panels and explicitly composed Rex observation-history panels retain their
filtered reading anchors through the same normalization seam.

`dashboard.OpenViewer` is the independent, optional local launch/reuse boundary.
It verifies IPC peer credentials and live context before executing a local path;
locks and atomic private receipts serialize an exact build/configuration/anchor.
Pending receipts survive uncertain split outcomes and block unsafe duplicate
creation. Launch records store ownership metadata only, not observation data.
Changing layouts never starts observed agents; closing a Rex observer cancels
its own local subscription processes, not remote work. No Rex settings, legacy
collector, remote scripts or remote application hooks are installed.

## Storage and installation safety

Default state root: `<os.UserConfigDir>/agent-dashboard/state` (macOS Application
Support, Linux XDG config, Windows AppData). `AGENT_DASHBOARD_HOME` overrides it;
set it consistently for Claude/hooks and viewer. Session IDs hash to directory
names. Unknown/unopted sessions do not create files or store event payloads.
`deactivate` stops collection for one session; closing a viewer does not.

Each transaction uses a cross-process OS lock (flock/LockFileEx) with a bounded
wait, then fsyncs and atomically renames a JSON snapshot. Reader locks provide
consistent snapshots, including on Windows. Kernel locks release on process
exit; no stale PID heuristics. Limits: 500 events, 256 agents, 1024 tasks, 1024
active tools, 2048 result dedup keys, 4 MiB hook input, 256 KiB layout. The footer
shows dropped records at bounds, rather than inventing replacement data. State
files >8 MiB or incompatible/corrupt state are rejected, never silently reset.
Per-session history is bounded; total session retention is manual (remove an
inactive session directory with Claude/viewers closed). There is no unbounded
transcript/event append file.

Directories/files are created 0700/0600 on POSIX. Windows uses inherited profile
ACLs: POSIX mode bits are not a Windows ACL guarantee. Keep the state root in a
private local profile, not a shared or network filesystem. Do not commit state,
user transcripts, live layout IDs/notes or credentials. Explicit task subjects,
notes, paths and caller text can still be sensitive. No prompts, arbitrary tool
arguments, responses, shell commands or source are persisted by the collector.

The installer serializes its own settings writers, validates JSON/types, backs
up exact old bytes, checks for a competing edit, then atomically replaces the
settings file. Idempotent installs make no backup/write when unchanged. Editors
that ignore the lock still have a small unavoidable check/rename race. Uninstall
matches this executable's exact command+args only; it keeps unrelated handlers,
settings, skills, telemetry, backups and open viewers. Skill linking refuses to
replace an existing foreign path/directory. No source changes to Claude or Rex.
Moving the binary requires uninstalling at the old location before reinstalling.

## External file data and shared presentation values

Source 0.4.0-dev adds `external_data.go` behind `file-data-bindings-v1`.
`PreparePresentation` reads referenced approved local JSON paths once each per
observation refresh into private transient state. Errors are per-source results,
not global canvas failures. One pure pointer/freshness/format resolver serves
text bodies, all leaf titles and static/dynamic DAG labels. The same effective
layout drives drawing and navigation. Raw values are never serialized into
Claude telemetry, Rex reports or semantic-classification mailboxes.

The module owns strict bounded decoding, duplicate-key checks, freshness and
clock-skew diagnostics, safe formatting, and optional `PublishDataFile` atomic
writes. There is no hypothetical network/process adapter, expression evaluator
or per-widget file reader. Layout parsing/rendering never opens data files.
Entity bindings remain distinct from presentation value bindings. Producers
run only under the caller's lifecycle; the dashboard does not dispatch scripts,
agents or models. See DATA.md for the complete interface and trust limitations.

## Dynamic presentation and caller-owned classification

Version 0.3.0 adds one pure materialization step between normalized observations
and the split-tree renderer. Identity-based selection, repeats and generated DAG
nodes use the same observation seam. Focus, links, scrolling and history use the
same effective tree, while authored layouts and telemetry remain unchanged.

Optional private scope mailboxes are refreshed only at observation load, not by
rendering or animation. The dashboard discovers candidates and consumes explicit
fingerprinted replies; it never owns an LLM client, native agent dispatcher or
terminal/role inference. The initiating harness preferably supplies one native
background subagent for initial and later decisions. A worker-owned Monitor can
consume the bounded, deduplicated `scope-watch` JSON-line event source; watch
renewal stays in that worker's context. The command only watches mailbox changes,
not the model or conversation. Background long polling, bounded foreground
delegation and finally main-agent checkpoints are explicit capability fallbacks.
Unknown/missing judgements are excluded. See DYNAMIC.md for protocol, privacy,
budgets, failure semantics, shutdown and honest runtime acceptance boundaries.
