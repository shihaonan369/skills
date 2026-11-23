# Agent Dashboard · composable TUI skill

**A presentation capability, not a workflow. The caller owns composition.**

A terminal-independent, Claude Code-first TUI module with self-contained
executables. It supplies **Panels** (`session`, `agent`, `agents`, `tasks`,
`events`, `text`, `dag`) and `row`/`column` composition. Version 0.2.0 also adds
optional Rex `programs`/`sources` inspectors and shared typed bindings across
all record panels and DAGs. The calling skill/agent owns
layout, role assignment and workflow; hooks supply last-observed real telemetry.
There is no screenshot workflow template or hard-coded AI Hero pipeline.

**Version 0.2.0** adds optional Rex adapters, shared observation bindings and
forward-only DAGs. Version 0.1.0 assets remain unchanged and lack the Rex
capabilities. Bootstrap reuses existing executables without upgrading them;
check `capabilities` before using [Optional Rex integration](#optional-rex-integration).
Remove retired `style: return` edges explicitly from caller layouts when migrating.

**Version 0.3.0** adds shared live membership across every panel shape and
agent-owned semantic scopes, gated by `dynamic-layout-v1` / `agent-scope-v1`.
Prefer the same native background subagent for initial and ongoing classification;
Claude Code workers can own Monitor with the quiet `scope-watch` event source.
Older 0.2.0 assets lack these features. Existing binaries are not upgraded. Read
[Dynamic membership and classification](docs/DYNAMIC.md) for setup and the
unverified native-harness acceptance boundary. Source bootstrap pins follow the
[two-phase release lifecycle](#release-pin-lifecycle).

**Source-only additions:** this checkout identifies as **0.4.0-dev** and adds
`file-data-bindings-v1`: local JSON data sources with shared text/title/DAG-label
bindings, numeric/progress/countdown formatting and an optional atomic
`data-write` producer. Published 0.3.0 assets and bootstrap pins lack this
capability. See [External file data](docs/DATA.md); existing installations remain
unchanged and are not silently upgraded.

## First use

Install/discover the skill in Claude Code and invoke `/agent-dashboard` with your
current task and desired composition. **The agent handles runtime setup**; a
source checkout no longer requires the user to download or compile a binary.

1. The agent runs `bootstrap.sh` on macOS/Linux/WSL or `bootstrap.ps1` in native
   Windows PowerShell. The helper selects the host OS/native architecture,
   including Apple Silicon and Windows ARM64 under a translated shell.
2. An existing nonempty executable is reused without network access. Otherwise,
   a source checkout acquires the reviewed release pinned in its helper and
   `bootstrap.sha256` from this repository's GitHub Releases,
   verifies the archive against bundled `bootstrap.sha256` and the published
   checksum manifest, and installs only the matching binary under `bin/`.
   Prebundled ZIPs use their included executable only: a missing/mismatched binary
   is an incomplete/wrong-platform package, never permission for a fallback download.
3. The agent runs `doctor`. Before changing Claude settings/hooks, it asks for
   approval; once approved, the agent runs `install` and rechecks configuration.
   Start a fresh Claude session after the hook change, then invoke the skill
   again. Existing sessions are not assumed to have reloaded hooks.
4. The agent attaches the actual Claude session and supplies the caller's
   composition. It opens the viewer through an available authorized terminal
   adapter, or reports the normal viewer command. Version 0.2.0 can
   create/reuse a right-side Rex pane automatically while preserving focus.

No global PATH change, package manager, compiler, Python, or Node installation
is performed. POSIX bootstrapping uses existing system utilities: `sh`, `curl`
or `wget`, `shasum` or `sha256sum`, and `unzip` or ZIP-capable `tar` (such as
bsdtar). Native Windows uses built-in PowerShell 5.1+ and .NET, not those POSIX
utilities. A minimal Linux image missing these utilities, a read-only skill
location, blocked network access, or execution policy can block first use; the
agent reports the actual blocker rather than asking for manual downloads or
bypassing policy. A platform ZIP with a working binary needs no download tools.

The helper does not run downloaded code, update existing binaries, modify source
docs/layouts, or configure hooks. Temporary files stay under ignored `bin/` and
are cleaned on success/failure. Concurrent calls serialize and publish only a
complete file without replacement. POSIX interruption by SIGKILL can leave a
lock directory; a timed-out caller reports it rather than guessing that another
process is dead. Windows uses an OS sharing lock; its empty lock file may remain
and does not block later runs.

Claude hooks remain **Claude Code-specific**; discovering this skill in Pi does
not supply Claude variables or full Pi lifecycle/tools/tasks. Version 0.2.0
can observe basic Pi **program reports through Rex**, without installing
Claude hooks. That separate boundary is described under Optional Rex integration.

## Download

Prebundled platform ZIPs remain available for offline transfer or manual setup;
the agent normally acquires a missing runtime through the first-use flow above.
Get a ready-to-use platform ZIP from
[GitHub Releases](https://github.com/shihaonan369/skills/releases/latest).
Packages are available for **macOS, Windows, and Linux**, each in **x64 and ARM64**.
Each ZIP contains this skill directory, one matching executable, docs, examples,
and third-party notices. No end-user compilation or extra runtime is required.

Choose the uploaded `agent-dashboard-0.3.0-<os>-<arch>.zip` asset, **not** the
automatic “Source code” archives. Source checkouts omit `bin/` and `target/`.
Extract to a permanent directory before installing hooks; keep existing unrelated
skills intact. Platform verification and unsigned-build caveats are covered under
[Safety, distribution and limitations](#safety-distribution-and-limitations).

### Version 0.3.0

| Operating system | x64 / AMD64 | ARM64 |
| --- | --- | --- |
| macOS | [Intel Mac ZIP](https://github.com/shihaonan369/skills/releases/download/agent-dashboard-v0.3.0/agent-dashboard-0.3.0-darwin-amd64.zip) | [Apple Silicon ZIP](https://github.com/shihaonan369/skills/releases/download/agent-dashboard-v0.3.0/agent-dashboard-0.3.0-darwin-arm64.zip) |
| Windows | [Windows x64 ZIP](https://github.com/shihaonan369/skills/releases/download/agent-dashboard-v0.3.0/agent-dashboard-0.3.0-windows-amd64.zip) | [Windows ARM64 ZIP](https://github.com/shihaonan369/skills/releases/download/agent-dashboard-v0.3.0/agent-dashboard-0.3.0-windows-arm64.zip) |
| Linux | [Linux x64 ZIP](https://github.com/shihaonan369/skills/releases/download/agent-dashboard-v0.3.0/agent-dashboard-0.3.0-linux-amd64.zip) | [Linux ARM64 ZIP](https://github.com/shihaonan369/skills/releases/download/agent-dashboard-v0.3.0/agent-dashboard-0.3.0-linux-arm64.zip) |

[SHA-256 checksums](https://github.com/shihaonan369/skills/releases/download/agent-dashboard-v0.3.0/SHA256SUMS)
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
Full remote Claude lifecycle telemetry still requires the collector there.
Optional Rex mode can read existing selected-host program reports without a
remote skill installation; it is not full remote lifecycle telemetry.

### Manual setup (optional)

The agent performs these steps after approval during normal skill use. The
commands below are reference material for manual administration, not required
user actions.

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
There is no separate orchestrator. Version 0.2.0 optionally supports
`open --session '<Claude-ID>'` for a verified Rex split/reuse; normal `tui` remains
terminal-independent.

Native Windows PowerShell, from this directory:

```powershell
$D = '.\bin\windows-amd64\agent-dashboard.exe'
& $D install --skill-dir (Get-Location).Path
& $D doctor
```

Directory symlinks on Windows require Developer Mode or suitable privileges. If
symlink creation is unavailable, the agent can create the named skill junction
with approval using `cmd /c mklink /J` and properly quoted absolute source/target
paths, then rerun install; an existing link/junction resolving to this source is
reused. Report a policy/privilege blocker if that is not permitted. The tool
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
forward-only fan-out/merge and cross-rank edges inside the same TUI;
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

## Optional Rex integration

Two independent adapters and a shared record-binding capability are available
in version 0.2.0, gated by
`capabilities` (published **0.1.0 does not have them**):

- `rex-local-open-v1`: `open --session '<Claude-ID>'` creates/reuses an owned
  right-hand pane with focus preserved, or explicitly returns normal terminal
  viewer arguments when local Rex context cannot be verified.
- `rex-observer-v1`: `rex-sources` resolves only selected hosts into pinned
  configuration; `rex-watch` reads actual structured program reports through
  persistent official CLI connections, with events, reconnects and last-known
  source health. `open --sources @private.json --layout @layout.json` presents
  that observer using the same optional launcher.
- `observation-bindings-v1`: one typed binding interface lets `session`,
  `agent`/`agents`, `tasks`, `events` and DAG nodes display Rex observations.
  Cards/work rows require explicit program selection and retain program-report
  labels; events are bounded local observation changes with baseline/gap
  boundaries. See [shared observations and bindings](docs/OBSERVATIONS.md).

The agent performs source resolution, private file creation, opening and
verification. No remote deployment, hooks/config edits, screen parsing, agent
role inference, or additional runtime is involved. Program `done` is not Issue
completion; missing progress stays unknown. Pi/Claude reports are supported when
those programs actually report to Rex; full Pi agent telemetry is not added.
Claude lifecycle and Rex program observation remain separate modes, not a
fabricated merged agent model.

Read **[Rex adapter contract and agent procedure](docs/REX.md)** for exact
commands, opt-in scopes, capability/distribution gates, ownership and ambiguous
launch recovery. The **[small caller example](examples/rex/README.md)** supplies
presentation only; the caller can arrange every panel shape with explicit
typed bindings, including program-bound DAGs and observation history. `2`/`3` show program/source diagnostics in this mode.

## Dynamic panels and monitoring scope (0.3.0+)

The caller can declare collections once and have new observed members populate
DAGs, repeated cards, lists, annotations and split subtrees automatically. One
shared projection supplies every shape, with stable IDs and explicit bounds;
static layouts remain static. See [dynamic examples](examples/dynamic/README.md).

For semantic requests such as “show implement agents”, the **initiating agent**
plans scope and delegates **initial and subsequent judgements to the same native
background subagent** where available. In Claude Code, prefer worker-owned Monitor
with the deduplicated `scope-watch` stream; the worker handles first candidates,
later arrivals and bounded watch renewal without routing raw events through the
main conversation. The dashboard only queues requests and applies decisions; it
never starts Claude/Pi or calls an LLM. Fall back to background long polling,
then foreground subagent batches, and only then main-agent checkpoints. Disclose
scheduling gaps; unknown identities stay excluded. Read
[the complete scope/worker contract](docs/DYNAMIC.md).

## External variables and countdowns (source-only)

A caller can bind text values, every leaf's title, and static/dynamic DAG labels
to approved local JSON fields. One shared reader/resolver/formatter supplies all
surfaces; data refreshes do not rewrite layouts or create telemetry. Optional
`data-write` atomically publishes private snapshots with freshness metadata.
Remaining seconds come from the script; only an explicit deadline opts into the
local display clock. Missing, malformed and expired data remain unknown/stale.
Read [the data contract](docs/DATA.md) and
[countdown example](examples/data/README.md). No model or background classifier
is involved in ordinary value updates.

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
relationship scope, `i` opens optional relation
details, and `c` clears selection. These inspect declarations, not traffic.
Observed field changes briefly emphasize only the affected row, with a static
marker/summary retained after the effect; `x` clears that local summary. See the
[Panel contract](docs/PANELS.md#observed-change-echoes) for baseline/history rules.
Declared edges have decorative moving
packets and short trails (not measured traffic); `a` toggles motion, `p` freezes
the displayed snapshot and animation while hooks continue collecting. Embed the
`dag` object into any caller layout; all edges participate in an acyclic forward
graph. Retired `style: return` edges are rejected rather than silently rewritten.

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
python3 -m unittest discover -s tests -p 'test_bootstrap*.py' -v
go mod download all # cache all module licenses required by the packager
go run ./cmd/package
```

Packaging cross-builds six pure-Go binaries (`CGO_ENABLED=0`), identifies them
using `internal/buildinfo.Version` (currently `0.4.0-dev`), writes per-platform zips and `target/SHA256SUMS`, and collects dependency license notices. Build output
(`bin/`, `target/`, toolchain) stays out of Git. Publish the six ZIPs and
`target/release/SHA256SUMS` as GitHub Release assets; do not substitute automatic
source archives. `target/SHA256SUMS` is the separate local-build manifest covering
both binaries and archives, while the release manifest uses download filenames.
Tests
cover ingestion, real multi-process writes/readers, isolation, task normalization,
layout validation/live replacement, arbitrary roles, rendering safety/geometry,
installer preservation, CLI and fail-open hooks. Rex tests cover source scope,
protocol isolation, whole-report replacement, native CLI subprocess bounds and
cleanup, local launch idempotence/uncertain outcomes, and caller panel rendering.
An opt-in read-only Lua contract test uses the existing official Rex runtime:
`AGENT_DASHBOARD_TEST_REX="$(command -v rex)" go test -race ./internal/rexwatch`.
All API data in that contract fixture is synthetic and never submitted to Rex. `tests/pty_smoke.py <binary>` is
an optional **developer-only POSIX/Python** PTY test, not a product dependency.
Cross-builds are not runtime tests on those OS/architectures.

Bootstrap contract tests use synthetic ZIPs and mocked downloads, without touching
real Claude settings or sessions. The PowerShell suite requires an existing
developer PowerShell executable (`pwsh`/`powershell`, or an absolute path via
`AGENT_DASHBOARD_TEST_PWSH`); it is skipped explicitly if unavailable. Cross-host
PowerShell tests exercise .NET file/ZIP behavior with simulated Windows
architecture variables; they are not native Windows or PowerShell 5.1 runtime
validation. Python/PowerShell developer tooling is not shipped as a product
runtime dependency.

### Release pin lifecycle

A platform ZIP cannot include its own trusted archive hash. Build, validate and
publish the six versioned ZIPs plus their archive-only `SHA256SUMS` first. Then
verify public downloads against those local artifacts and commit the new
bootstrap version constants and reviewed pins to source main in a follow-up.
Local packaging never regenerates trusted pins or overwrites a published release.
Release tags/ZIPs retain the preceding source-acquisition pins; the generated
`.prebundled` marker ensures platform packages use only their matching included
binary and fail if it is missing, rather than downloading an older fallback.
For a fresh source checkout with current acquisition pins, use source main;
a tag's source archive is not the ready-to-run platform distribution.

Existing binaries are still reused offline, not upgraded. Obtain approval for
replacing an installed skill/binary and preserve unrelated skills/settings.
Never derive first-use pins from an unverified download or silently retarget
to `latest`. Old tags and assets remain untouched.

After publishing a release, `python3 tests/release_smoke.py` anonymously downloads
all six ZIPs, checks their release hashes/package contents, and smoke-runs the
host-matching executable with isolated synthetic state. This is developer tooling,
not an end-user Python requirement. It deliberately fails if assets are missing
or inaccessible to a public user.
