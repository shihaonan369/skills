# Agent Skills

Reusable agent skills, developed in one source repository rather than copied
into application configuration directories. Each skill owns its documentation,
implementation, examples, and tests.

## Available skills

| Skill | Overview |
| --- | --- |
| [Agent Dashboard](agent-dashboard/README.md) | A composable terminal dashboard for Claude Code. Presents observed session, agent, task, and event data alongside caller-declared graphs; the caller owns the layout and workflow. |

## Using a skill

Choose a skill above and follow its README for requirements, downloads,
installation, usage, and limitations. Packaging and setup are specific to each
skill; there is no repository-wide installer.

## Repository layout

```text
README.md                 Repository overview and skill index
AGENTS.md                 Repository-wide contributor rules
<skill-name>/
  README.md               Human-facing setup, usage, and development guide
  SKILL.md                Agent-facing instructions
  ...                     Implementation, tests, examples, and supporting docs
```

Keep the root README at overview level. Put each skill's detailed instructions
in its own README, with longer references under that skill's directory. When
adding a skill, add a concise entry to the index above.

## Development

Read [AGENTS.md](AGENTS.md) and the relevant skill's README before making changes.
Develop in this repository and use per-skill symlinks for local discovery;
preserve unrelated installed skills and hooks. Keep credentials, user
transcripts, and private telemetry out of Git.

Track work in [GitHub Issues](https://github.com/shihaonan369/skills/issues).
