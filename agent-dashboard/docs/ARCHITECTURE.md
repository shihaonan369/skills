# Architecture and observation contract

## Three independent concerns

1. **Caller/workflow** chooses work, skills, agents and layout. This tool makes
   no AI Hero or screenshot workflow assumptions. AI Hero is an example only.
2. **Observation**: synchronous short-lived Claude settings hooks normalize real
   lifecycle/tool events into a bounded local snapshot after per-session opt-in.
3. **Presentation**: a caller-owned split tree binds Panel buffers to that
   snapshot. Bubble Tea + Lip Gloss provide terminal IO, color, resize, Unicode
   cell sizing and Windows support. The pure-Go executable has no runtime install.

Viewer-local `link_focus.go` computes declared reachability and an optional
in-leaf inspector; `echo.go` compares observed field values and decorates rows.
Neither changes storage, hook ingestion or the version-1 layout schema. Snapshot
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
