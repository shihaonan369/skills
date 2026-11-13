# What Is the Entry and Return Contract for One-Key Panel Zoom?

Type: grilling
Labels: wayfinder:grilling
Mode: HITL
Status: open
Assignee: none
Map: [Agent Dashboard Controls and Effects Roadmap](../map.md)
Blocked by: none

## Question

Decide the user contract for panel zoom and return: entry/exit actions, eligible leaf panels, preservation of the original layout, focus, and each panel's scroll/pan positions, and behavior on window resize or caller layout hot reload.
Explicit zoom must not modify the caller's persisted layout definition or affect collection or agents; avoid ambiguous conflicts with existing quit, help, diagnostic-view, and pause actions. Inspect current local key bindings before proposing shortcuts rather than asking the user for discoverable facts.
