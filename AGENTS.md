# Personal skills development

- This repository is the source of truth for the user's own skills. Each skill lives at `<name>/SKILL.md`, with its implementation, tests and assets co-located. Agent config directories contain symlinks, not development copies.
- `agent-dashboard/legacy/rex/` is an uninstalled historical baseline, not the current product or an active installation.
- Product target: terminal-independent TUI, Claude Code first, macOS/Linux/Windows, no extra runtime/package installation for end users. Build-time toolchains are a separate concern. Never describe Python curses as native-Windows zero-dependency support.
- Agent Dashboard is a composable TUI capability, not a workflow. The caller owns actual panel layout and data bindings; never hard-code or infer a screenshot/AI Hero pipeline. Keep AI Hero integration as a separate illustrative caller example, not a product default.
- Use real lifecycle and task data; unknown values stay unknown. Keep demo telemetry explicitly labelled and separate from real sessions. Do not substitute screen heuristics or process counts for full agent telemetry without user approval.
- Preserve other skills and hooks when installing/removing this skill. Back up settings, use atomic/idempotent changes, and keep telemetry private. Never commit credentials or user transcripts.
- Keep changes in this repository. Do not modify Rex configuration or reinstall the legacy Rex-only dashboard. Do not publish, push, or deploy to remote hosts without user instruction.
- Record exact validation scope. A Linux/Windows cross-build is not a verified runtime test. The user requested a fresh Pi to take over implementation; work autonomously within the accepted scope and report blockers instead of silently weakening requirements.

## Agent skills

### Issue tracker

Track issues and specs as local Markdown under `.scratch/<feature>/`.
Before creating, fetching or updating issues, read `docs/agents/issue-tracker.md`.

### Triage labels

Use the five default triage roles as issue status strings.
Before triaging issues, read `docs/agents/triage-labels.md`.

### Domain docs

Single-context: root `GLOSSARY.md` and `docs/adr/`.
Before exploring the codebase, read `docs/agents/domain.md`.
