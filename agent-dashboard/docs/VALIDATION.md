# Validation matrix · source 0.2.0-dev

Latest checks recorded **2026-11-19 UTC** on **macOS / Darwin arm64 (Apple Silicon)**.
Historical sections below retain their original scope.
Local build toolchain: **Go 1.27.4**, archive SHA-256 checked before extraction
into ignored `.build-tools/go/`; no system-wide toolchain/runtime installed.
Claude Code used for historical integration checks: **2.1.312**.
Distribution builds are unsigned; release assets are not signed/notarized packages.

## Optional Rex adapters (2026-11-19)

These results cover the source/development implementation, **not a public
0.2.0 release**. The published 0.1.0 assets and bundled trusted pins remain
unchanged. No Git commit/push, Release replacement, remote deployment,
application settings edit, or real Claude API invocation was performed.

| Scope | Result / exact boundary |
| --- | --- |
| Go regression | `go test -count=1 -race -cover -json ./...`: **85 top-level tests + 31 subtests passed**; helper entrypoint explicitly skipped in the parent and exercised as real native child processes. `go vet ./...` passed |
| Coverage | Dashboard **90.7%**, CLI **71.7%**, Rex observer **88.2%**, packager **24.2%**; version constants have no executable statements |
| Public interfaces | CLI dispatch/configuration, official CLI executor boundary, `Monitor.Snapshot` / `Ready` / `Close`, `OpenViewer`, layout parsing and public Bubble Tea model/rendering. No private monitor state or private snapshot message is required by the new tests |
| Observation correctness | Whole-pane/whole-record replacement clears omitted fields and removed roots/children; source-qualified IDs isolate identical record IDs across hosts; known zero vs unknown progress; scope/topology/host/instance/sequence failures retain last-known data; reconnect baseline staging; late old/stopped callbacks rejected; snapshot copies are detached; discovery progress does not declare an incomplete source ready |
| Native CLI processes | Actual Go helper subprocesses test child PID identity, cancellation/reaping, bounded stderr/stdout/wire lines and callback rejection. An initial real-process test exposed promoted `bytes.Buffer.ReadFrom` bypassing a writer limit; the final executor does not embed that buffer |
| Official Lua runtime | Opt-in read-only contract test ran in the installed Rex CLI, with local synthetic API functions only. It verifies queued pre-baseline events cannot roll current state backward, removed reports/omitted fields clear, decoded `msg` and zero progress survive, foreign/unselected panes are never queried, missing server-event session context is safe, and process owners do not escape. No synthetic reports are submitted to Rex |
| Local launch ownership | Concurrent opens share an OS lock; repeated opens reuse one viewer; focus/direct argv preserved; exact exited-owned viewer replacement; foreign labels/programs/placements untouched; missing/stale/unverified context falls back conservatively; uncertain and late split results reconcile without duplicate RPCs; durable pending receipts block unsafe retries; absolute root and original error causes are retained |
| Real selected-host read | Native Mac viewer read structured reports from this host and one explicitly selected remote Rex server. The checked snapshot contained **5 local Pi reports + 11 remote Pi/Claude reports**. Counts are records, not inferred agents; live IDs, endpoints, status messages and source bindings remain private/ignored |
| Native Rex UI/lifetime | Right-side packaged Go viewer opened and reused at the same ID; original pane/window focus verified. Compact caller canvas showed all checked reports. Own older viewer quit with `q`; a new packaged observer exited with Ctrl-C; known owned viewer/subscription PIDs were confirmed gone. A fresh native viewer remains open; observed programs and remote configuration were not changed |
| Viewer semantics | Program reports create no Claude lead/session, task, tool or Issue completion. Actual headless Bubble Tea runtime verifies source-buffer scroll survives live reads and clock/report-age-only changes do not generate echoes. Caller source/key filters, inert control text and clipped geometry are covered |
| Terminal-independent regression | **Three native macOS POSIX PTY runs** passed: ordinary canvas with q, ordinary canvas with Ctrl-C, and synthetic Review DAG with effects/navigation/resize. Terminal restoration and collection retention passed; this is not new real-Claude acceptance |
| Bootstrap regression | **19 POSIX + 13 PowerShell tests passed** again. PowerShell is **7.7.0 on macOS**, not Windows/PowerShell 5.1 validation. No download pins were regenerated |
| Distribution | Six **0.2.0-dev** binaries/ZIPs cross-built locally with the embedded observer, new Rex documentation/example, both bootstrap helpers and unchanged pins; exactly one platform binary per archive, no Python/legacy/test/Lua runtime files. All **12** local-manifest and **6** archive-manifest entries verified; native packaged capabilities and isolated synthetic frame passed |
| Independent review | One read-only launcher review completed. It is not a complete independent observer/security audit; no second completed review is claimed |

Actual source servers used Rex **0.1.0-preview.19** locally and
**0.1.0-preview.21** remotely. Observer Lua runs on the **Mac CLI**, not inside
the remote server. Reading a remote Linux server is therefore **not native Linux
or Windows dashboard runtime validation**. No remote files, hooks, scripts or
Rex settings were installed. All non-Mac-arm64 targets remain cross-build-only.
The temporary Python/curses probe is not product code or a shipped runtime.

The CLI does not expose a documented snapshot watermark/readiness barrier to
this adapter. Events invalidate and re-read current full pane reports; periodic
60-second reconciliation explicitly covers startup gaps. No lossless-delivery,
fixed-refresh, exhaustive-agent-telemetry, code-signing, or Issue-completion
claim is made. Full Pi lifecycle/tools/tasks are still not implemented.

Reproduce the extra local Lua check only with an existing Rex CLI/session:

```sh
AGENT_DASHBOARD_TEST_REX="$(command -v rex)" \\
  .build-tools/go/bin/go test -count=1 -race -cover ./...
```

Ordinary tests skip that opt-in test if the environment variable is absent;
portable fake-CLI and real native helper-process contracts still run. Existing
Claude integration evidence below is historical, not rerun for this adapter.

## Agent-managed first-use bootstrap (2026-11-16)

The agent now runs a platform bootstrap before `doctor`: missing binaries are
downloaded automatically from the pinned public 0.1.0 release, checked against
both bundled archive pins and the published manifest, and published atomically
without replacing source files or configuring hooks. Existing executables are
reused offline. Hook configuration still requires approval and is performed by
the agent rather than delegated as shell instructions to the user.

| Scope | Result / exact boundary |
| --- | --- |
| POSIX bootstrap contracts | **19 tests passed** on macOS `/bin/sh`: verified acquisition, offline reuse, changed archive/remote checksum rejection against source pins, malformed/missing/duplicate checksums, missing/duplicate/empty ZIP members, path traversal never extracted, symlink/foreign-path refusal, a racing destination directory, parallel first use, cleanup, spaces and skill-directory symlinks, curl/wget and SHA-256 utility selection, ZIP-capable tar fallback. Linux architectures and Rosetta detection use mocked system utilities, not native Linux/Rosetta execution |
| PowerShell bootstrap contracts | **13 tests passed** under **PowerShell 7.7.0 on macOS**, with only the download boundary replaced by a fixture function in memory. Actual parsing, .NET ZIP/file operations, locking, publication, cleanup, native-architecture environment selection, offline reuse, missing pins, integrity failures, invalid entries, link/foreign-directory refusal, concurrent calls, and reuse of an abandoned empty lock file were exercised. This is **not Windows or PowerShell 5.1 runtime validation** |
| Public POSIX acquisition | Fresh isolated source directory without `bin/`: real anonymous GitHub manifest + Mac ARM64 ZIP download, bundled-pin verification, executable installation, and native `version` startup passed. A second call reused the binary without acquisition output; skill instructions and README remained byte-identical |
| Public PowerShell acquisition | Actual unmodified helper ran in PowerShell on macOS with Windows architecture environment variables: downloaded and verified the public Windows ARM64 ZIP, selected ARM64 despite an AMD64 process variable, and streamed the expected PE member. A second call reused it. The Windows executable was **not run** |
| Go regression | `go test -count=1 -race -cover -json ./...`: **54 top-level tests + 15 subtests passed**; `go vet ./...` passed. Core coverage **91.3%**, CLI **61.7%**, packager **24.2%**. Two new packager tests cover bootstrap/pin inclusion without developer-runtime files and all six platform pins |
| Distribution | Six native-target binaries/ZIPs rebuilt locally. Every archive includes both helpers, bundled pins, and exactly one platform binary; developer tests/Python are excluded. All **12** binary/archive entries in the local build manifest and **6** archive-only release-manifest entries were checked. Non-Mac targets remain **cross-build-only** |

The developer-only PowerShell distribution was downloaded and SHA-256 verified
against its GitHub asset digest into ignored `.build-tools/powershell/`; no
system-wide runtime/package installation was performed. Run the contract tests
from the skill directory with:

```sh
AGENT_DASHBOARD_TEST_PWSH="$PWD/.build-tools/powershell/pwsh" \\
  python3 -m unittest discover -s tests -p 'test_bootstrap*.py' -v
```

The initial POSIX tests failed before `bootstrap.sh` existed, then passed after
implementation. All network/ZIP test fixtures are explicitly synthetic. Public
smokes used temporary source directories under ignored `target/`, cleaned on
exit. No real Claude session, app settings, hooks, or private telemetry were
changed; no GitHub assets were replaced and no commit/push/publish was performed.
These changes do not add Pi telemetry. The original public 0.1.0 asset hashes
remain pinned; local repackaging does not silently change the trusted downloads.

## Release packaging and privacy checks (2026-11-15)

The source-only repository had no GitHub Release assets, so a fresh checkout
could not run the advertised executable. Distribution now explicitly separates
source checkouts from ready-to-use platform skill ZIPs. README download links
cover macOS, Windows and Linux, each in x64/ARM64.

The packager generates a separate `target/release/SHA256SUMS` containing the six
ZIP basenames, suitable for download-directory verification; the existing local
`target/SHA256SUMS` still covers binaries plus archives. A regression test verifies
basename-only checksums and rejects a missing archive. The test first failed
before the helper existed, then passed after the fix.

`go test -count=1 -race -cover -json ./...` and `go vet ./...` passed after this
packaging-only change: **52 passing top-level tests + 15 passing subtests**;
core coverage **91.3%**, CLI **61.7%**, packager **4.7%**. No new runtime behavior
or non-Mac runtime verification is claimed.

The two public source branches were history-rewritten with approval to replace
personal author/committer identities, machine-account paths and historical
operational identifiers. Targeted scans checked every reachable commit on both
branches; third-party license notices and the public repository namespace remain
intact. PNG text/EXIF/time metadata was removed while retaining pixel/color
chunks. This is a targeted privacy check, not an exhaustive security audit or a
guarantee that external clones/caches have been purged.

`tests/release_smoke.py` exercises anonymous GitHub metadata/asset downloads,
requires all six platform ZIPs plus `SHA256SUMS`, verifies every hash and archive
layout, checks packaged Markdown for personal-data patterns, and starts the
host-matching binary with an isolated labelled demo. Before publication it
reproduced the original missing-distribution failure with HTTP 404. Its public
post-publication result is recorded in the Release notes; the source snapshot
itself does not claim a future download succeeded.

## Link focus and observed change echoes (preceding implementation check)

| Scope | Result / exact boundary |
| --- | --- |
| Full suite | `go test -count=1 -race -cover -json ./...`: **51 passing top-level tests + 15 passing subtests**; `go vet ./...` passed. Helper-process parent skip is not counted as a passing test |
| Coverage | Core **91.3%**; CLI **61.7%**; packager **0% automated statement coverage**, exercised through actual six-target packaging |
| Link focus | Directional neighbor/transitive sets, independent paired edges, return self-edges and cycles, return exclusion without deletion, isolated/unknown/unbound nodes, keyboard-only selection, no automatic pan, local clear/scope/sidebar controls |
| Geometry/lifetime | Sidebar at leaf widths ≥82 cells, explicit narrow hint, matching pan limits, fixed cell allocations, CJK/emoji viewport edges, 64-node/256-edge graph; hot-swap preserves focused leaf ID and clears removed node/panel selection |
| Change semantics | Initial/reopen baselines, exact model/tool/task/status row differences, parallel same-name call identities, tool error vs agent failure, returned/Stop vs completion, multi-field batches, duplicate/non-field snapshots do not extend effects, exact 1.2s deadline and static markers |
| Pause/history | Epoch-tagged reads reject pre-pause results after resume, fresh baseline read on resume, read errors retain pending baseline, motion-off/help/history retain static information without replay, filtered and bounded event-history reading anchors |
| Native PTY | **9 macOS PTY runs**: three 160×56 fanout runs with full focus/sidebar/task-echo/motion-off/resume assertions (initial q/Ctrl-C pair plus a packaged-binary Ctrl-C repeat); two default buffer runs with q/Ctrl-C; TDD/Review/no-spec at 140×56; delegation at 120×40. All exercise SIGWINCH resize, clean exit, terminal restoration and collection retention |
| Fuzz smoke | Extended parser/viewer seam now also draws selected/sidebar views: **5 seconds, 2 workers, 46,387 executions**, no failures. Bounded smoke only, not exhaustive fuzz verification |
| Render benchmark | Same-host sequential `GOMAXPROCS=1`, 140×56 synthetic review: unselected **6.93 ms/frame**, focus + sidebar **5.86 ms/frame**; prior HEAD **10.13 ms/frame** in that run. Host load varied markedly between samples, so **no speedup/regression conclusion or FPS guarantee** is claimed. Plain-frame rendering only, not terminal IO |
| Distribution | Six pure-Go target binaries/archives rebuilt and their SHA-256/contents checked; prototypes, local decision tracker and developer test scripts excluded. Non-Mac-arm64 targets are cross-build-only |

This update changes **presentation only**: no hook/schema/storage installation
change, no new real Claude invocation or user transcript access, no remote deployment,
publish, push or app settings edit. PTY field updates use explicitly labelled
synthetic demo tasks through the public `plan` command; they are not real agent
activity. The historical real-Claude checks below were not rerun.

Viewer keys and exact semantics are in [PANELS.md](PANELS.md) and
[DAG.md](DAG.md#link-focus). DAG terminal interactions are documented engineering
defaults following the user's implementation authorization, not claimed as
separately human-accepted keybindings. The remaining roadmap items are not
implemented.

A separate Ghostty-owned TTY was opened at 160×50 for a **labelled synthetic**
interactive preview. A developer-only launcher toggles one demo task every five
seconds through public CLI calls in an isolated temporary state root; q/close
stops its updater and removes that root. This verifies launch/TTY ownership,
not exhaustive GUI-terminal visual or real-Claude acceptance. No preview launcher
or Python dependency is included in the product archives.

## Parent/child round-trip flow (preceding check)

| Scope | Result / exact boundary |
| --- | --- |
| Full suite | `go test -count=1 -race -cover ./...`: **40 top-level tests + 13 subtests** passed; `go vet ./...` passed |
| Coverage | `internal/dashboard`: **88.9%**; CLI: **61.7%**; packager: **0% automated statement coverage**, exercised through actual packaging |
| Opposite directions | Existing version-1 schema accepts separate parent → child delegation and child → parent `return` reporting. Reply does not change acyclic ranks or invent agents/tasks/messages |
| Directional motion | Public layout/snapshot/viewer test checks both edge labels, paired-arrow caption and persistent upward marker, and observes multiple consecutive upward packet steps on the child's return rail |
| Caller examples | Full/compact/standalone AI Hero Review variants declare each worker's report to lead, while retaining separate worker → verdict aggregation. No-spec variants omit Spec in both directions. A small labelled synthetic `delegation.layout.json` illustrates the round trip without AI Hero |
| Terminal checks | macOS PTY: Review, no-spec Review and small delegation fixture, each with q/Ctrl-C exits. The delegation fixture explicitly checks paired-arrow/upward markers and report label; all check packets/heavy focus, navigation/toggle/resize and terminal restoration |
| Live composition | Existing real session's private layout updated with explicit worker-to-lead report declarations; actual actor IDs, task states and collector settings preserved. No new Claude API call is needed to draw the declared return channels |
| Distribution | Six platform binaries/archives refreshed with current direction markers, docs and examples; hashes/contents verified. Other OS/architectures remain cross-build-only |

The packaged native viewer was reopened directly in Ghostty at **140×47**, with
Rex launch-context variables removed and an independent Ghostty-owned TTY. It uses
the same persisted non-demo session previously viewed in Rex. At this later check
the original Rex test session/viewer had already closed: the root's last observed
state is `ended`, with **66 hooks**. No new Claude was started, and opening the
viewer did not add hooks. The opt-in collection flag is not proof of a running
Claude process. Ghostty process/TTY startup is verified; this is not a claim of
exhaustive GUI-terminal visual acceptance or proof of measured messages. The
earlier read-only Claude check remains the telemetry source; no transcripts or
live actor IDs are committed.

## Visual effects and Rex live check (preceding check)

| Scope | Result / exact boundary |
| --- | --- |
| Full Go suite | `go test -count=1 -race -cover ./...` passed: **38 top-level tests + 11 subtests**; `go vet ./...` passed |
| Coverage | `internal/dashboard`: **88.7%**; CLI: **61.7%**; packager: **0% automated statement coverage**, exercised by actual packaging |
| Focus | Native DAG and ordinary Panels switch between heavy highlighted/bold-heading focus and thin rounded unfocused frames without geometry changes |
| Motion paths | Moving heads and short trails stay on the actual declared stroke polylines over 50 frames, including fan-out/merge, dashed, cross-rank and return paths; no fabricated state/progress |
| Motion isolation | Motion message reschedules only its own 100 ms timer, not snapshot IO; state/hook counters unchanged; `a` disables/re-enables packets, `p` freezes animation and ignores in-flight snapshot results |
| Render benchmark | `BenchmarkAnimatedAIHero`: **1,577,709 ns/op (~1.58 ms/frame)**, 966,927 B/op, 4,187 allocs/op on this Apple M2, 140×56 review fixture. Plain-frame rendering only, not color-terminal IO or cross-platform performance |
| Real Rex view | Existing Claude terminal preserved; only viewer replaced. Two captures had different packet coordinates on the same graph. Heavy focus frame, motion off/on, exact paused-screen stability and resume verified. Hook count stayed **65 → 65** across these visual operations |
| Real PTY | Three AI Hero demo layouts at 140×56, each with q and Ctrl-C exit; explicit flow-head/heavy-frame assertions, motion toggle, navigation, resize, terminal restoration and collection retention |
| Artifacts | Six platform binaries/archives refreshed; current docs/examples included and hashes verified. Non-Mac-arm64 targets remain cross-build-only |

### Actual Claude session used for this acceptance

A new local interactive Claude session was opened through Rex at the user's
request, using the installed hooks and `/agent-dashboard` skill. The bounded
read-only audit used two actual parallel check workers, two native tasks and a
private live layout. Observed worker IDs were published consistently into graph
nodes, actor cards, event filters and task owners; both tasks completed and the
workers returned. At the visual acceptance check the root was waiting,
`demo=false`, `active=true`, `finished=false`, with 65 observed hooks.

This confirms live Claude-to-native-DAG binding and observation, **not** full
`/tdd`, `/code-review`, an AI Hero issue implementation or test-result inference.
Verdict stayed unbound, no `finish`/`deactivate` was issued, and at that preceding
acceptance the interactive Claude session remained usable. No repository code changes, commit/push, remote
operation or hook installation/settings edit were delegated to that session.
Live layout files and native transcripts remain private and are not committed.

The later visual upgrade did not require another Claude API call: the existing
last-observed snapshot was sufficient to check motion/focus and prove the frame
clock did not manufacture hooks. Running-node pulse styling was exercised with
synthetic running fixture states; the live workers had already returned and were
not shown pulsing. Edge motion remains explicitly labelled decorative.

## AI Hero workspace adaptation (preceding check)

Presentation examples only: TDD, two-axis review and review without a spec now
compose native DAGs with session/actor/task/event/context Panels. Live templates
have placeholder IDs; separately named `.demo.layout.json` fixtures bind the
isolated synthetic data. No executable workflow behavior or generic demo seed
was changed.

| Scope | Result / exact boundary |
| --- | --- |
| Full Go suite | `go test -count=1 -race -cover ./...` passed: **35 top-level tests + 11 subtests**; process-helper entrypoint behavior unchanged |
| Static checks | `go vet ./...` passed |
| Coverage | `internal/dashboard`: **88.2%**; CLI: **61.7%**; packager: **0% automated statement coverage**, exercised through actual packaging |
| Example bindings | All three live workspaces publish without creating agents/tasks; demo graph/card/filter bindings resolve to the same fixture IDs; demo and live sessions remain isolated |
| Example semantics | TDD loop is red-green (not refactor); no-spec variant removes the Spec node/card/edges/filter; verdict remains unbound caller annotation; standalone/compact graphs match the full live workspaces |
| Geometry | Existing renderer geometry/safety suite exercised every bundled layout, including all new live/demo variants, at 28..160 columns / 8..50 rows |
| Real PTY | **Six macOS PTY runs**: each of TDD, two-axis review and no-spec review at 140×56, with separate q/Ctrl-C exits; styled DAG, navigation, resize, restored terminal and collection retained |
| Plain frame | Review demo rendered at 140×56 with graph, both actor cards, task/event/context Panels, explicit synthetic labels and no unknown actor bindings |
| Packages | All six platform archives refreshed with live/demo/no-spec examples and current docs; archive contents and SHA-256 entries verified |

This did **not** execute TDD or code-review skills, a real AI Hero issue, or a new
Claude end-to-end session. There was no settings change, remote deployment,
publish or push. Non-Mac-arm64 targets remain cross-build-only, not runtime
verified. See `examples/ai-hero/README.md` for bindings and viewport tradeoffs.

## Native DAG integration (preceding check)

Validated on the same macOS arm64 host and Go 1.27.4 toolchain. This update adds
`kind: dag` to the existing native layout/viewer, not a Python runtime path.

| Scope | Result / exact boundary |
| --- | --- |
| Full Go suite | `go test -count=1 -race -cover ./...` passed: **33 top-level tests + 6 subtests** (the process-writer helper entrypoint is skipped in parent runs and exercised in child processes) |
| Static checks | `go vet ./...` passed |
| Coverage | `internal/dashboard`: **88.1%**; CLI: **61.7%**; packager: **0% automated statement coverage**, exercised by actual six-target packaging |
| Graph publication | Native DAG + event/task composition; persisted snapshot; live task-state replacement; invalid graph rollback; layout does not create agents/tasks or complete work |
| Graph validation | Ordinary-edge cycles/self-cycles rejected; return annotations allowed; missing/duplicate IDs, endpoints, edges, unknown fields/styles/colors, invalid bindings and field/node/edge limits rejected |
| Graph rendering | Stable topological order; fan-out/merge; connected source junctions; cross-rank and return rails; edge legend; literal actor/task states, unknown bindings and unbound declarations |
| Viewport and safety | Independent focus/pan/scroll/Home; overscroll saturation; every bundled layout tested at 28..160 columns / 8..50 rows; CJK/emoji/combining glyphs; terminal-control/bidi stripping; corrupt stored graph diagnostic; 64-node/256-edge viewport |
| Fuzz smoke | 5 seconds, 2 workers, **253,440 executions** of the layout-parser/viewer seam; no failures. This is a bounded smoke test, not exhaustive fuzz verification |
| Real PTY | Native DAG fixture in macOS PTY: styled alternate screen, navigation/pan/scroll, focus/pause/help, SIGWINCH resize, separate q and Ctrl-C exits; terminal restored and collection retained |
| Distribution | All six pure-Go binaries rebuilt; all six archives include DAG contract and native examples, exactly one target binary and no Python runtime files; SHA-256 entries verified |
| Existing baseline issue | `review.layout.json` used `teal` but the old palette rejected it. Palette support now includes teal and the earlier DAG colors; existing and new examples all parse |

The initial packager run found uncached transitive-module license sources;
`go mod download all` populated the build-time module cache before packaging.
The packager now reports that prerequisite explicitly if a module directory is
missing instead of returning an empty-path error. No end-user runtime was added.

No new actual Claude run was performed for this presentation-only integration;
the previous real-Claude checks below remain historical evidence, not a fresh
DAG-in-Claude end-to-end run. Linux, Windows and Intel Mac are still **cross-build
only**, not runtime verified. There was no remote deployment, settings change,
publish or push during this update. The old Python DAG script remains a
reference-only prototype and is excluded from runtime packages.

## Prior rectangle-panel baseline

| Scope | Result / exact boundary |
| --- | --- |
| Go tests | `go test -count=1 -race -cover ./...` passed: **25 top-level tests + 6 subtests**; real helper processes included |
| Static checks | `go vet ./...` passed |
| Core package coverage | `internal/dashboard`: **84.0% statements**; CLI package: **57.4%**; packager: **0% automated statement coverage**, exercised by actual six-target packaging |
| Concurrency | 12 actual OS processes ×8 hook writes, with concurrent readers; all 96 recorded, valid snapshots |
| Telemetry semantics | Parallel/cross-actor tool correlation, permissions, result dedup, ordinary failures vs agent failure, background workers across Stop, explicit finish, independent worker turns, API failure and unconfirmed termination |
| Data/privacy | Unopted sessions create no data; isolation/deactivate; malformed/oversize/corrupt hooks fail open; corrupt state not overwritten; per-session bounds; POSIX permissions; arbitrary tool inputs excluded |
| Caller layouts | Live row→column replacement, arbitrary role names, missing actor bindings remain unknown, filters, per-panel scroll/focus, strict validation/rollback, nesting/node bounds; no default workflow |
| Renderer | CJK/emoji/combining marks, cell-width clipping, terminal-control/bidi sanitization, independent geometry tests across 28..160 columns / 8..40 rows; too-small diagnostics |
| Real PTY | macOS PTY with real binary: alternate-screen entry/exit, SGR style output, key navigation/focus/scroll/pause/help, SIGWINCH resize exercise, q and Ctrl-C clean exits, terminal configuration restoration, collection retained |
| Settings installer | Unit tests: exact byte backup, idempotent merge, narrow uninstall, malformed config refusal, other settings/hooks/skills preserved, foreign paths refused; `doctor` never claims liveness |
| Local installation | Actual `~/.claude/settings.json` merged; original bytes backed up; after removing only new handlers from the in-memory comparison, original settings/env/permissions/hooks exactly matched. Existing source symlink reused. `doctor`: all 12 event handlers configured |
| Legacy reference | 24 Python baseline tests passed unchanged. This does **not** imply legacy installation; Rex code remains uninstalled and untouched |

The PTY test ignores only macOS's kernel-managed PENDIN pending-input-redisplay
bit when comparing terminal configuration. Echo, canonical/raw input and all
other captured terminal attributes must restore. This is not a manual visual
acceptance test in every GUI terminal.

## Actual Claude runs (not injected telemetry)

All were tiny, bounded-budget local smoke tests, in private temporary directories
removed after completion. No implementation issue, external deployment, or
user project modification was performed. Transcript persistence was disabled;
no user transcripts/credentials were added to this repository.

1. **Bash integration**: fresh Claude with user settings + scoped test hooks;
   one actual Bash invocation → **6 hooks**, one completed tool, zero errors,
   SessionEnd observed, `finished=false`.
2. **Native tasks and subagent**: actual TaskCreate/TaskUpdate/TaskList and one
   general-purpose subagent with its own Bash call → **18 hooks**, native task
   retained as completed, lead tools=4, worker tools=1 and status `returned`, no
   active tools, zero errors, `finished=false`. This confirms real subagent tool
   coverage and the recognized native task payload on this installed version.
3. **Skill invocation through real installed global hooks**: fresh Claude invoked
   `/agent-dashboard` with a caller-provided private layout → correct real
   `${CLAUDE_SESSION_ID}`, caller layout name and exact two leaf IDs persisted,
   **5 observed hooks after opt-in**, `finished=false`. Skill substitution,
   executable discovery, start, caller layout publication and loaded global
   hooks were exercised.

An initial deliberately isolated run with `--setting-sources ''` could not
resolve local login configuration; it produced a real StopFailure/SessionEnd,
zero tool calls, and `finished=false`. Retrying with the user's settings source
succeeded. That failed attempt is not counted as a successful execution test.

Models can stay unknown in no-transcript-persistence smoke tests; actual model
attribution was not inferred. Model scoping/observed model-switch behavior is
covered by synthetic local transcript tests, not claimed as real multi-model
routing validation. Full AI Hero issue execution, permission-dialog interaction,
force-kill recovery and long-duration observation are **not** end-to-end verified.

## Cross-platform artifacts vs runtime verification

All six binary formats were produced with `CGO_ENABLED=0`; each platform zip
contains SKILL.md, docs, AI Hero examples, dependency notices and one executable.
`target/SHA256SUMS` covers binaries and archives. Packaging contents/hash checks
and archive extraction/local binary startup were checked on this Mac.

| Target | Binary/archive | Actual runtime |
| --- | --- | --- |
| macOS arm64 | built | **tested here**: CLI, PTY and real Claude |
| macOS amd64 | cross-built Mach-O | **not run**; no Intel Mac/Rosetta validation |
| Linux amd64 | cross-built static ELF | **not run** |
| Linux arm64 | cross-built static ELF | **not run** |
| Windows amd64 | cross-built PE console executable | **not run** |
| Windows arm64 | cross-built PE console executable | **not run** |

No remote host was used/deployed to; cross-builds are not OS runtime tests.
Native Windows symlink/junction privileges, ACLs, PowerShell UX and actual
console behavior still need runtime acceptance. No package signing,
notarization, Gatekeeper/SmartScreen acceptance or public download is claimed.

## Reproduce locally

From `agent-dashboard/`:

```sh
.build-tools/go/bin/go test -count=1 -race -cover ./...
.build-tools/go/bin/go vet ./...
.build-tools/go/bin/go mod download all
.build-tools/go/bin/go run ./cmd/package
python3 tests/pty_smoke.py bin/darwin-arm64/agent-dashboard
python3 tests/pty_smoke.py bin/darwin-arm64/agent-dashboard --ctrl-c
python3 tests/pty_smoke.py bin/darwin-arm64/agent-dashboard --layout=examples/dag/fanout.layout.json
python3 tests/pty_smoke.py bin/darwin-arm64/agent-dashboard --layout=examples/dag/fanout.layout.json --ctrl-c
.build-tools/go/bin/go test ./internal/dashboard -run '^$' -fuzz '^FuzzDAGLayoutViewer$' -fuzztime=5s -parallel=2
# Repeat for tdd / review / review-no-spec; add --ctrl-c for the alternate exit:
python3 tests/pty_smoke.py bin/darwin-arm64/agent-dashboard --layout=examples/ai-hero/review.demo.layout.json --size=140x56 --effects
```

The Python PTY runner is a developer-only POSIX test, not required by the product.
`demo --json @examples/ai-hero/tdd.demo.layout.json` is explicitly synthetic; it must
not be described as running the AI Hero workflow or as live Claude telemetry.
