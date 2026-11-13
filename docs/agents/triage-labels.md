# Triage Labels

Map canonical triage roles to GitHub labels in `shihaonan369/skills`.

| Canonical role | GitHub label | Meaning |
| --- | --- | --- |
| `needs-triage` | `needs-triage` | Maintainer needs to evaluate this issue |
| `needs-info` | `needs-info` | Waiting on reporter for more information |
| `ready-for-agent` | `ready-for-agent` | Fully specified, ready for an AFK agent |
| `ready-for-human` | `ready-for-human` | Requires human implementation |
| `wontfix` | `wontfix` | Will not be actioned |

When a skill applies a triage role, apply the corresponding GitHub label.
When moving between roles, remove the previous triage-role label while
preserving unrelated labels.

These are triage roles, not implementation progress or wayfinding states.
