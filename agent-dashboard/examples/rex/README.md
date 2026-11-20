# Caller-owned Rex presentation example

Requires the version 0.2.0 capabilities described in
[REX.md](../../docs/REX.md); published 0.1.0 does not contain them.

`overview.layout.json` is a small, workflow-free composition: one compact
`programs` panel shows each selected source on its own row. It supplies **no
telemetry, host endpoints, agent roles, or fake status**. The caller resolves
real approved sources into a separate private UTF-8 file and may replace the
composition entirely.

The agent performs setup and opening, for example on POSIX:

```sh
"$D" rex-sources --host devbox > "$private/sources.json"
"$D" open --sources "@$private/sources.json" --layout @examples/rex/overview.layout.json
```

Use an existing private directory with `umask 077`. Source aliases can be renamed
in that private configuration while keeping the resolved endpoint and host pin.
No real operational source IDs/configuration belong in this example.

Compact rows show source, self-reported state/app, known percentage or `?`,
session/block identity, and record ID. Shortened block IDs distinguish repeated
pane labels but are not binding keys. Press `2` for full program details and
`3` for source identities/health; `1` returns to this caller canvas. Binding a
particular `program_ids` requires the full source-qualified key returned by
`rex-watch --status`, not an ID guessed from the screen. Existing reports do
not prove live agent delegation, full tool telemetry, or Issue completion.

## Shared record-panel template

`records.template.layout.json` illustrates **all panel shapes**, using the one
shared program binding contract from [OBSERVATIONS.md](../../docs/OBSERVATIONS.md).
It requires `observation-bindings-v1`. Every all-zero `rex:…` key is an intentionally
unbound placeholder, not a real identity or synthetic status report. The agent
copies the file to the caller's private directory and replaces **every** such
key with an actual full key from the private `rex-watch --status` snapshot.
The same record appears in several shapes to demonstrate composability; that
does not mean multiple agents/tasks exist. No workflow or graph edge is inferred.

The unbound `session` panel shows the selected Rex scope; single/list record
panels and a DAG node project the selected report. The events panel deliberately
has no program filter so that source and reconnect-baseline boundaries remain
visible. `programs`, `sources`, caller text and row/column composition retain
their existing behavior. Start with a tall terminal (for example 180 × 64), or
remove/rearrange panels; this is an optional caller template, not a default view.
