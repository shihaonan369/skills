# Which Relationships Should DAG Link Focus Highlight?

Type: prototype
Labels: wayfinder:prototype
Mode: HITL
Status: resolved
Assignee: Pi (local prototype session)
Map: [Agent Dashboard Controls and Effects Roadmap](../map.md)
Blocked by: none

## Question

Use explicitly demo-labelled low-fidelity sketches or a throwaway prototype to decide DAG link-focus interactions with the user: how to select nodes, switch between direct neighbors and upstream/downstream paths, and distinguish incoming edges, outgoing edges, return edges, and unrelated nodes.
Cover bidirectional edges, cycles, isolated nodes, and unbound observed state. Express only relationships in the declared graph; highlighting is not evidence of actual communication. Confirm recognition cues that do not rely solely on color. Store assets separately and link them here; resolution requires user feedback.

## Assets

- [Three-option interactive prototype](https://github.com/shihaonan369/skills/blob/prototype/dashboard-control-effects/agent-dashboard/prototypes/dag-link-focus.prototype.html)
- [Running instructions, trial scenarios, and validation boundaries](https://github.com/shihaonan369/skills/blob/prototype/dashboard-control-effects/agent-dashboard/prototypes/README.md)
- Screenshots: [in-place graph focus](https://github.com/shihaonan369/skills/blob/prototype/dashboard-control-effects/.scratch/dashboard-control-effects/assets/dag-link-focus-A.png), [relationship sidebar](https://github.com/shihaonan369/skills/blob/prototype/dashboard-control-effects/.scratch/dashboard-control-effects/assets/dag-link-focus-B.png), [relationship ledger](https://github.com/shihaonan369/skills/blob/prototype/dashboard-control-effects/.scratch/dashboard-control-effects/assets/dag-link-focus-C.png).

## Comments

Pi claimed the ticket and built a read-only synthetic prototype. All three options passed JavaScript syntax checks and initial Chrome headless rendering checks; these do not establish terminal runtime validation or full interaction acceptance.

The proposal used A as the default foundation and compared whether B's sidebar or C's relationship ledger was needed. Relationship computation switches between direct neighbors/all upstream/all downstream/combined. Return-edge participation is optional without deleting declared edges. Examples include bidirectional edges, self-loops, cycles, isolated nodes, and unknown/declared-only states.

The prototype initially awaited user feedback without an Answer. The user then replied (translated), "Follow the recommendation," confirming the visual choices below, not full interaction acceptance or product implementation authorization.

## Answer

The user accepted the recommendation to use A by default and expand B's sidebar on demand.

- Adopt A's in-place graph focus as the default visual direction, preserving spatial structure and deemphasizing unrelated items.
- Expand B's relationship sidebar on demand rather than keeping it permanently visible; C is not the default layout.
- Rationale follows the recommendation: retain the overall graph's spatial context by default and inspect node/relationship details when needed.
- Selection does not turn unknown states, declared relationships, or demo labels into evidence of actual activity.

This ticket resolves the **visual-direction choice**. Terminal node selection, scope defaults, return-edge participation, and narrow-window sidebar behavior from the original question were not individually approved. They are explicitly split into [How Should Terminal DAG Link Focus Select Nodes and Open the Sidebar?](07-terminal-link-focus-contract.md); do not claim the full interaction contract is settled. No Go renderer, product schema, collection, or installation configuration was changed, and direct promotion of prototype code was not authorized.
