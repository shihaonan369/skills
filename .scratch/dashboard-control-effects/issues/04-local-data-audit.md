# Audit Local Data Available for Timelines and Collapsible Events

Type: task
Labels: wayfinder:task
Mode: AFK
Status: open
Assignee: none
Map: [Agent Dashboard Controls and Effects Roadmap](../map.md)
Blocked by: none

## Question

Before choosing the next batch, perform a read-only audit of timestamps, event correlation IDs, tool starts/returns, agent ownership, concurrency, truncation, and persistence boundaries in current local code and tests. Record what the swimlane timeline and collapsible event waterfall can each show without inference, and where information is insufficient.
This local audit unlocks control selection. Do not change collection/rendering, inspect real private sessions/credentials, or access external services. Record results with code paths and test evidence; if tests run, record their exact scope. Raise a separate research ticket if external facts are needed rather than claiming unperformed verification. Link results as a separate asset instead of duplicating details in the map.
