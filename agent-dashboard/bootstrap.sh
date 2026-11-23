#!/bin/sh
# Agent-invoked first-use setup. Only a verified executable is installed.
set -eu
umask 077

fail() { printf 'agent-dashboard bootstrap: %s\n' "$*" >&2; exit 1; }
[ "$#" -eq 0 ] || fail 'no arguments are accepted; the release source is pinned'
skill_dir=$(CDPATH='' cd -P -- "$(dirname -- "$0")" && pwd)
[ -f "$skill_dir/SKILL.md" ] || fail 'SKILL.md is missing beside bootstrap.sh'

case $(uname -s) in
    Darwin) os=darwin ;;
    Linux) os=linux ;;
    *) fail 'unsupported OS; native Windows must use bootstrap.ps1' ;;
esac
arch=$(uname -m)
# A translated shell on Apple Silicon should still fetch the native executable.
if [ "$os" = darwin ] && [ "$arch" = x86_64 ] && command -v sysctl >/dev/null 2>&1; then
    if [ "$(sysctl -n hw.optional.arm64 2>/dev/null || :)" = 1 ]; then arch=arm64; fi
fi
case "$arch" in
    arm64|aarch64) arch=arm64 ;;
    x86_64|amd64) arch=amd64 ;;
    *) fail "unsupported architecture: $arch" ;;
esac
target=$os-$arch
directory=$skill_dir/bin/$target
destination=$directory/agent-dashboard
[ ! -L "$skill_dir/bin" ] && [ ! -L "$directory" ] || fail 'bin directories must not be symlinks'

existing() {
    [ ! -L "$destination" ] || fail 'refusing an executable symlink'
    if [ -e "$destination" ]; then
        [ -f "$destination" ] && [ -s "$destination" ] && [ -x "$destination" ] || fail 'existing executable is not a nonempty executable file'
        printf '%s\n' "$destination"
        return 0
    fi
    return 1
}
if existing; then exit 0; fi
[ ! -e "$skill_dir/.prebundled" ] || fail 'prebundled package has no matching executable; acquire a complete package for this platform instead of downloading a fallback'

# No package manager, compiler, Python, or Node installation is performed.
if command -v curl >/dev/null 2>&1; then downloader=curl
elif command -v wget >/dev/null 2>&1; then downloader=wget
else fail 'no HTTPS downloader available (curl or wget); report this environment blocker'; fi
if command -v shasum >/dev/null 2>&1; then hasher=shasum
elif command -v sha256sum >/dev/null 2>&1; then hasher=sha256sum
else fail 'no SHA-256 utility available; report this environment blocker'; fi
if command -v unzip >/dev/null 2>&1; then extractor=unzip
elif command -v tar >/dev/null 2>&1; then extractor=tar
else fail 'no ZIP reader available (unzip or ZIP-capable tar); report this environment blocker'; fi

mkdir -p "$directory"
lock=$directory/.bootstrap-lock
stage=''
locked=0
cleanup() {
    if [ -n "$stage" ]; then rm -rf -- "$stage"; fi
    if [ "$locked" -eq 1 ]; then rmdir -- "$lock"; fi
}
trap cleanup 0
trap 'exit 130' INT
trap 'exit 143' TERM
trap 'exit 129' HUP
attempt=0
until mkdir "$lock" 2>/dev/null; do
    if existing; then exit 0; fi
    attempt=$((attempt + 1))
    [ "$attempt" -lt 60 ] || fail 'bootstrap lock is busy; retry later (do not delete an active lock)'
    sleep 1
done
locked=1
if existing; then exit 0; fi
stage=$(mktemp -d "$directory/.bootstrap.XXXXXX")
version=0.4.0
asset=agent-dashboard-$version-$target.zip
base=https://github.com/shihaonan369/skills/releases/download/agent-dashboard-v$version

fetch() {
    url=$1 output=$2 limit=$3
    if [ "$downloader" = curl ]; then
        curl --fail --location --silent --show-error --proto '=https' --proto-redir '=https' \
            --connect-timeout 15 --max-time 180 --max-filesize "$limit" --retry 2 \
            --output "$output" "$url" || fail "download failed: $url"
    else
        wget --https-only --timeout=30 --tries=3 -O "$output" "$url" || fail "download failed: $url"
    fi
    size=$(wc -c < "$output" | tr -d '[:space:]')
    [ "$size" -le "$limit" ] || fail 'download exceeds size limit'
}
printf 'agent-dashboard: acquiring verified %s release %s\n' "$target" "$version" >&2
checksum() {
    awk -v name="$asset" '$2 == name { count++; digest=$1; if ($0 !~ /^[0-9a-fA-F]+  [A-Za-z0-9_.-]+$/) bad=1 } END { if (bad || count != 1 || length(digest) != 64 || digest ~ /[^0-9a-fA-F]/) exit 1; print tolower(digest) }' "$1"
}
expected=$(checksum "$skill_dir/bootstrap.sha256") || fail 'bundled archive checksum is missing, duplicate, or malformed'
fetch "$base/SHA256SUMS" "$stage/SHA256SUMS" 65536
published=$(checksum "$stage/SHA256SUMS") || fail 'missing, duplicate, or malformed archive checksum'
[ "$published" = "$expected" ] || fail 'published checksum differs from bundled pin; nothing installed'
fetch "$base/$asset" "$stage/archive.zip" 33554432
if [ "$hasher" = shasum ]; then
    actual=$(shasum -a 256 "$stage/archive.zip" | awk '{print $1}')
else
    actual=$(sha256sum "$stage/archive.zip" | awk '{print $1}')
fi
[ "$actual" = "$expected" ] || fail 'archive checksum mismatch; nothing installed'

member=agent-dashboard/bin/$target/agent-dashboard
# Stream exactly one expected member, never unpack archive paths or overwrite docs.
if [ "$extractor" = unzip ]; then
    (ulimit -f 65536; unzip -Z1 "$stage/archive.zip") > "$stage/members" || fail 'cannot list ZIP members'
else
    (ulimit -f 65536; tar -tf "$stage/archive.zip") > "$stage/members" || fail 'tar cannot read this ZIP; report this environment blocker'
fi
count=$(awk -v name="$member" '$0 == name { count++ } END { print count+0 }' "$stage/members")
[ "$count" -eq 1 ] || fail 'expected executable member is missing or duplicated'
if [ "$extractor" = unzip ]; then
    (ulimit -f 65536; unzip -p "$stage/archive.zip" "$member") > "$stage/agent-dashboard" || fail 'cannot read executable member'
else
    (ulimit -f 65536; tar -xOf "$stage/archive.zip" "$member") > "$stage/agent-dashboard" || fail 'cannot read executable member'
fi
size=$(wc -c < "$stage/agent-dashboard" | tr -d '[:space:]')
[ "$size" -gt 0 ] && [ "$size" -le 33554432 ] || fail 'executable member has invalid size'
chmod 755 "$stage/agent-dashboard"
# Same-filesystem hard-link publication is atomic and refuses an existing file.
if existing; then exit 0; fi
ln "$stage/agent-dashboard" "$directory/" || fail 'cannot publish executable without overwriting an existing path'
printf '%s\n' "$destination"
