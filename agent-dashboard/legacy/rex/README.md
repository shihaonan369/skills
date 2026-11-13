# Rex Multi-Agent Orchestration Dashboard

Claude Code **skill + lifecycle hooks + Python stdlib TUI**. Implements the
reference's peach lead, mint orchestration layer, blue worker cards, purple
advisor rail, plan/dependencies and timestamped session log in a right Rex split.
No browser, HTTP server, daemon, pip packages or Rex Lua configuration changes.

## Use

Installed locally at `~/.claude/skills/rex-dashboard` (symlink to this directory's
`skills/rex-dashboard`). **Restart Claude Code once after installing the hooks.**
Run Claude inside a Rex terminal, then say:

> Implement issue #123 and show a live multi-agent dashboard in a right-side panel.

Claude can discover the skill from that request. To invoke it explicitly:

```text
/rex-dashboard Implement issue #123 and show the live plan, agents/subagents, and validation progress
```

The skill opens the panel, publishes a plan and continues the actual work in the
same Claude conversation. It doesn't spawn agents purely for a pretty graph.
Repeat invocations in the same Claude session reuse the viewer. Separate Claude
sessions get separate telemetry and panels. Closing the viewer doesn't stop any
agents; invoking again opens a fresh viewer for the same persisted state.

### What is automatic?

| Display | Source |
| --- | --- |
| Agent IDs / types / starts / returns | `SubagentStart` / `SubagentStop` |
| Current tools, completed tool count, errors | `PreToolUse` / `PostToolUse` / `PostToolUseFailure` |
| Permission waits | `PermissionRequest`; subsequent activity clears the wait |
| Plan steps / status / owner / dependencies | successful TaskCreate, TaskUpdate, TaskList or TodoWrite |
| Model | bounded tail of that agent's own local transcript, when available |
| Elapsed time | recorded lifecycle times |
| Planning/routing/review notes | explicit `note` calls by the orchestrator |
| Issue completion | explicit `finish`; a normal `Stop` means waiting, not completed |

The green layer shows observed orchestration, **not** fictitious JEV fork counts,
confidence scores or hidden reasoning. Per-agent progress is completed/assigned
plan steps when ownership is known; otherwise `progress —`. Overall progress is
completed/reported steps, not a predicted percentage of time remaining. No cost
or token estimates are invented. Model names are never assumed from agent type.

The purple rail appears on panels ≥80 columns and tall enough for the graph;
it remains `not attached` unless a real subagent's **type** contains `advisor`.
Use `note --actor <real-agent-id>` for advisory notes. Below 80 columns the graph
uses the available width without the rail. Smaller heights switch to worker rows;
very small panels ask to resize. Extra agents are paged with Tab; their actual
IDs, activity and recent events appear in the focus box when space permits.

### Keys

| Key | Action |
| --- | --- |
| `Tab`, `j`, `k` | Select / page workers |
| `↑`, `↓`, `PgUp`, `PgDn` | Scroll event log |
| `Home` | Return log to newest events |
| `p` | Pause local rendering (collection continues) |
| `l` | Expand log |
| `?`, `Esc` | Show / dismiss help |
| `q`, `Ctrl-C` | Close viewer only |

## Install / uninstall

Requires Python 3.10+ with curses (tested on macOS Python 3.11), Claude Code
supporting SubagentStart/SubagentStop hooks (tested installed version 2.1.312),
and the Rex CLI. Run this on the **same host where Claude and Rex terminals run**.
For a remote/devcontainer session, install there; no automatic cross-host copying
or local monitoring of remote Claude transcripts is provided.

```bash
python3 ~/.config/rex/agent-dashboard/dashboard.py install
```

Installation merges only its own hooks into `~/.claude/settings.json`, preserves
existing entries (including Rex's SessionStart hook), backs up changed settings,
and creates the skill symlink. Re-running is idempotent. The synchronous local
hooks preserve lifecycle ordering; they return no permissions, blocking decisions
or extra context. Hook errors always fail open. Hooks for sessions that have not
called `start` don't create telemetry or store event contents.

```bash
python3 ~/.config/rex/agent-dashboard/dashboard.py uninstall
```

Uninstall removes this install's exact hook commands and its skill symlink only.
It preserves other hooks, backups, saved telemetry and already-open viewers.
Moving the source directory requires uninstalling before moving and reinstalling
at the new location, because hook commands contain absolute paths.

## CLI / debugging

`--session` is a **Claude session ID**, not `REX_SESSION`. The installed skill
uses Claude's `${CLAUDE_SESSION_ID}` substitution. The Rex environment variables
identify the pane beside which to open the viewer, not the telemetry session.
The launcher requires both inherited IDs; it never guesses a different workspace.

```bash
D="$HOME/.config/rex/agent-dashboard/dashboard.py"
python3 "$D" start --session '<claude-id>' --title 'Issue #123 · Fix retries'
python3 "$D" start --session '<claude-id>' --title 'Issue #123' --ratio 55
python3 "$D" note --session '<claude-id>' 'Reproduction confirmed; implementing fix'
python3 "$D" plan --session '<claude-id>' --json '[{"id":"1","subject":"Reproduce","status":"completed"},{"id":"2","subject":"Fix","status":"in_progress"},{"id":"3","subject":"Verify","status":"pending","blockedBy":["2"]}]'
python3 "$D" finish --session '<claude-id>' --summary 'Regression tests passed; reviewed'
python3 "$D" status --session '<claude-id>'
python3 "$D" tui --session '<claude-id>' --plain
```

`plan --json` replaces the explicit plan; `@/absolute/file.json` is supported.
Successful native task tools can subsequently update it. `start --no-panel`
activates collection without opening a Rex split. `tui` can run in any ordinary
terminal with access to that host's telemetry. `status` emits a diagnostic JSON
snapshot; `--plain` prints one clipped TUI frame.

A clearly labelled, synthetic **layout preview** is available separately:

```bash
python3 "$D" demo --session rex-dashboard-preview
```

Demo telemetry is isolated under that ID and never represents real agents. Its
task counts are seeded examples, not an autonomous simulation or live Claude run.

## Storage, performance and limitations

- Per-session SQLite WAL databases under
  `~/.local/state/rex-agent-dashboard/<session-hash>/telemetry.sqlite3`.
  Session directories are 0700; database files are 0600. Override the data root
  with `REX_DASHBOARD_HOME` consistently for the hooks, launcher and viewer.
- Serialized short write transactions prevent overlapping hooks losing updates.
  Read transactions take consistent snapshots. The TUI polls at 2 Hz and redraws
  at up to 4 Hz, without any network service. Event history is bounded to 1,500
  records per session; 250 newest are available in the viewer.
- Only task metadata, agent metadata, tool summaries and explicit notes are
  persisted. Whole prompts, tool responses, source and shell command lines are
  not copied. Paths, search patterns, descriptions and notes can still contain
  sensitive information: keep the telemetry private. Source transcripts are
  read locally for model identification only, from a bounded 128 KiB tail.
- Observation begins at `start`; there is no historical replay of older tool
  events or reconstruction of agents already running before attachment. A model
  can still be read from an existing transcript after the first hook reports it.
- Hook availability and tool payloads depend on Claude Code. Events lacking a
  subagent ID are attributed to the lead; missing model data stays unknown.
  Workers are projected under the lead; nested parentage is not inferred when
  hooks don't provide it. An Agent tool's requested model is not assumed to be
  the subagent's actual runtime model.
- No agent heartbeat exists in these hooks. The footer says how long since the
  last event and explicitly calls long silence `quiet`, not offline or hung.
  Force-killing Claude can leave the last observed status running if SessionEnd
  never fires. This is an observation dashboard, not a process supervisor.
- No automatic pause/cancel/retry controls, token billing or secret reasoning
  introspection. `p` pauses the display only. Ordinary tool failures show up as
  errors but don't falsely mark the entire agent failed.
- The source directory and Python interpreter must remain available while hooks
  are installed. All installation changes are local; Rex's `init.lua` is intact.

## Tests

```bash
python3 -m unittest discover -s ~/.config/rex/agent-dashboard/tests -v
```

Tests cover hook-to-snapshot behavior, concurrent writers, session isolation,
plan updates and rollback, transcript scoping, explicit finish versus normal
Stop, fail-open hooks, settings merge/idempotence/uninstall, panel reuse, Rex
argument placement, responsive render sizes and terminal-control sanitization.
The panel launcher is also smoke-tested against the installed Rex CLI. Synthetic
hook injection validates the local event-to-TUI path; a full Claude issue run
is a separate user-level acceptance check, not claimed by those tests.

Primary references checked during implementation:
[Claude hooks](https://code.claude.com/docs/en/hooks),
[Claude skills](https://code.claude.com/docs/en/skills),
and the installed `rex __internal skill` / command help.
