# Which Observed Changes Should Change Echo Emphasize?

Type: prototype
Labels: wayfinder:prototype
Mode: HITL
Status: resolved
Assignee: Pi (local state-echo prototype session)
Map: [Agent Dashboard Controls and Effects Roadmap](../map.md)
Blocked by: none

## Question

Use demo-labelled before/after states or a throwaway prototype to decide with the user which observed changes trigger one-time emphasis, where that emphasis appears, how long it lasts, and how concurrent changes remain readable.
Clarify triggering on initial load, reopen, duplicate snapshots, history scrolling, pause, and resume; distinguish a single tool error from failure of the whole agent. Motion-off mode must retain text or markers without stealing focus/scroll position or changing dimensions. Store and link assets separately; resolution requires user feedback.

## Assets

- [Interactive state-rule prototype](https://github.com/shihaonan369/skills/blob/prototype/dashboard-control-effects/agent-dashboard/prototypes/state-change-echo.prototype.html)
- [Initial-state screenshot](https://github.com/shihaonan369/skills/blob/prototype/dashboard-control-effects/.scratch/dashboard-control-effects/assets/state-change-echo.png)
- [Running instructions and validation boundaries](https://github.com/shihaonan369/skills/blob/prototype/dashboard-control-effects/agent-dashboard/prototypes/README.md)

## Comments

After the previous visual choice was recorded, the user replied (translated), "Yes." This session then claimed this ticket and advanced only this item.
The prototype is a single HTML file with a pure reducer and an in-memory synthetic source; it connects to neither real telemetry nor the product. It offers free exploration and five guided scenarios: tool error versus agent failure, no replay after pause, duplicates and reopen, static/history views, and multiple changes in one batch.

Proposal at the time (**awaiting user confirmation**):
- Emphasize only changed rows; a single tool error must not mark the whole agent card as failed. Preserve parallel tools independently.
- Simulate 1200 ms of transient emphasis with a manually advanced clock; retain a textual change summary until the next batch, a new baseline, or explicit clearing.
- Initial load, reopen, and duplicate snapshots create no new emphasis; duplicates do not extend the emphasis deadline.
- Pause freezes the display while the synthetic source may keep changing; resume adopts the latest baseline without replay or an accumulated echo queue.
- Disabling effects retains states and static summaries; simulated history viewing suppresses new emphasis without replaying old changes.
- Only an explicit completed task state marks that task complete; returned/stopped do not mean completion.

JavaScript syntax checks, initial macOS Chrome headless rendering, and screenshot inspection were completed. Card and field heights were then bounded to prevent content changes from expanding the sketch. No formal terminal integration, real animation timing, cross-platform runtime, or full browser interaction acceptance was performed. The screenshot shows the initial state, not evidence of change animation.

The prototype initially awaited feedback. The user later instructed (translated), "No, start work," accepting the recommended rules and authorizing product implementation.

## Answer

Adopt the recommended rules in this ticket's Comments: emphasize only changed rows; separate tool errors from agent lifecycle; duplicates do not extend emphasis; initial load/reopen/resume establish a baseline without replay; disabling effects retains a static summary. Use 1.2 seconds as the first implementation's transient-emphasis default, not as a completed timing acceptance test.

The user explicitly authorized the confirmed DAG link focus and change echo, not panel zoom or new data controls. See this effort's spec.md for execution and validation records; this ticket records only the rule decisions.
