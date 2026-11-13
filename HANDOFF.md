# Handoff: portable Agent Dashboard skill

## Your assignment

You are a **new Pi instance**, opened at the user's explicit request to take over implementation. Develop the portable Agent Dashboard in this repository, not in the previous Pi's Rex configuration folder. Continue autonomously through design, implementation, tests and clear status reporting; do not just acknowledge this handoff and stop. Avoid changing the migration/scaffolding files while the previous Pi is doing its immediate startup verification; after inspecting the repo you own subsequent development.

User's latest instruction:

> /rex 可，但是把这个放到 ~/source/skills/ 里，然后ln到~/.claude下，以后在这个skills仓库里开发我自己的skills。你新开一个Pi接手开发即可。现在的rex方案挪过去以后就可以清理掉了

The previous Pi has done the migration and legacy uninstall. Your job is the new terminal-independent implementation, not re-installing the old prototype.

## Product context and agreed scope

Original request: reproduce the supplied Multi-Agent Orchestration Dashboard as a reusable skill. User tells Claude “实现某个 issue，并实时展示 dashboard”; they observe real agent/subagent status, overall plan, task progress, orchestration topology and events.

Original screenshot: `agent-dashboard/references/dashboard-reference.png` (copied from the user's Desktop screenshot). Visual vocabulary: dark terminal, peach lead card, mint orchestration/routing layer, blue worker cards, purple optional advisor rail, plan/dependencies and timestamped session log. Do not reproduce decorative fake data as actual telemetry.

The first implementation was Rex-specific: Python stdlib curses TUI, SQLite, global Claude hooks and `rex split`. User asked whether Rex alone could monitor an uninstrumented remote Claude with no remote install. After inspecting the native Rex events/API, the previous Pi explained that it cannot supply full structured Claude agent telemetry: terminal/process/screen status is not agent lifecycle/task/tool telemetry. User explicitly rejected degraded substitutes.

User then requested a **generic skill usable in any terminal, preferably cross-platform with no extra dependency installation**. A grilling round proposed four independent decisions; the user's “可” and explicit instruction to launch a new Pi authorize moving forward on the recommended scope:

1. Skill and its bundled implementation may live in every environment where Claude actually runs, including remote hosts/containers. Collect locally through real events/hooks; no separate long-running daemon/service deployment. This is not a promise of remote monitoring with no remote skill at all.
2. Zero-extra-dependencies means **end users need not install Python, Node, pip/npm packages, tmux or another runtime**. Ship a self-contained executable for each supported OS/architecture along with the skill. A developer build toolchain is a separate concern. Do not claim Python curses meets native-Windows zero-dependency requirements.
3. “Generic” means terminal-independent; **Claude Code first**. Multi-harness support (Codex/Pi/etc.) is not a first-version acceptance condition.
4. Arbitrary terminals have no uniform split-control interface. The product may run in another terminal pane/tab/window with a clear launch command. **Automatic right splitting on every terminal is not required.** It is not acceptable to silently weaken real telemetry or other functional requirements.

For remaining implementation choices, choose sensible documented defaults. Ask only for decisions that materially change the accepted scope or require user-only permissions. Do not restart the full grilling process by asking these four already-accepted questions again.

## Repository and migration status

- Repository root: `<HOME>/source/skills` (new local git repository, no remote configured, no commits made by the migration Pi).
- Every user-authored skill should be self-contained under `<skill-name>/SKILL.md` in this repo. Future development stays here.
- Product directory: `agent-dashboard/`.
- Claude link: `~/.claude/skills/agent-dashboard` → this repo's `agent-dashboard/`.
- Current root `agent-dashboard/SKILL.md` is an explicit **development placeholder**, with `disable-model-invocation: true`. Replace it with working invocation instructions and enable model invocation when the portable implementation is genuinely usable. Do not advertise fake launch commands.
- Entire old implementation moved, not copied, into `agent-dashboard/legacy/rex/`: `dashboard.py`, `tui.py`, tests, README, and old skill. It's an uninstalled reference for behavior, visuals and tests; you may reuse useful ideas and eventually remove unnecessary legacy artifacts once the replacement is verified. Do not reinstall it or develop under `~/.config/rex/`.
- The old `~/.config/rex/agent-dashboard/` directory, old `~/.claude/skills/rex-dashboard` link, old nine global dashboard hooks and the old DEMO pane/state have been removed. Existing unrelated Claude settings, skills and Rex SessionStart hook were preserved. Settings backups were retained.
- The old Rex configuration's README no longer points to the removed directory. No changes were made to Rex `init.lua`.
- The previous Pi will run legacy tests once and verify your new pane startup, then leave implementation to you. Its final status should not claim that your portable version is finished.

## Existing code worth understanding

Read `agent-dashboard/legacy/rex/README.md`, then implementation and tests as needed. Baseline tests (24 at handoff):

```sh
python3 -m unittest discover -s agent-dashboard/legacy/rex/tests -v
```

The baseline includes serialized SQLite transactions; isolated active sessions; fail-open hooks; TaskCreate/TaskUpdate/TaskList/TodoWrite normalization; permissions, tool failures, starts/returns; same-agent transcript model scoping; a compact adaptive renderer; explicit issue `finish` versus an ordinary Claude `Stop`; bounded/sanitized events; settings backup/merge/uninstall; and Rex-only pane reuse with retry reconciliation. Preserve the useful behavior without depending on its Python runtime or Rex-specific launcher.

Baseline limitations must not become false promises: collection starts at opt-in; no reliable heartbeat; force-killed Claude may leave last observed state; native hooks may not expose nested parentage; native task owners may not map to a displayed worker; main/subagent model must come from actual data, not requested model guesses.

## Engineering objectives / acceptance

1. Keep the implementation self-contained in the skill directory. Pick a suitable self-contained executable approach (Go/Rust/etc. after inspecting the environment and current official docs); build-time dependencies are allowed but end-user runtime installs are not.
2. Functional TUI on the supported macOS/Linux/Windows terminal targets. Handle resize, UTF-8/cell widths, colors, keyboard navigation, multiple workers, plan and log views. Closing the viewer must not stop Claude/agents.
3. Opt-in per-session accurate telemetry using supported Claude hooks/events, not screen-scraping guesses or process-count proxies. Investigate skill-scoped hooks versus global settings hooks using current official docs before choosing; document lifetime, subprocess coverage and hook reload/restart behavior. Never edit Claude's source.
4. Correct lifecycle semantics: Stop ≠ issue completed; parallel tools cannot clear each other's activity; task failure ≠ whole agent failed; active background workers should remain visible; models/progress remain unknown unless backed by data. Never fabricate the screenshot's JEV/confidence/fork metrics.
5. Installation/discovery from this repo via per-skill symlinks into `~/.claude/skills/`. Preserve other skills. If config hooks are needed, use atomic, backed-up, idempotent changes and narrow uninstall. No telemetry, secrets or transcripts in git.
6. Clear cross-platform packaging and usage docs. Avoid hard-coded home paths, Rex environment variables, terminal-specific requirements and shell-only commands without Windows equivalents. Explain native Windows linking/permissions and any actual distribution constraints honestly.
7. Automate meaningful tests for lifecycle normalization, concurrent writes, isolation, storage bounds, renderer safety, installation and CLI integration; verify on this Mac. Cross-build Windows/Linux if feasible, but report cross-compilation separately from real OS runtime verification.
8. Give the user a working local demonstration and concrete verified usage when ready. Label synthetic demo data explicitly. Record unverified platforms/limitations instead of claiming tests you did not run. Build/test locally; do not publish, push, deploy remotely or modify unrelated application configs.

## Available facts and references

- Installed Claude version previously checked: 2.1.312.
- Local Python reference baseline ran on Python 3.11.12, but this does **not** satisfy the new no-runtime-install target.
- Current primary references: https://code.claude.com/docs/en/hooks and https://code.claude.com/docs/en/skills . Use current official docs for exact hooks/skill semantics.
- This is a Pi coding session; read relevant Pi docs when touching its own harness. You do not need to build a Pi extension to deliver the Claude skill.
- A Bumblebee background research task was started for cross-platform self-contained TUI + Claude hook lifecycle + terminal split limits: `<research-run-id>`. One later attempt to retrieve it failed; do not treat its conclusions as verified or block development waiting for it. Retrieve if available or independently verify primary sources.

## Start now

Read repository `AGENTS.md`, inspect the migrated baseline, assess available toolchains without installing anything system-wide, then choose and implement the portable approach. Keep the user informed in this new Pi tab. Finish with tested usage and an honest verification matrix, or explain a real blocker clearly.


## Native DAG follow-up (2026-11-13)

The earlier portable implementation and its caller-owned Panel contract are
active; the historical assignment above is preserved for context. A native Go
DAG Panel is now integrated via the same `layout` / `tui` commands:

- `agent-dashboard/internal/dashboard/dag.go`: bounded graph schema/validation,
  deterministic ranked layout, cross-rank/return rails, observed-state binding,
  cell-aware viewport rendering. It never executes or infers a workflow.
- `Panel.Kind == "dag"` with a `dag` graph object composes inside any row/column.
  Left/right pan, up/down scroll, Home resets the focused viewport.
- Contract: `agent-dashboard/docs/DAG.md`. Start with the explicitly synthetic
  `agent-dashboard/examples/dag/fanout.layout.json`; optional AI Hero and issue
  loop graph layouts are caller examples only.
- The old `agent-dashboard/dag/dag-view.py` is reference-only, not the native
  runtime and not in platform archives. No Python runtime is needed by users.
- Seven DAG-focused Go tests, one added CLI integration test and bounded fuzz
  smoke cover this addition. All bundled layouts parse, including the previous
  `teal` example that exposed a missing palette entry.
- Latest exact checks/coverage, Mac PTY and six-target artifact verification are
  recorded in `agent-dashboard/docs/VALIDATION.md`. No new real Claude run or
  Linux/Windows/Intel-Mac runtime validation is claimed for the DAG integration.
- No app settings, remote host, git remote, published release or Rex installation
  was changed. The separate Matt setup is complete in root `AGENTS.md` plus
  `docs/agents/*.md`: local Markdown tracker, default triage vocabulary and
  single-context domain documentation conventions.


## AI Hero examples follow-up (2026-11-13)

- `examples/ai-hero/{tdd,review,review-no-spec}.layout.json` now compose native
  DAGs and detail Panels in one TUI. Live bindings use placeholders, not demo IDs.
- Separate `{tdd,review,review-no-spec}.demo.layout.json` files consistently bind
  the unchanged generic synthetic fixture. Use these only with `DEMO-layout`.
- TDD follows the inspected local skill's red-green loop, with refactoring outside
  that loop. Review verdict is caller annotation; no-spec variant removes the
  Spec node/card/edges/event binding. No skill or agent is executed by a layout.
- `.dag.json` graph objects and `.dag.layout.json` compact live alternatives match
  their primary workspaces, enforced by fixture-consistency tests.
- `examples/ai-hero/README.md` documents exact live binding locations, private-file
  handling and synthetic hot-swaps. Full compositions recommend 140×56 or larger;
  narrow windows stack detail Panels and leave the DAG as a pannable viewport.
- Validation: 35 passing top-level tests + 11 subtests, race/vet, six Mac PTY runs
  covering all three demo layouts and both exit keys. All six archives refreshed.
  Exact coverage and runtime boundaries are in `docs/VALIDATION.md`. No new real
  Claude/AI Hero execution, app settings change, deployment, commit or push.

## Motion/focus follow-up (2026-11-13)

- DAG edges now have colored decorative packet heads and short trails on their
  exact declared routes, including return/cross-rank rails. Graphs label the flow
  decorative; it is not measured traffic. Observed running/in-progress node
  borders have a gentle brightness pulse; other statuses do not.
- Focused leaves (DAG or ordinary Panels) use heavy purple frames/bold headings;
  unfocused frames remain rounded, with unchanged cell allocations.
- Viewer motion clock is 100 ms and independent of 500 ms snapshot polling.
  `a` toggles motion; `p` freezes snapshot/effects, ignoring in-flight snapshots;
  collection is unaffected. No schema change or extra runtime.
- Actual interactive Claude-to-DAG acceptance was performed via a new Rex session:
  two real parallel read-only checks, two completed native tasks, true actor IDs
  consistently bound; verdict unbound, no explicit overall finish. This is not
  a full AI Hero issue or invocation of `/code-review`/`/tdd`.
- Only that session's viewer was replaced for visual acceptance. Real captures
  confirmed moving packets, heavy focus, toggle/pause/resume and unchanged hook
  count (65→65). Claude remained running interactively. Live IDs/transcripts and
  working layout are private, not source fixtures.
- Latest exact validation: 38 passing top-level tests + 11 subtests, race/vet,
  ~1.58 ms plain-frame benchmark at 140×56, six AI Hero PTY effect runs, six-target
  packaging. See `agent-dashboard/docs/VALIDATION.md` for boundaries and history.

## Explicit delegation/report round trips (2026-11-13)

- The same version-1 graph schema uses two independent edges for a communication
  contract: parent → child delegates; child → parent reports with `style: return`.
  Replies do not affect ordinary-edge ranking. No relationship/message is inferred.
- Return/external rails now retain upward/downward markers, with source-colored
  packets following source → destination. Explicit opposite endpoint pairs show
  `⇄ paired arrows`; effects remain decorative, not measured message traffic.
- AI Hero Review graph objects, full/compact/live/demo/no-spec variants include
  worker-to-lead reports, independently of worker-to-verdict aggregation.
  `examples/dag/delegation.layout.json` is a small labelled synthetic example.
- The real session's private layout has corresponding report edges; actual
  bindings/lifecycle/task data are unchanged. Ghostty uses the same session store
  previously used by Rex, without a terminal dependency in the product.
- Later runtime check supersedes the earlier live-session note: the old Rex test
  session and Ghostty viewer had closed, and the root's observed state is now
  `ended` (66 hooks). A new Ghostty viewer shows that persisted non-demo snapshot
  at 140×47; no Claude or agent was restarted. Opt-in collection is not liveness.
- Regression tests verify stable ranks and upward-moving report particles via the
  public publication/snapshot/viewer seam. Full suite: 40 top-level + 13 subtests,
  race/vet; core coverage 88.9%. Exact PTY/package/runtime boundaries are recorded
  in `agent-dashboard/docs/VALIDATION.md`.
