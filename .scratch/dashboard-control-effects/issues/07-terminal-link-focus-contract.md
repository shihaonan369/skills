# How Should Terminal DAG Link Focus Select Nodes and Open the Sidebar?

Type: grilling
Labels: wayfinder:grilling
Mode: HITL
Status: open
Assignee: none
Map: [Agent Dashboard Controls and Effects Roadmap](../map.md)
Blocked by: 01

Dependency: [Which Relationships Should DAG Link Focus Highlight?](01-dag-link-focus.md) (visual direction resolved; this ticket is now unblocked).

## Question

Within the selected visual direction, decide the production terminal interaction contract with the user:
- How does node selection start, move, and clear? How are panel focus, node selection, canvas scrolling, and panning distinguished? Can off-screen nodes be selected, and does selection move the viewport?
- What are the default scope and switching actions for direct neighbors, upstream, downstream, and combined views? Do return edges participate by default? What finite relationship sets describe bidirectional edges, self-loops, and cycles, and how do text/symbols distinguish them without implying actual communication?
- How does the sidebar open/close on demand, and what happens when the window is too narrow, without automatically changing the caller's layout? How is the viewport preserved on close?
- How do selection and relationship views remain consistent when caller layout updates remove a selected node, the window resizes, rendering pauses, or animation is disabled? Unknown, declared-only, and isolated nodes must not be reinterpreted as lifecycle outcomes.

Inspect local key handling and existing layout contracts first; do not replace production bindings with browser-prototype keys. The prototype is a proposal, not confirmation of these details. Do not implement product changes or additional collection as part of this design ticket.

## Comments

This ticket carries the terminal-interaction questions exposed by the visual prototype but not individually confirmed by the user; do not reopen confirmed visual choices.

The user later explicitly instructed (translated), "No, start work," authorizing the two selected enhancements. Keys, scopes, sidebar behavior, node removal, and pause behavior were implemented using the reversible engineering defaults in spec.md; see the [DAG documentation](../../../agent-dashboard/docs/DAG.md#link-focus) for the production contract. This is not item-by-item HITL approval. The ticket remains open for feedback or adjustment after actual use and does not block the authorized implementation.
