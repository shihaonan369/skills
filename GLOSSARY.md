# Agent Dashboard

A terminal dashboard that composes caller-declared structure with actually
observed information. This glossary distinguishes presentation relationships
from observed facts.

## Language

**Link focus**:
A view state that shows related nodes and directed relationships in the declared
graph around a selected node. It does not represent actual communication,
runtime state, or message delivery.
_Avoid_: link tracing, traffic monitoring

**Change echo**:
Local visual feedback for newly observed information changes. It expresses
differences between snapshots, not a heartbeat, throughput, or additional
lifecycle events.
_Avoid_: activity monitoring, status heartbeat
