# Agent Dashboard · composable TUI skill

**A presentation capability, not a workflow. The caller owns composition.**

A terminal-independent, Claude Code-first TUI module with self-contained
executables. It supplies **Panels** (`session`, `agent`, `agents`, `tasks`,
`events`, `text`, `dag`) and `row`/`column` composition. The calling skill/agent owns
layout, role assignment and workflow; hooks supply last-observed real telemetry.
There is no screenshot workflow template or hard-coded AI Hero pipeline.

## Download

Get a ready-to-use platform ZIP from
[GitHub Releases](https://github.com/shihaonan369/skills/releases/latest).
Packages are available for **macOS, Windows, and Linux**, each in **x64 and ARM64**.
Each ZIP contains this skill directory, one matching executable, docs, examples,
and third-party notices. No end-user compilation or extra runtime is required.

Choose the uploaded `agent-dashboard-0.1.0-<os>-<arch>.zip` asset, **not** the
automatic “Source code” archives. Source checkouts omit `bin/` and `target/`.
Extract to a permanent directory before installing hooks; keep existing unrelated
skills intact. Platform verification and unsigned-build caveats are covered under
[Safety, distribution and limitations](#safety-distribution-and-limitations).

### Version 0.1.0

| Operating system | x64 / AMD64 | ARM64 |
| --- | --- | --- |
| macOS | [Intel Mac ZIP](https://github.com/shihaonan369/skills/releases/download/agent-dashboard-v0.1.0/agent-dashboard-0.1.0-darwin-amd64.zip) | [Apple Silicon ZIP](https://github.com/shihaonan369/skills/releases/download/agent-dashboard-v0.1.0/agent-dashboard-0.1.0-darwin-arm64.zip) |
| Windows | [Windows x64 ZIP](https://github.com/shihaonan369/skills/releases/download/agent-dashboard-v0.1.0/agent-dashboard-0.1.0-windows-amd64.zip) | [Windows ARM64 ZIP](https://github.com/shihaonan369/skills/releases/download/agent-dashboard-v0.1.0/agent-dashboard-0.1.0-windows-arm64.zip) |
| Linux | [Linux x64 ZIP](https://github.com/shihaonan369/skills/releases/download/agent-dashboard-v0.1.0/agent-dashboard-0.1.0-linux-amd64.zip) | [Linux ARM64 ZIP](https://github.com/shihaonan369/skills/releases/download/agent-dashboard-v0.1.0/agent-dashboard-0.1.0-linux-arm64.zip) |

[SHA-256 checksums](https://github.com/shihaonan369/skills/releases/download/agent-dashboard-v0.1.0/SHA256SUMS)
cover the six downloadable ZIPs. Before extracting, compare the downloaded
archive's digest with `SHA256SUMS`:

- macOS: `shasum -a 256 <file.zip>`
- Linux: `sha256sum <file.zip>`
- Windows PowerShell: `Get-FileHash <file.zip> -Algorithm SHA256`

Select the OS/architecture **where Claude runs**, including SSH hosts or
containers; WSL uses a Linux package. The macOS ARM64 runtime has been tested
locally. Other targets are cross-built, not runtime-verified.

## Local quick start

After extracting the platform package, binaries live under `bin/<os>-<arch>/agent-dashboard[.exe]`:

| Host | Binary directory |
| --- | --- |
| Apple Silicon / Intel Mac | `darwin-arm64` / `darwin-amd64` |
| Linux ARM64 / x86-64 | `linux-arm64` / `linux-amd64` |
| Native Windows ARM64 / x64 | `windows-arm64` / `windows-amd64` |

Use the binary for the host **where Claude runs**, including containers/SSH
hosts. An ANSI/UTF-8 terminal is needed (Windows Terminal recommended). No end-user
Go/Python/Node/npm/pip/tmux installation, server, or terminal split API is needed.
No remote telemetry is collected without installing the skill on that host.

macOS example, from this directory (choose your architecture):

```sh
D=./bin/darwin-arm64/agent-dashboard
"$D" install --skill-dir "$PWD"
"$D" doctor
```

This creates/reuses `~/.claude/skills/agent-dashboard` pointing here and merges only
its own exact hooks into Claude settings with backup/atomic replacement. Other
skills/hooks are preserved. **Start a fresh Claude session after installation.**
`doctor` checks configuration, not whether managed policy permits hooks or whether
a running Claude session loaded them. Observe increasing `status.hook_count`.

Then ask your existing workflow to call `/agent-dashboard` and supply its layout.
The caller should keep using its existing skills, not switch to a dashboard-owned
process. The skill uses the actual substituted Claude session ID, opts in, and
prints a viewer command to run in another terminal pane/tab/window on that host.
There is no automatic terminal-specific split and no separate orchestrator.

Native Windows PowerShell, from this directory:

```powershell
$D = '.\bin\windows-amd64\agent-dashboard.exe'
& $D install --skill-dir (Get-Location).Path
& $D doctor
```

Directory symlinks on Windows require Developer Mode or suitable privileges. If
symlink creation is unavailable, create the named skill junction yourself using
`cmd /c mklink /J` with properly quoted absolute source/target paths, then rerun
install; an existing link/junction resolving to this source is reused. The tool
refuses to replace foreign paths. WSL is a **Linux** installation with its own
Claude home/state, not native Windows runtime verification. Profile ACLs control
Windows telemetry privacy; mode 0600 alone does not establish Windows ACLs.

## Caller interface

```sh
"$D" start --session '<Claude-ID>' --title 'The actual current work'
"$D" layout --session '<Claude-ID>' --json @my-private-layout.json
"$D" tui --session '<Claude-ID>'
```

The caller defines the rectangle tree, buffers, labels, colors, sizes, actor/task
bindings and optional narrow-screen stacking. Layout updates hot-reload without
restarting the viewer. Without a supplied layout, the viewer explicitly says so
and shows a **session inspector**, not a guessed workflow graph.

See **[Panel contract](docs/PANELS.md)** for the JSON interface and keyboard
behavior. **[Native DAG Panel](docs/DAG.md)** adds caller-declared graphs,
fan-out/merge, cross-rank edges and return annotations inside the same TUI;
no Python or separate viewer is required. **[AI Hero example](examples/ai-hero/README.md)** demonstrates two
different caller-owned arrangements, using the locally inspected `/tdd` and
`/code-review` skills without modifying or invoking them. These examples ship
with every platform package; neither is imported into the executable. Their
primary workspaces compose native DAGs with detail Panels; live templates and
explicitly named `.demo.layout.json` synthetic fixtures have separate bindings.

Optional data adapters, independent of layout/workflow:

```sh
"$D" plan --session '<Claude-ID>' --json @reported-plan.json
"$D" note --session '<Claude-ID>' --text 'A concise factual operational note'
"$D" note --session '<Claude-ID>' --actor '<observed-ID>' --text 'Worker note'
"$D" finish --session '<Claude-ID>' --summary 'Caller explicitly reports completion'
"$D" status --session '<Claude-ID>'
"$D" tui --session '<Claude-ID>' --plain --width 100 --height 32
"$D" deactivate --session '<Claude-ID>'
```

In PowerShell replace `"$D"` with `& $D` and quote `@file` arguments, e.g.
`--json '@my-private-layout.json'`. The executable's `start`/`demo` output prints
a platform-appropriate viewer command, including paths with spaces.

`plan` replaces supplied task metadata, not a prescribed plan. Records contain
`id`, `subject`, `status` (`pending`, `in_progress`, `completed`, `blocked`,
`deleted`), optional `owner` and `blockedBy`. Native task tools update metadata
when available/recognized. No reported plan = progress unknown. `finish` is an
optional explicit declaration, not a workflow gate; ordinary Stop is not finish.
Closing the TUI never stops Claude/agents or disables collection. `deactivate`
turns collection off for that session while retaining its last snapshot.

## Synthetic demonstration

```sh
"$D" demo --json @examples/ai-hero/tdd.demo.layout.json
"$D" tui --session DEMO-layout
# Change composition without changing any synthetic agent/task state:
"$D" layout --session DEMO-layout --json @examples/ai-hero/review.demo.layout.json
```

All seeded data is isolated under `DEMO-layout`, labelled **DEMO / SYNTHETIC**,
and rejected by real hook ingestion. This is a layout preview, not a live AI Hero
or Claude run. A real Claude smoke test is reported separately in validation.

### Native DAG preview

```sh
"$D" demo --json @examples/dag/fanout.layout.json
"$D" tui --session DEMO-layout
```

This composes a native DAG with task/event Panels using isolated synthetic data.
Tab focuses panels with a heavy highlighted frame; up/down scroll, left/right
pan the DAG, Home resets its viewport. `[` / `]` select a DAG node, `f` changes
relationship scope, `r` toggles return participation, `i` opens optional relation
details, and `c` clears selection. These inspect declarations, not traffic.
Observed field changes briefly emphasize only the affected row, with a static
marker/summary retained after the effect; `x` clears that local summary. See the
[Panel contract](docs/PANELS.md#observed-change-echoes) for baseline/history rules.
Declared edges have decorative moving
packets and short trails (not measured traffic); `a` toggles motion, `p` freezes
the displayed snapshot and animation while hooks continue collecting. Embed the `dag` object into any caller layout; ordinary cycles are
rejected, and explicit return annotations do not execute a loop.

## Safety, distribution and limitations

- [Architecture/observation contract](docs/ARCHITECTURE.md): opt-in lifetime,
  subagent coverage, hook cancellation/reload, concurrency, storage/privacy bounds,
  unknown parentage/models, task response limitations and fail-open semantics.
- [Validation matrix](docs/VALIDATION.md): distinguishes tested Mac runtime,
  terminal PTY and Claude integration from cross-compilation only.
- Release binaries/archives are **unsigned builds**, not notarized or
  Windows-signed. macOS quarantine/Gatekeeper and Windows
  SmartScreen/enterprise policy can require user approval. Verify package hashes
  and source. No bypass or signature claim is made.
- A per-platform zip contains the skill, docs, examples, notices and one matching
  executable. Keep the executable at its installed path while hooks exist. Before
  moving it, uninstall using the old binary and install again at the new path.
- Uninstall with `"$D" uninstall --skill-dir "$PWD"` (PowerShell: `& $D uninstall
  --skill-dir (Get-Location).Path`). Only this binary's exact hooks and matching
  skill link are removed. Telemetry and settings backups remain private/local.
- `legacy/rex/` is historical, uninstalled reference only. No Rex configuration is
  touched or required. Do not install its old Python collector.

## Developer build and tests

Go 1.24+ is a **build-time** dependency. This Mac's local toolchain was downloaded
and hash-verified under ignored `.build-tools/go/`, without system installation.
From the skill directory use `go`, or `.build-tools/go/bin/go` here:

```sh
go test -race -cover ./...
go vet ./...
go mod download all # cache all module licenses required by the packager
go run ./cmd/package
```

Packaging cross-builds six pure-Go binaries (`CGO_ENABLED=0`), writes per-platform
zips and `target/SHA256SUMS`, and collects dependency license notices. Build output
(`bin/`, `target/`, toolchain) stays out of Git. Publish the six ZIPs and
`target/release/SHA256SUMS` as GitHub Release assets; do not substitute automatic
source archives. `target/SHA256SUMS` is the separate local-build manifest covering
both binaries and archives, while the release manifest uses download filenames.
Tests
cover ingestion, real multi-process writes/readers, isolation, task normalization,
layout validation/live replacement, arbitrary roles, rendering safety/geometry,
installer preservation, CLI and fail-open hooks. `tests/pty_smoke.py <binary>` is
an optional **developer-only POSIX/Python** PTY test, not a product dependency.
Cross-builds are not runtime tests on those OS/architectures.

After publishing a release, `python3 tests/release_smoke.py` anonymously downloads
all six ZIPs, checks their release hashes/package contents, and smoke-runs the
host-matching executable with isolated synthetic state. This is developer tooling,
not an end-user Python requirement. It deliberately fails if assets are missing
or inaccessible to a public user.
