# Caller-owned external countdown

Requires version 0.4.0+ and `file-data-bindings-v1`; published 0.3.0 assets
do not support this example. Follow [DATA.md](../../docs/DATA.md) first.
This is an optional caller illustration, not a workflow or a fabricated native
agent/task. No model or subagent is involved in numeric updates.

1. Choose an approved private absolute JSON path on the viewer's host.
2. Copy `countdown.template.layout.json` to private storage, or add its fields to
   the caller's existing composition. Replace `data_sources.timer.path` with that
   actual path (including a drive-qualified path on native Windows). The bundled
   path is a placeholder, not a directory the dashboard should create implicitly.
3. Publish the private layout once with the actual session ID. Do not replace an
   unrelated composition without the caller's approval.
4. Start the producer through the caller's existing tools, using the verified
   executable `D` and the same absolute `DATA` path. The viewer loads data without
   repeatedly publishing layouts or touching Claude/Rex telemetry.
5. Stop the independently owned producer when no longer needed. Closing the
   viewer does not stop it. A stopped producer eventually appears stale.

## POSIX shell producer (using existing host tools)

```sh
# D and DATA are already-approved absolute paths.
end=$(( $(date +%s) + 120 ))
while :; do
    remaining=$(( end - $(date +%s) ))
    [ "$remaining" -ge 0 ] || remaining=0
    "$D" data-write --file "$DATA" --json "{\"remaining_seconds\":$remaining}" || break
    [ "$remaining" -gt 0 ] || break
    sleep 1
done
```

Only validated numeric values are interpolated into this example JSON. For
arbitrary strings, use the producer language's JSON encoder or pass a complete
JSON file/stdin; do not concatenate untrusted text into commands.

## Native PowerShell producer

```powershell
# $D and $Data are already-approved absolute paths.
$End = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds() + 120
do {
    $Remaining = [Math]::Max(0, $End - [DateTimeOffset]::UtcNow.ToUnixTimeSeconds())
    $Json = @{ remaining_seconds = $Remaining } | ConvertTo-Json -Compress
    & $D data-write --file $Data --json $Json
    if ($LASTEXITCODE -ne 0) { throw 'External data publication failed.' }
    if ($Remaining -gt 0) { Start-Sleep -Seconds 1 }
} while ($Remaining -gt 0)
```

These snippets use existing shell utilities only; no end-user Python/Node/Go
installation is required. The snippets are not native-Windows runtime acceptance
evidence. `data-write` supplies a fresh publication timestamp automatically.
The script owns the countdown; the viewer only formats remaining seconds.
