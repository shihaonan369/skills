# Agent Skills

Reusable agent skills, developed in one source repository rather than copied
into application configuration directories.

## Available skills

| Skill | What it provides | Runtime |
| --- | --- | --- |
| [Agent Dashboard](agent-dashboard/README.md) | A composable terminal dashboard for Claude Code: session and agent cards, reported tasks, events, and caller-declared DAGs. Includes relationship focus, optional relation details, and observed-change echoes. The caller owns the layout and workflow. | A self-contained executable for macOS, Windows, or Linux; no Go, Python, Node.js, tmux, or extra package installation required. |

Agent Dashboard observes real lifecycle/tool events after session opt-in. It does
not execute workflows, invent agent relationships, infer progress from terminal
screens, or treat a normal Claude turn ending as completion. Missing data stays
unknown. Included AI Hero layouts are optional examples, not product defaults.

## Download ready-to-use packages

**Use [GitHub Releases](https://github.com/shihaonan369/skills/releases/latest),
not GitHub's automatic “Source code” ZIP.** Source checkouts do not contain the
ignored executables. Each platform ZIP contains a complete `agent-dashboard/`
skill directory, its matching native executable, documentation, examples, and
third-party notices.

### Agent Dashboard 0.1.0

| Operating system | x64 / AMD64 | ARM64 |
| --- | --- | --- |
| macOS | [Intel Mac ZIP](https://github.com/shihaonan369/skills/releases/download/agent-dashboard-v0.1.0/agent-dashboard-0.1.0-darwin-amd64.zip) | [Apple Silicon ZIP](https://github.com/shihaonan369/skills/releases/download/agent-dashboard-v0.1.0/agent-dashboard-0.1.0-darwin-arm64.zip) |
| Windows | [Windows x64 ZIP](https://github.com/shihaonan369/skills/releases/download/agent-dashboard-v0.1.0/agent-dashboard-0.1.0-windows-amd64.zip) | [Windows ARM64 ZIP](https://github.com/shihaonan369/skills/releases/download/agent-dashboard-v0.1.0/agent-dashboard-0.1.0-windows-arm64.zip) |
| Linux | [Linux x64 ZIP](https://github.com/shihaonan369/skills/releases/download/agent-dashboard-v0.1.0/agent-dashboard-0.1.0-linux-amd64.zip) | [Linux ARM64 ZIP](https://github.com/shihaonan369/skills/releases/download/agent-dashboard-v0.1.0/agent-dashboard-0.1.0-linux-arm64.zip) |

[SHA-256 checksums](https://github.com/shihaonan369/skills/releases/download/agent-dashboard-v0.1.0/SHA256SUMS)
cover the six downloadable ZIPs. Select the OS/architecture **where Claude
runs**, including SSH hosts or containers; WSL uses a Linux package.

The macOS ARM64 runtime has been tested locally. Other targets are cross-built,
not runtime-verified. Builds are unsigned: no macOS notarization or Windows
signing/SmartScreen acceptance is claimed.

## Get started

1. Download the matching platform ZIP and compare its SHA-256 digest with
   `SHA256SUMS`. On macOS use `shasum -a 256 <file.zip>`, on Linux use
   `sha256sum <file.zip>`, or on Windows use `Get-FileHash <file.zip> -Algorithm SHA256`.
2. Extract it to a permanent directory. Keep the executable there after installing
   hooks; moving it later requires uninstalling at the old path first.
3. Follow [the installation guide](agent-dashboard/README.md#local-quick-start)
   to link the skill and configure its opt-in telemetry hooks. Existing unrelated
   skills/hooks are preserved. Start a fresh Claude Code session after installing.
4. Ask your workflow to show `/agent-dashboard` with the layout it needs, and run
   the printed viewer command in another terminal pane, tab, or window.

For a synthetic preview without installing hooks, from the extracted skill
directory on Apple Silicon:

```sh
D=./bin/darwin-arm64/agent-dashboard
"$D" demo --json @examples/dag/fanout.layout.json
"$D" tui --session DEMO-layout
```

Other platforms use the matching executable listed in the skill's README;
PowerShell invokes it with `& $D`. Demo data is explicitly labelled and isolated
from real sessions. Closing the viewer never stops agents or disables collection.

## Development

Each skill lives under `<skill-name>/SKILL.md`, with its implementation, tests,
examples, and documentation co-located. Develop in this repository and use
per-skill symlinks for local discovery; preserve other installed skills.

See [build and test instructions](agent-dashboard/README.md#developer-build-and-tests),
[the Panel contract](agent-dashboard/docs/PANELS.md), and
[the validation matrix](agent-dashboard/docs/VALIDATION.md).
The historical Rex implementation is uninstalled reference material, not the
current runtime. Never commit credentials, user transcripts, or private telemetry.
