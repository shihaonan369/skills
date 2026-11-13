---
name: rex-dashboard
description: Show a live multi-agent orchestration TUI in a right-side Rex panel while implementing an issue or running a multi-step task. Use when the user asks for a dashboard, agent/subagent monitoring, or 实时看板/右侧面板 during work.
argument-hint: "[issue or task]"
---

# Rex Agent Dashboard

Keep doing the user's task in this conversation. The dashboard is an observer,
not a separate orchestrator. Use actual agents only when the work benefits.

## Attach before doing the work

From the project directory, run:

```bash
python3 "${CLAUDE_SKILL_DIR}/../../dashboard.py" start --session "${CLAUDE_SESSION_ID}" --title "<short issue/task title>"
```

Use the **Claude session ID** substituted above. `REX_SESSION` identifies a Rex
workspace and is not interchangeable with it. The command opens a right split
without stealing focus; subsequent calls reuse that session's live viewer.
It needs `REX_SESSION` and `REX_BLOCK` inherited from Rex. Outside Rex, report
that requirement rather than choosing another session or creating a browser.

The result must contain a panel block ID before you say the panel is open.
If hooks were just installed, ask the user to restart Claude Code once; an
already-running session may retain its old hook configuration.

## Publish the plan, then execute

Use TaskCreate / TaskUpdate or TodoWrite for the actual plan. Installed hooks
mirror successful changes automatically. If those tools are unavailable, publish
an explicit plan (and republish it when statuses change):

```bash
python3 "${CLAUDE_SKILL_DIR}/../../dashboard.py" plan --session "${CLAUDE_SESSION_ID}" --json '[{"id":"1","subject":"Reproduce the issue","status":"in_progress"},{"id":"2","subject":"Implement the fix","status":"pending"},{"id":"3","subject":"Verify and review","status":"pending","blockedBy":["2"]}]'
```

Task statuses: `pending`, `in_progress`, `completed`, `blocked`, `deleted`.
`owner` can be an actual agent ID/name; `blockedBy` contains task IDs. Publishing
a plan replaces the previous explicit plan. Use `--json @/absolute/plan.json`
for longer plans. Progress means **completed / reported plan steps**, not an
estimate of remaining wall time. Use real task completion, never made-up agent
percentages. Unknown model/progress stays unknown.

Execute the issue normally. Hooks observe tool activity, permission waits,
subagent starts/stops and task changes. At each phase boundary or important
routing/review decision, add one short factual note:

```bash
python3 "${CLAUDE_SKILL_DIR}/../../dashboard.py" note --session "${CLAUDE_SESSION_ID}" "Reproduction confirmed; delegating tests while implementing the fix"
```

Advisor notes can use `--actor <actual-agent-id>`. The purple advisor rail is
populated only when a real subagent's type contains `advisor`; it is optional.
Write concise operational notes, not private reasoning, credentials or source.

## Verify and hand back

Keep all plan steps accurate, including blocked/failed verification. Mark work
finished only after implementing and checking it:

```bash
python3 "${CLAUDE_SKILL_DIR}/../../dashboard.py" finish --session "${CLAUDE_SESSION_ID}" --summary "Fix implemented; regression tests passed; review complete"
```

Leave the dashboard open for the user. `q` closes only its viewer, not Claude or
its agents. A normal Claude Stop means **waiting for the next prompt**, not
proof that the issue is complete. A subsequent prompt resumes live observation.
