# Validation matrix · version 0.4.0

Latest checks recorded **2026-11-23 UTC** on **macOS / Darwin arm64 (Apple Silicon)**.
Historical sections below retain their original scope.
Local build toolchain: **Go 1.27.4**, archive SHA-256 checked before extraction
into ignored `.build-tools/go/`; no system-wide toolchain/runtime installed.
Claude Code used for historical integration checks: **2.1.312**.
Distribution builds are unsigned; release assets are not signed/notarized packages.

## Version 0.4.0 release preparation (2026-11-23)

The user authorized both a public 0.4.0 release and the local installed-binary
upgrade after the read-only two-axis review. Feature commit:
`09655e277bdf4b10d583712aeb773d8d46b262a0`. Review found **0 documented-standard
violations, 3 non-blocking maintenance heuristics and 0 confirmed spec defects**.
No unrelated refactoring was included. The source-only record below is historical.

- Fresh release regression passed **121 Go top-level tests + 111 subtests**, with
  race/coverage enabled, **2 conditional skips**, and `go vet`. Coverage: dashboard
  **90.1%**, CLI **72.6%**, Rex observer **88.7%**, packager **25.3%**.
- All **20 POSIX + 14 PowerShell** bootstrap tests passed; PowerShell ran on macOS,
  not native Windows/5.1. The optional official Rex Lua contract passed using
  synthetic API data and no telemetry submission.
- **Five source macOS PTY runs** passed: external data live refresh/pause/stale/
  corruption/recovery, dynamic scope-watch, dynamic polling, all-panel Rex, and
  fanout effects/controls/resize. Native/reported telemetry remained truthful;
  scope decisions and observation fixtures were explicitly synthetic.
- Distribution preparation uses an isolated ignored workspace for all six
  0.4.0 platform ZIPs. Archive verification covers **12 local + 6 archive-only**
  hashes, one matching binary, current docs, **20 JSON examples** and absence of
  development runtime/test/source/legacy files. The extracted native package is
  checked with offline bootstrap, exact version/six capabilities, all layouts,
  synthetic rendering and the external-data PTY before publication.
- Publication is two-phase: authenticated draft asset verification, then public
  anonymous download checks, and only then a separate source-main bootstrap
  repin. The tag/ZIPs retain verified 0.3.0 acquisition pins; `.prebundled` forbids
  older-version fallback. Published 0.1.0/0.2.0/0.3.0 assets stay immutable.
- The authorized local upgrade replaces only the native executable after public
  verification, retaining a rollback backup. It does not change Claude settings,
  hooks, caller layouts, Rex configuration or running-session data. Existing
  processes are not assumed to hot-reload a replaced executable.

Only **macOS arm64** runtime is verified; the other five targets are cross-builds.
Builds remain unsigned/not notarized. Real native background-subagent / Monitor /
LLM classification and watch renewal remain **unverified end-to-end**; this
numeric-data feature does not require them. Final publication, public download,
source-repin and local-upgrade outcomes are recorded in the follow-up source-main
entry rather than retroactively altering immutable release assets.

## External data bindings, source-only 0.4.0-dev (2026-11-23)

Implemented shared file snapshots, JSON-pointer value bindings, safe formatting,
freshness/failure diagnostics, text values, all nine leaf titles, static and
dynamic DAG labels, and an optional native atomic `data-write` producer. Values
are not native telemetry, classification evidence or inferred entity state.
No commit/push/release or installed-binary/settings/hooks/private-layout/Rex
changes are part of this implementation. Stable bootstrap pins remain 0.3.0.

| Scope | Result / exact boundary |
| --- | --- |
| Red/green integration | Initial shared text/title/DAG test failed with `json: unknown field "data_sources"` before production code; passed after implementation. The test also proves subsequent rendering uses cached snapshots, refreshes need no layout rewrite, and raw values are not serialized into native state |
| Go regression | `go test -count=1 -race -cover -json ./...`: **121 top-level tests + 111 subtests**, **2 conditional skips**; `go vet ./...` passed. Coverage: dashboard **90.1%**, CLI **72.6%**, Rex observer **88.7%**, packager **25.3%** |
| Shared-value coverage | All nine leaf titles, repeated text/dynamic DAG templates, static DAG edges, native and Rex states; exact text/numbers, percent/bar, pointers/arrays/escapes, remaining seconds versus explicit deadline, freshness/clock skew, unavailable/corrupt/oversized/symlink/directory sources, invalid updates and recovery, unchanged serialized native/Rex data and caller declarations |
| Real external producers | The documented POSIX shell and PowerShell countdown snippets each ran with a shortened two-second budget against the native executable and published a final zero. PowerShell ran on macOS, not native Windows/5.1 |
| New native PTY | Live file publication updates text/title/DAG in the real viewer while native session state stays byte-identical. Missing-source recovery, pause/freshness-clock freeze, resume, stale/corrupt diagnostics, recovery, resize, q/terminal restore and cleanup passed. Explicitly synthetic data, not agent/harness telemetry |
| Regression PTY | Three further macOS runs passed: dynamic scope-watch with synthetic decisions, all-panel synthetic Rex observations, and DAG fanout effects/controls/resize |
| Bootstrap and Rex Lua | All **20 POSIX + 14 PowerShell** bootstrap tests passed; existing optional official Rex Lua contract passed with synthetic API data. No remote telemetry submission or configuration change |
| Development packages | Six **0.4.0-dev** platform ZIPs built in an isolated ignored workspace; **12 local + 6 archive-only** hashes verified. Packages include DATA.md, countdown guidance/template and **20 JSON examples**, with one matching binary and no development runtime/test/source/legacy files |
| Extracted native package | Offline prebundled bootstrap reuse with network/hash/ZIP tools absent from PATH, exact development version and six capabilities, all 20 layouts (caller-template file paths substituted in temporary copies), synthetic plain frame and the new external-data PTY passed |

Only **macOS arm64** runtime is verified. The other five targets are cross-builds,
not native Linux/Windows acceptance; Windows ACL privacy and PowerShell 5.1 remain
unverified. Builds are unsigned/not notarized. Local-file refresh is best effort,
not a hard real-time filesystem guarantee. This feature needs no model or native
subagent; it does not change the previous unverified real background-worker /
Monitor / LLM acceptance boundary. No public artifacts were published or repinned.
Logs and isolated package workspace pointers are under ignored `target/` with
`external-data-*` names.

## Published 0.3.0 and source-main pins (2026-11-21)

Published [Agent Dashboard v0.3.0](https://github.com/shihaonan369/skills/releases/tag/agent-dashboard-v0.3.0)
at **2026-11-21 14:39:54 UTC** and verified it resolves as Latest. Release-code
commit: `cf8a442b4f0d03354518b4fbb763b14487af85ec`. Annotated tag:
`agent-dashboard-v0.3.0`, tag object `993cf2e158e82e5927813292731df44d6ac6a749`.

- The draft contained exactly **six platform ZIPs + archive-only SHA256SUMS**.
  All seven asset sizes/server SHA-256 digests matched reviewed local files;
  authenticated draft downloads also matched every local asset.
- After publication, `tests/release_smoke.py --tag agent-dashboard-v0.3.0
  --version 0.3.0 --expected-checksums <reviewed-local-manifest>` passed anonymous
  public downloads of all six ZIPs and the manifest. The public manifest matched
  local bytes exactly. Checks covered platform membership, package paths/docs,
  privacy patterns and native startup, scope-watch closure, five capabilities,
  all **19** bundled examples and isolated synthetic rendering.
- Only after that public verification, source-main bootstrap constants and all
  six trusted archive hashes were repinned to **0.3.0** in a separate follow-up.
  Release tags/ZIPs retain their preceding acquisition pins and were not rebuilt,
  moved or overwritten. Prebundled helpers reuse the matching included binary.
- A fresh source-like directory with spaces downloaded the public release
  anonymously through the actual helper, verified its checksums, installed only
  the native executable atomically, and matched the reviewed package binary.
  Version/capabilities passed; offline reuse succeeded with HTTPS/hash/ZIP tools
  absent from PATH. Locks/staging files were removed and instructions/pins were
  unchanged by setup. The disposable installation was removed afterward.
- Post-repin regression again passed **113 Go top-level tests + 74 subtests** with
  race/coverage enabled and two conditional skips, `go vet`, and all **20 POSIX +
  14 PowerShell** bootstrap tests. PowerShell ran on macOS, not native Windows.
- Original **0.1.0 and 0.2.0 release/asset IDs, names, sizes and digests** were
  compared to pre-release snapshots and are unchanged.

These checks do **not** establish native background-subagent/Monitor/LLM
integration or foreign-platform runtime support; the acceptance boundaries in
the preparation section below still apply. No installed production binary,
application settings/hooks, private layout or Rex configuration was changed.

## Version 0.3.0 release preparation (2026-11-21)

The user authorized commit, push and release after the two-axis review. This
release includes shared dynamic membership, agent-owned semantic scopes, and a
worker-owned Monitor procedure for both initial and subsequent classification.
Review found no confirmed spec defects or documented-standard violations; two
non-blocking maintainability heuristics remain (duplicate selection notices and
repeated string-enum validation). No unrelated refactor was bundled into release
preparation. Earlier source-only sections below retain their historical scope.

| Scope | Result / exact boundary |
| --- | --- |
| Go regression | `go test -count=1 -race -cover -json ./...`: **113 top-level tests + 74 subtests passed**, **2 conditional test skips**. `go vet ./...` passed. Coverage: dashboard **90.0%**, CLI **71.8%**, Rex observer **88.7%**, packager **25.3%** |
| Bootstrap | **20 POSIX + 14 PowerShell tests passed**. PowerShell exercised the existing developer runtime on macOS; native Windows/PowerShell 5.1 remain unverified |
| Official Rex Lua contract | Opt-in observer test passed in the existing official Lua runtime with synthetic API data; no telemetry was submitted and no Rex configuration changed |
| Source macOS PTY | **Six synthetic runs** passed on the 0.3.0 binary: dynamic mailbox scope-watch stream, dynamic scope-poll fallback, all-panel Rex observation, forward delegation q and Ctrl-C exits, fanout effects/controls/resize. Included paused discovery, pending/include/exclude transitions, terminal restoration and child cleanup |
| Distribution | Six **0.3.0** platform ZIPs built in an isolated ignored workspace. All **12** local and **6** archive-only manifest hashes verified. Archives include one platform executable, `.prebundled`, current docs and all **19** JSON examples; no Go/Lua/Python/test/legacy runtime files |
| Extracted macOS package | Passed offline bootstrap with HTTPS/hash/ZIP utilities absent from PATH, exact version and five capabilities, all 19 layouts, isolated synthetic rendering and a further scope-watch PTY run |
| Release integrity | Public publication and source-main repinning are separate subsequent steps. Release ZIP helpers retain the preceding verified 0.2.0 acquisition pins; `.prebundled` prevents any old-version fallback. Existing 0.1.0/0.2.0 assets must remain unchanged |

**Real native background-subagent + Monitor + LLM classification and actual
watch-renewal delivery remain unverified.** Monitor mechanics were checked in
current official documentation; classification/UI tests use synthetic replies.
Only macOS arm64 was runtime-tested. All other targets are **cross-build-only**;
packages are unsigned and not notarized. No installed skill/binary, application
settings/hooks, private layout or observed remote host was changed.

## Background-first initial and ongoing classification (2026-11-21)

The caller protocol now delegates **both initial and later semantic decisions to
one native background subagent**, preferably using its own Claude Code Monitor.
The main conversation supplies scope/context and starts/stops the worker, without
an initial classification pass. A new `scope-watch` command streams pending and
changed requests; it is a local event source, not a classifier or dispatcher.
Monitor availability, ownership, bounded deadlines/renewal, cancellation and
background-poll/foreground/main-agent fallbacks are documented in DYNAMIC.md.

| Scope | Result / exact boundary |
| --- | --- |
| Go regression | `go test -count=1 -race -cover -json ./...`: **113 top-level tests + 74 subtests passed**, **2 conditional test skips** (plus the buildinfo package with no tests). `go vet ./...` passed. Coverage: dashboard **90.0%**, CLI **71.8%**, Rex observer **88.7%**, packager **25.3%** |
| Watch transport | Public mailbox/watch interfaces verified initial and later subagent-origin synthetic replies updating graph/cards/lists, fingerprint dedup across atomic writes, unknown-to-changed identity requeue, restart replay, all **71** initial candidates as **64 + 7** requests, empty startup without idle repeats, deadline without mailbox closure, context cancellation, output/read failures and duration bounds. CLI verified JSON-line encoding and immediate closed-stream exit |
| Red/green evidence | The first watch test failed with the missing public WatchScope API; the CLI test failed on the missing duration flag. Both passed after implementation. Ignored logs: `target/scope-monitor-red.log`, `target/scope-monitor-cli-red.log`, `target/scope-monitor-go-tests.json` |
| Native macOS PTY | **Two synthetic runs** passed on the newly built binary: one long-lived scope-watch subprocess delivered initial, later and paused-view candidates, synthetic replies updated the canvas, and closure emitted a final event and reaped the stream; the existing bounded scope-poll path also passed. Both checked exclusions, resize/controls, terminal restoration and synthetic Rex-child cleanup |
| Platform builds | Six current-source binaries built under ignored `target/scope-monitor-builds/`. macOS arm64 runtime exercised; Darwin amd64, Linux arm64/amd64 and Windows arm64/amd64 remain **cross-build-only** |
| Documentation verification | Official Claude Code Monitor/subagent references checked through Context7 and public docs. Installed CLI reports **2.1.312**; this does **not** establish native tool exposure, background-event delivery or real LLM classification in the initiating harness |

**No real native background subagent, Monitor invocation, watch renewal event or
LLM judgement was run in this update.** PTY worker replies are explicit fixtures;
`origin:subagent` records supplied provenance, not authenticated proof of a real
subagent. End-to-end harness acceptance remains unverified. No application
settings/hooks, installed binaries, bootstrap pins, private layouts or remote
hosts were changed; nothing was committed, pushed, published or deployed.
Earlier 0.3.0-dev ZIPs recorded below predate scope-watch and the revised worker
protocol; they were not rebuilt or revalidated as current distribution assets.

## Dynamic membership and agent-owned scopes (2026-11-20)

Source 0.3.0-dev adds `dynamic-layout-v1` and `agent-scope-v1`. One shared
selection/materialization layer expands all panel shapes and row/column subtrees.
An optional private mailbox queues new candidates and accepts fingerprinted
include/exclude/unknown decisions from the initiating agent or its native
subagent. No dashboard-owned model client, Claude/Pi subprocess, agent dispatcher,
workflow inference or remote service was added.

| Scope | Result / exact boundary |
| --- | --- |
| Go regression | `go test -count=1 -race -cover -json ./...`: **109 top-level tests + 74 subtests passed**, with **2 conditional skips**. `go vet ./...` passed |
| Coverage | Dashboard **89.9%**, CLI **71.3%**, Rex observer **88.7%**, packager **25.3%** |
| Shared membership | Native role/state selectors, source-qualified program/source selectors, all nine leaf shapes and both split kinds, item-binding scope/ambiguity validation, empty/removal behavior, stable focus and removed navigation-state cleanup passed through public layout/observation/viewer interfaces |
| Semantic scopes | Real private file/OS-lock/atomic-write transport tested with synthetic classifier replies. Pending/unknown stay excluded; one decision updates DAG/cards/lists; metadata/new-dispatch invalidation, stale-batch rollback, concurrent replies, bounded long-poll/stop, symlink refusal, malformed input, provenance and no tool-input leakage passed. Classification precedes display limits: all **71** candidates were handled with a one-member display limit |
| Native macOS PTY | **5 synthetic runs**: dynamic scoped Rex graph/cards/lists/program/source/annotation canvas; forward delegation q/Ctrl-C; fanout controls/effects/resize; existing all-panel Rex observer. New records queue automatically, remain excluded until a synthetic include, and excluded records never enter the canvas. Scoped discovery continues while paused, while displayed membership stays frozen. Terminal restoration, resume, child reaping and stop protocol passed |
| Pause regression evidence | The new PTY check first failed when scoped discovery stopped with the paused viewer, then passed after keeping observation refresh/mailbox discovery active under the existing paused-snapshot epoch guard |
| Bootstrap/observer regression | **20 POSIX + 14 PowerShell tests passed** with existing developer tools; PowerShell ran on macOS, not native Windows/PowerShell 5.1. The opt-in official Rex Lua contract passed separately with synthetic API data and no telemetry submission |
| Distribution | Six **0.3.0-dev** binaries/ZIPs built in an isolated ignored workspace. All **12** local and **6** archive-manifest hashes verified. Each archive has one matching binary, the bootstrap marker, dynamic/scope docs and **19** JSON examples; developer scripts/tests/Go/Lua/legacy files are excluded. The native package passed offline bootstrap, exact capabilities, all example layouts and isolated synthetic rendering |

**Native harness background-subagent scheduling and real LLM classification were
not end-to-end verified.** The skill defines a subagent-first procedure and
explicit foreground/main-agent fallback; the dashboard supplies the transport,
not an autonomous wake-up mechanism. General-purpose type alone is insufficient
identity evidence. Missing caller context stays unknown, and Rex reports remain
program reports rather than native agent telemetry.

No public release, commit, push, deployment, application settings/hook change,
private layout rewrite or installed-binary replacement occurred for this update.
The published 0.2.0 assets/bootstrap pins remain unchanged and lack these new
features. Only macOS arm64 was runtime-tested; all other targets remain
**cross-build-only**, unsigned and not runtime-verified. Historical sections below
retain the scope/version of their original checks.

## Published 0.2.0 and source-main pins (2026-11-20)

Published **Agent Dashboard v0.2.0** at **2026-11-20 09:32:56 UTC**, tag
`agent-dashboard-v0.2.0`, release-code commit `bc4065a`. The release contains
exactly **six platform ZIPs + archive-only SHA256SUMS**. Source-main bootstrap
constants and all six trusted archive pins were repinned in a separate follow-up;
the release tag and uploaded assets were not rebuilt or moved.

- Authenticated draft metadata confirmed every uploaded asset's exact local
  byte size and SHA-256 before publication.
- `python3 tests/release_smoke.py --expected-checksums <reviewed-local-manifest>`
  passed **anonymous downloads of all six ZIPs and the manifest**, matching the
  public manifest byte-for-byte to reviewed local artifacts and every ZIP to
  its expected SHA-256. Archive paths, platform membership, required docs,
  bootstrap helpers/marker and personal-data patterns were checked.
- The anonymously downloaded native macOS arm64 package passed **offline
  bootstrap, exact version and all three capabilities, CLI validation of all
  17 bundled JSON examples, and isolated synthetic plain rendering**.
- A fresh source-like directory with spaces, the real helper and new bundled
  pins downloaded published 0.2.0 **without authentication**, verified the
  remote manifest/archive, installed atomically, and matched the reviewed native
  binary exactly. Version/capabilities passed; a second bootstrap succeeded
  offline with no HTTPS/hash/ZIP utilities on PATH. Temporary bootstrap files
  were removed and skill instructions were unchanged.
- After repinning, **97 top-level Go tests + 56 subtests** passed again with
  race/coverage enabled and two conditional skips; `go vet` passed. All
  **20 POSIX + 14 PowerShell bootstrap tests** passed again on macOS.
- GitHub's latest release resolves to 0.2.0. The original **0.1.0 asset IDs,
  names, sizes and digests are unchanged**.

The source follow-up does not replace an installed binary or modify application
settings/hooks, private layouts, Rex configuration, or remote hosts. Public
cross-platform downloads/package checks are not foreign-OS runtime verification;
only macOS arm64 was run. The real fresh-download test did not install hooks,
submit telemetry or invoke Claude.

## Version 0.2.0 release preparation (2026-11-20)

The user authorized commit, push and release. Version 0.2.0 includes the optional
Rex adapters, shared typed observations and forward-only DAG contract. This
section records pre-publication checks; the older sections retain their dated
source/development scope, including statements that no release occurred then.

| Scope | Result / exact boundary |
| --- | --- |
| Go regression | `go test -count=1 -race -cover -json ./...`: **97 top-level tests + 56 subtests passed**, with **2 conditional skips**. `go vet ./...` passed; the opt-in official Rex Lua contract also passed separately with synthetic API data, without submitting telemetry |
| Coverage | Dashboard **90.9%**, CLI **71.7%**, Rex observer **88.7%**, packager **25.3%** |
| Bootstrap | **20 POSIX + 14 PowerShell tests passed**. Both helpers reject an incomplete prebundled package before network access and still reuse its existing binary offline. PowerShell ran on macOS, not native Windows or PowerShell 5.1 |
| Native PTY | **4 synthetic macOS runs**: delegation with q and Ctrl-C, fanout effects/controls/resize, and all Rex panel shapes through a synthetic CLI child. Terminal restoration, retained collection and child reaping passed. No new real-Claude or real-Rex telemetry acceptance is claimed |
| Packaging | Six **0.2.0** binaries/ZIPs built in an isolated ignored workspace. All **12** local and **6** archive-manifest hashes verified. Each archive includes exactly one platform binary, the generated `.prebundled` marker, both helpers, observations documentation and **17 forward-only JSON examples**; no developer/runtime scripts or legacy implementation are shipped |
| Native package | Extracted macOS arm64 package passed version/capability discovery, offline bootstrap, public CLI validation of all **17** examples and isolated synthetic plain rendering |
| Publication boundary | The release ships six platform ZIPs and archive-only `SHA256SUMS`. Source-main acquisition pins are updated only after public downloads match reviewed local artifacts. A package never downloads a fallback; release tags/ZIPs retain preceding source pins to avoid a self-referential archive checksum (README → Release pin lifecycle) |

Existing installed executables, Claude settings/hooks, private layouts, Rex
configuration and remote hosts were not changed. Public 0.1.0 assets are retained.
Only macOS arm64 received runtime checks; macOS amd64, Linux and Windows are
**cross-build-only**, not verified runtime targets. Builds remain unsigned.

## Forward-only DAG simplification (2026-11-20)

The current native DAG accepts only forward acyclic edges (`solid` / `dashed`).
Reverse/return routes, paired-arrow indicators, upward flow animation, the
return-traversal preference and its `r` control are removed. All edges contribute
to ranking; opposite pairs, self-edges and other cycles are rejected. Retired
`style: return` edges receive an explicit validation/migration diagnostic rather
than silent deletion or reinterpretation. Nodes still use the shared typed
Claude/Rex binding resolver, including literal returned/last-known state.

| Scope | Result / exact boundary |
| --- | --- |
| Go regression | `go test -count=1 -race -cover -json ./...`: **97 top-level tests + 56 subtests passed**, with **2 conditional skips** (helper parent entrypoint and opt-in official Lua contract). `go vet ./...` passed |
| Coverage | Dashboard **90.9%**, CLI **71.7%**, Rex observer **88.7%**, packager **24.2%**. The preceding shared-binding section records its earlier source snapshot, not the current percentage |
| Publication / saved layouts | Public layout/CLI tests reject the retired style even on a forward edge, reject former cycle/self-edge escapes, preserve the last valid published layout, and diagnose older persisted graphs without rewriting caller state |
| Examples / semantics | All **17** bundled JSON layouts and graph companions pass the public layout contract and render without reverse indicators. AI Hero delegation and findings remain forward; worker-to-lead and verdict-to-lead back edges are gone. Forward motion, link focus, unknown state, native/Rex typed bindings and change echoes retain regression coverage; the retired key has no hidden effect |
| Native macOS PTY | **4 explicitly synthetic runs**: forward delegation with q and Ctrl-C; fanout with effects, relationship controls, task echoes, motion-off/resume and resize; Rex all-panel observer with Ctrl-C using a labelled synthetic CLI child. Exit/terminal restoration passed, Claude collection stayed enabled, and the owned Rex child was reaped. These are not new real-Claude or real-Rex telemetry acceptance |
| PTY readiness | The developer harness waits for bounded actual startup/effect output; it no longer assumes a fixed window will catch a decorative head on a short forward edge |
| Distribution | Six **0.2.0-dev** platform binaries/ZIPs built in an isolated ignored repository workspace. All **12** local and **6** release-manifest hashes and all bundled forward-only examples verified. The packaged native binary passed capability discovery, isolated synthetic preview and deprecated-style rejection. Other OS/architectures remain **cross-build-only** |

The public 0.1.0 release/pins and existing installed binaries are unchanged. No
private layout, application settings, hooks, remote host, running agent, commit,
push or deployment was changed/performed. The unshipped earlier Python prototype
remains historical reference, not the native runtime. Older validation sections
below preserve dated evidence; reverse-flow capabilities there are **not** part
of the current native graph contract. See [DAG.md](DAG.md) for migration details.

## Shared observation bindings across panel shapes (2026-11-20)

This source-only update adds `observation-bindings-v1`. It does not publish a
release or change the immutable 0.1.0 bootstrap pins. No application settings,
hooks, real Claude API calls, observed programs, remote files or Rex configuration
were changed. No commit/push/deployment was performed for this update.

| Scope | Result / exact boundary |
| --- | --- |
| Go regression | `go test -count=1 -race -cover -json ./...`: **95 top-level tests + 39 subtests passed**, **2 top-level skips** (native helper parent entrypoint and opt-in official Lua test). `go vet ./...` passed. The native helper is exercised as a real child by other tests; the opt-in Lua test also passed separately |
| Coverage | Dashboard **91.2%**, CLI **71.7%**, Rex observer **88.7%**, packager **24.2%**; build-version constants have no executable statements |
| Shared view / binding seam | Public `ParseLayout`, `ProgramState`, `Viewer` and real headless Bubble Tea lifecycle cover one program key across session/card/list/work-report/DAG shapes, common relationship resolution, exact source filtering, typed namespaces, unknown/deleted records, last-known state without activity animation, explicit zero vs unknown progress, inert text, bounded cell geometry, and shared echoes/expiry. Original Claude layout/lifecycle/DAG/echo regression remains passing |
| Selection honesty | Rex does not populate Claude `Agents`/`Tasks`/`Tools`/`Events`, lead/session or completion. Native list panels require explicit typed program selection. Duplicate shorthand filter IDs do not duplicate records; ambiguous bindings are rejected; program `done` does not receive a native success check |
| Local history | Public `Monitor.Snapshot` through the CLI executor seam covers silent timestamp/heartbeat updates, complete-record changes, child removal with unknown outcome, detached snapshots, reconnect-baseline gap markers without fabricated replay, unchanged reconciliation, the 500-entry bound and exact discarded-entry count. Viewer event panels preserve filtered reading anchors; Rex diagnostic source scrolling remains separate |
| Official Lua runtime | Existing opt-in read-only Lua contract passed against the installed official Rex CLI, using only synthetic locally replaced API functions. No synthetic report was submitted to a Rex server; no new real-host cross-source acceptance is claimed |
| Native Rex-mode PTY | **3 macOS POSIX PTY runs with an explicitly synthetic CLI process**: current native binary with q and Ctrl-C, plus isolated packaged native binary with Ctrl-C. Every run renders all panel kinds from the shared caller template, consumes a live structured report change, shows record/DAG state and echo output, exercises keys/resize, restores terminal settings, and confirms the owned subscription PID is reaped. This is product process/TUI integration, **not a new real Rex program-telemetry acceptance** |
| Native Claude-mode PTY regression | **3 macOS POSIX PTY runs**, all labelled synthetic: ordinary canvas with q, ordinary canvas with Ctrl-C, and fanout DAG with effects, relationship controls, task echoes, motion-off/resume, navigation and resize. Terminal restoration and collection retention passed; no real Claude invocation was made |
| Bootstrap regression | **19 POSIX + 13 PowerShell tests passed**. PowerShell runs on macOS, not native Windows/PowerShell 5.1 validation. No download hashes or release assets were regenerated |
| Distribution | Six **0.2.0-dev** binaries/ZIPs cross-built in an isolated ignored repository workspace, leaving existing local `bin/` assets untouched. All archives contain shared-observation docs and the binding template, exactly one platform binary, and unchanged bootstrap pins; developer tests/Go/Lua/Python/legacy files are excluded. All **12** local-manifest and **6** archive-manifest hashes verified; native packaged capability discovery and Rex-mode synthetic PTY passed |

All non-Mac-arm64 targets remain **cross-build-only**. These checks do not
validate native Linux/Windows execution, full Pi lifecycle/tools/tasks, or
lossless history/fixed freshness guarantees. Review feedback covered the design;
no independent full code/security audit is claimed.

Developer-only PTY reproduction, from the skill directory after building a
native binary at the chosen path:

```sh
python3 tests/pty_rex_smoke.py target/observations-validation-agent-dashboard
python3 tests/pty_rex_smoke.py target/observations-validation-agent-dashboard --ctrl-c
```

The script uses a labelled local synthetic CLI executable and temporary
source/layout files under ignored `target/`; it never selects real hosts. It
is excluded from product ZIPs. The shared binding and local-history contracts
are in [OBSERVATIONS.md](OBSERVATIONS.md).

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
