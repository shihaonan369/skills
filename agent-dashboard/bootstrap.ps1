# Agent-invoked native Windows first-use setup. Compatible with PowerShell 5.1.
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Get-ExistingBinary([string] $Path) {
    $item = Get-Item -LiteralPath $Path -Force -ErrorAction SilentlyContinue
    if ($null -ne $item) {
        if ($item.PSIsContainer -or ($item.Attributes -band [IO.FileAttributes]::ReparsePoint) -or $item.Length -eq 0) {
            throw 'Existing executable must be a nonempty regular file, not a link.'
        }
        return $item.FullName
    }
    return $null
}

function Get-ArchiveChecksum([string] $Path, [string] $Asset) {
    $digests = New-Object 'System.Collections.Generic.List[string]'
    foreach ($line in [IO.File]::ReadAllLines($Path)) {
        $fields = $line -split '\s+'
        if ($fields.Count -ge 2 -and $fields[1] -ceq $Asset) {
            $match = [regex]::Match($line, '^([0-9a-fA-F]{64})  ' + [regex]::Escape($Asset) + '$')
            if (-not $match.Success) { throw 'Malformed archive checksum.' }
            $digests.Add($match.Groups[1].Value.ToLowerInvariant())
        }
    }
    if ($digests.Count -ne 1) { throw 'Missing or duplicate archive checksum.' }
    return $digests[0]
}

function Receive-Asset([Uri] $Uri, [string] $Path, [long] $Limit) {
    # Follow HTTPS redirects explicitly; never relax certificate verification.
    for ($redirect = 0; $redirect -le 5; $redirect++) {
        if ($Uri.Scheme -cne 'https') { throw 'Only HTTPS downloads are allowed.' }
        $request = [Net.HttpWebRequest] [Net.WebRequest]::Create($Uri)
        $request.AllowAutoRedirect = $false
        $request.Timeout = 60000
        $request.ReadWriteTimeout = 60000
        $request.UserAgent = 'agent-dashboard-bootstrap/0.2.0'
        $response = $request.GetResponse()
        try {
            $code = [int] $response.StatusCode
            if ($code -in @(301, 302, 303, 307, 308)) {
                $location = $response.Headers['Location']
                if ([string]::IsNullOrEmpty($location)) { throw 'Redirect has no location.' }
                $Uri = [Uri]::new($Uri, $location)
                continue
            }
            if ($code -ne 200) { throw "Download returned HTTP $code." }
            if ($response.ContentLength -gt $Limit) { throw 'Download exceeds size limit.' }
            $inputStream = $response.GetResponseStream()
            try {
                $outputStream = [IO.File]::Open($Path, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::None)
                try {
                    $buffer = New-Object byte[] 65536
                    $total = 0L
                    while (($read = $inputStream.Read($buffer, 0, $buffer.Length)) -gt 0) {
                        $total += $read
                        if ($total -gt $Limit) { throw 'Download exceeds size limit.' }
                        $outputStream.Write($buffer, 0, $read)
                    }
                } finally { $outputStream.Dispose() }
            } finally { $inputStream.Dispose() }
            return
        } finally { $response.Close() }
    }
    throw 'Too many download redirects.'
}

$stage = $null
$lockStream = $null
$oldTls = [Net.ServicePointManager]::SecurityProtocol
try {
    if ($args.Count -ne 0) { throw 'No arguments are accepted; the release source is pinned.' }
    if ($env:OS -ne 'Windows_NT') { throw 'Native Windows is required; macOS/Linux must use bootstrap.sh.' }
    if (-not (Test-Path -LiteralPath (Join-Path $PSScriptRoot 'SKILL.md') -PathType Leaf)) {
        throw 'SKILL.md is missing beside bootstrap.ps1.'
    }
    # W6432 reports native architecture for a translated/32-bit shell.
    $nativeArch = $env:PROCESSOR_ARCHITEW6432
    if ([string]::IsNullOrEmpty($nativeArch)) { $nativeArch = $env:PROCESSOR_ARCHITECTURE }
    switch ($nativeArch) {
        'ARM64' { $arch = 'arm64' }
        'AMD64' { $arch = 'amd64' }
        default { throw "Unsupported Windows architecture: $nativeArch" }
    }
    $target = 'windows-' + $arch
    $bin = Join-Path $PSScriptRoot 'bin'
    $directory = Join-Path $bin $target
    foreach ($path in @($bin, $directory)) {
        $item = Get-Item -LiteralPath $path -Force -ErrorAction SilentlyContinue
        if ($null -ne $item -and ($item.Attributes -band [IO.FileAttributes]::ReparsePoint)) {
            throw 'Bin directories must not be links or junctions.'
        }
    }
    $destination = Join-Path $directory 'agent-dashboard.exe'
    $existing = Get-ExistingBinary $destination
    if ($null -ne $existing) { Write-Output $existing; return }
    if (Test-Path -LiteralPath (Join-Path $PSScriptRoot '.prebundled')) {
        throw 'Prebundled package has no matching executable; acquire a complete package for this platform instead of downloading a fallback.'
    }
    [void] [IO.Directory]::CreateDirectory($directory)
    $lockPath = Join-Path $directory '.bootstrap.lock'
    $wait = [Diagnostics.Stopwatch]::StartNew()
    while ($null -eq $lockStream) {
        try {
            # OS sharing locks release on process exit; a leftover empty file is harmless.
            $lockStream = [IO.File]::Open($lockPath, [IO.FileMode]::OpenOrCreate, [IO.FileAccess]::ReadWrite, [IO.FileShare]::None)
        } catch [IO.IOException] {
            $existing = Get-ExistingBinary $destination
            if ($null -ne $existing) { Write-Output $existing; return }
            if ($wait.Elapsed.TotalSeconds -ge 60) { throw 'Bootstrap lock is busy; retry later.' }
            Start-Sleep -Milliseconds 200
        }
    }
    $existing = Get-ExistingBinary $destination
    if ($null -ne $existing) { Write-Output $existing; return }
    $stage = Join-Path $directory ('.bootstrap.' + [Guid]::NewGuid().ToString('N'))
    [void] [IO.Directory]::CreateDirectory($stage)
    $version = '0.2.0'
    $asset = "agent-dashboard-$version-$target.zip"
    $base = "https://github.com/shihaonan369/skills/releases/download/agent-dashboard-v$version"
    $expected = Get-ArchiveChecksum (Join-Path $PSScriptRoot 'bootstrap.sha256') $asset
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    [Console]::Error.WriteLine("agent-dashboard: acquiring verified $target release $version")
    $sums = Join-Path $stage 'SHA256SUMS'
    Receive-Asset ([Uri] "$base/SHA256SUMS") $sums 65536
    $published = Get-ArchiveChecksum $sums $asset
    if ($published -cne $expected) { throw 'Published checksum differs from bundled pin; nothing installed.' }
    $archive = Join-Path $stage 'archive.zip'
    Receive-Asset ([Uri] "$base/$asset") $archive 33554432
    $actual = (Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actual -cne $expected) { throw 'Archive checksum mismatch; nothing installed.' }

    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $zip = [IO.Compression.ZipFile]::OpenRead($archive)
    $stagedBinary = Join-Path $stage 'agent-dashboard.exe'
    try {
        $member = "agent-dashboard/bin/$target/agent-dashboard.exe"
        $entries = @($zip.Entries | Where-Object { $_.FullName -ceq $member })
        if ($entries.Count -ne 1) { throw 'Expected executable member is missing or duplicated.' }
        $entry = $entries[0]
        if ($entry.Length -le 0 -or $entry.Length -gt 33554432) { throw 'Executable member has invalid size.' }
        $inputStream = $entry.Open()
        try {
            $outputStream = [IO.File]::Open($stagedBinary, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::None)
            try {
                $buffer = New-Object byte[] 65536
                $total = 0L
                while (($read = $inputStream.Read($buffer, 0, $buffer.Length)) -gt 0) {
                    $total += $read
                    if ($total -gt 33554432) { throw 'Executable member exceeds size limit.' }
                    $outputStream.Write($buffer, 0, $read)
                }
                if ($total -ne $entry.Length) { throw 'Incomplete executable member.' }
            } finally { $outputStream.Dispose() }
        } finally { $inputStream.Dispose() }
    } finally { $zip.Dispose() }
    # Same-volume File.Move is atomic and refuses to replace an existing path.
    [IO.File]::Move($stagedBinary, $destination)
    Write-Output $destination
} catch {
    [Console]::Error.WriteLine('agent-dashboard bootstrap: ' + $_.Exception.Message)
    exit 1
} finally {
    [Net.ServicePointManager]::SecurityProtocol = $oldTls
    try {
        if ($null -ne $stage -and [IO.Directory]::Exists($stage)) { [IO.Directory]::Delete($stage, $true) }
    } finally {
        if ($null -ne $lockStream) { $lockStream.Dispose() }
    }
}
