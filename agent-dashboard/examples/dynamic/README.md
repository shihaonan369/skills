# Caller-owned dynamic presentation examples

These require version 0.3.0+ and `dynamic-layout-v1`; older 0.2.0 assets
cannot use them. Read [DYNAMIC.md](../../docs/DYNAMIC.md) before setup.
They are optional caller illustrations, not a workflow or product default.

- `actors.layout.json` follows exact reported `agent_type: implement`. It starts
  empty and adds matching observed Claude workers to the DAG, cards and list.
  The word implement is a deliberate example predicate, not inferred identity.
- `programs.layout.json` follows reported programs inside the caller's separately
  approved Rex source scope. It does not discover remote hosts, create agents or
  equate a program with a native task. The nodes are deliberately unconnected.

For semantic “implement agents”, copy a composition to private storage, initialize
an agent-owned scope mailbox, and replace each relevant selector's `roles` filter
with its absolute `scope_file` path. The initiating agent delegates the DYNAMIC.md
worker protocol to the same native background subagent for initial and later
classification. In Claude Code, prefer worker-owned Monitor with `scope-watch`;
follow the documented capability, lifetime, renewal and cancellation checks.
Neither example
silently starts a classifier, and neither contains fabricated telemetry or real
host/agent identifiers. Missing identity evidence stays pending/unknown.
