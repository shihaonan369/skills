# Agent Dashboard Controls and Effects Roadmap

Labels: wayfinder:map

## Destination

Produce a controls and effects roadmap that can be handed off: define the interactions, data boundaries, priorities, and acceptance criteria for the first three enhancements, and decide whether an agent swimlane timeline and a collapsible event waterfall belong in the next batch.
The map originally aimed to clarify key decisions. The user later explicitly authorized implementation of the two confirmed enhancements first; see Notes for that execution exception and its boundaries. Other unresolved directions remain planning work; publishing is excluded.

## Notes

- On 2026-11-13, the user confirmed twice: plan first without implementing; use a restrained science-fiction console aesthetic.
- First batch to explore: DAG link focus, change echo, and one-key panel zoom. These were exploration directions, not specific designs or authorization to begin implementation.
- Next-batch candidates: an agent swimlane timeline and a collapsible event waterfall. Establish local data capabilities before deciding.
- Confirmed shared constraints: never steal focus or scroll position, and never change layout dimensions through effects; all information remains available with animation disabled. Explicit zoom is a user action, not an effect-driven layout change.
- Reuse the existing animation toggle and pause semantics; keep continuous animation restrained and use brief emphasis for primary feedback. Clarify pause, resume, and viewer-reopen boundaries in the relevant tickets.
- Distinguish telemetry from decoration: relationships come from caller declarations and states from real observations; missing start/end times and associations remain unknown. Label demonstrations synthetic/demo.
- Keep the product composable, terminal-independent, and Claude Code first; introduce no default workflow, agent execution/control, or additional terminal or end-user runtime dependencies.
- At each session, read agent-dashboard/docs/PANELS.md and, when working on DAGs, agent-dashboard/docs/DAG.md. Use grilling and domain-modeling; prototype tickets also require prototype. Create a research ticket only for questions local material cannot answer.
- Historical local-ticket convention: work by Type; open tickets with no unresolved blockers form the frontier; mark a ticket claimed before work. Resolve at most one non-research ticket per session. Current tracking conventions are in docs/agents/issue-tracker.md; this local map and its tickets are retained without automatic migration.
- Mapping is complete; user feedback resolved the visual-direction tickets, with terminal interactions split into a follow-up ticket. Resume with the lowest-numbered open, unblocked ticket or the user's selection; a visual choice is not implementation authorization.
- **Execution exception (user instruction, translated: "No, start work")**: implement the confirmed DAG link focus (A by default, B's sidebar on demand) and change echo. Reversible, explicitly documented engineering defaults may follow existing keys/layouts; do not present them as item-by-item human decisions. Keep the terminal-interaction ticket for later acceptance. Local builds, regression tests, and the existing six-target packaging are allowed; installation/configuration changes, publishing, and pushing are not. See spec.md for execution details.
- This section records the directions agreed during this round. Keep detailed answers in their tickets; retain only summary links in this map.

## Decisions so far

- [Which relationships should DAG link focus highlight?](issues/01-dag-link-focus.md): in-place focus on the original graph by default, with an on-demand relationship sidebar; clarify terminal interactions separately.
- [Which observed changes should change echo emphasize?](issues/02-state-change-echo.md): emphasize only changed rows and retain a static summary; no replay on load, duplicate snapshots, or resume, and no escalation of tool errors to agent failure.

## Not yet specified

- Reviewing the later change-echo and zoom prototypes alongside link focus may reveal unresolved visual hierarchy or interaction conflicts across controls. Specific DAG terminal-selection and sidebar questions already have a separate ticket and are no longer unexplored territory.
- If selected, new controls need further decisions about time scale, density, and composition with existing panels after data capabilities and priorities are clear.

## Out of scope

- No product features beyond the two authorized enhancements; no changes to telemetry collection or installation configuration; no publishing, pushing, or remote deployment.
- No inferred communication traffic, CPU usage/productivity, fabricated end times, agent relationships, or task success.
- Heat bars, sparklines, command palettes, state-transition trails, and scan lines remain separate future exploration topics outside this roadmap's decision scope.
