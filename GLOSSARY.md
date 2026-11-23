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

**External data**:
Caller-authorized values supplied independently of native agent/task telemetry.
Displaying an external value does not establish an agent's identity, task state,
delegation relationship, progress, or completion.
_Avoid_: synthetic agent telemetry

**Value binding**:
A caller-declared projection from external data to a displayed value. Distinct
from an entity binding, which identifies an actually observed entity.
_Avoid_: variable interpolation, telemetry injection
