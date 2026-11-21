#!/usr/bin/env python3
"""Developer-only anonymous Release download check; no product runtime dependency."""
import argparse
import hashlib
import io
import json
import os
from pathlib import Path, PurePosixPath
import platform
import re
import subprocess
import sys
import tempfile
from urllib.error import HTTPError
from urllib.parse import quote
from urllib.request import Request, urlopen
import zipfile

TARGETS = ("darwin-arm64", "darwin-amd64", "linux-arm64", "linux-amd64",
           "windows-arm64", "windows-amd64")


def fetch(url):
    # Intentionally no auth token or GitHub CLI: exercise public-user downloads.
    request = Request(url, headers={"User-Agent": "agent-dashboard-release-smoke"})
    with urlopen(request, timeout=60) as response:
        data = response.read((32 << 20) + 1)
    if len(data) > 32 << 20:
        raise ValueError("asset exceeds this smoke test's 32 MiB limit")
    return data


def run(repo, tag, version, expected_checksums=None):
    if not re.fullmatch(r"[\w.-]+/[\w.-]+", repo):
        raise ValueError("expected owner/repository")
    metadata = json.loads(fetch(f"https://api.github.com/repos/{repo}/releases/tags/{quote(tag, safe='')}"))
    assert metadata["tag_name"] == tag and not metadata["draft"], "release not published"
    names = {f"agent-dashboard-{version}-{target}.zip": target for target in TARGETS}
    assets = {a["name"]: a for a in metadata["assets"]}
    assert set(assets) == set(names) | {"SHA256SUMS"}, "expected six platform ZIPs and SHA256SUMS"
    manifest = fetch(assets["SHA256SUMS"]["browser_download_url"])
    if expected_checksums is not None:
        assert manifest == Path(expected_checksums).read_bytes(), "published manifest differs from reviewed local artifacts"
    checksums = {}
    for line in manifest.decode().splitlines():
        digest, name = line.split("  ", 1)
        assert name in names and name not in checksums and re.fullmatch(r"[a-f0-9]{64}", digest)
        checksums[name] = digest
    assert set(checksums) == set(names), "incomplete release checksums"
    host_os = {"Darwin": "darwin", "Linux": "linux", "Windows": "windows"}.get(platform.system())
    host_arch = {"arm64": "arm64", "aarch64": "arm64", "x86_64": "amd64", "amd64": "amd64"}.get(platform.machine().lower())
    host_target = f"{host_os}-{host_arch}"
    dynamic = tuple(map(int, version.split(".")[:2])) >= (0, 3)
    output_dir = Path(__file__).resolve().parents[1] / "target"
    output_dir.mkdir(exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="release-smoke.", dir=output_dir) as tmp:
        for name, target in names.items():
            data = fetch(assets[name]["browser_download_url"])
            assert hashlib.sha256(data).hexdigest() == checksums[name], f"checksum mismatch: {name}"
            with zipfile.ZipFile(io.BytesIO(data)) as archive:
                members = archive.namelist()
                for member in members:
                    path = PurePosixPath(member)
                    assert not path.is_absolute() and ".." not in path.parts and path.parts[0] == "agent-dashboard"
                binary_name = "agent-dashboard.exe" if target.startswith("windows-") else "agent-dashboard"
                binary = f"agent-dashboard/bin/{target}/{binary_name}"
                assert [p for p in members if "/bin/" in p] == [binary], "wrong platform executable"
                required = ("SKILL.md", "README.md", "LICENSES.txt", ".prebundled",
                            "bootstrap.sh", "bootstrap.ps1", "bootstrap.sha256",
                            "docs/PANELS.md", "docs/DAG.md", "docs/REX.md",
                            "docs/OBSERVATIONS.md", "examples/rex/records.template.layout.json")
                if dynamic:
                    required += ("docs/DYNAMIC.md", "examples/dynamic/README.md",
                                 "examples/dynamic/actors.layout.json", "examples/dynamic/programs.layout.json")
                assert all("agent-dashboard/" + p in members for p in required), "incomplete skill package"
                assert any(p.startswith("agent-dashboard/examples/") for p in members)
                assert not any("/prototypes/" in p or "/.scratch/" in p or p.endswith(".py") for p in members)
                for member in members:
                    if member.endswith(".md"):
                        text = archive.read(member).decode()
                        assert not re.search(r"/(?:Users|home)/[^/\s`\"']+|[\w.+-]+@(?:live|outlook|hotmail|gmail|qq|163|icloud)\.[A-Za-z.]+", text), "personal data pattern in packaged docs"
                if target == host_target:
                    # Paths above are validated before extracting the host package.
                    archive.extractall(tmp)
                    skill = Path(tmp) / "agent-dashboard"
                    executable = skill / "bin" / target / binary_name
                    executable.chmod(0o755)
                    if host_os != "windows":
                        ready = subprocess.check_output(["sh", str(skill / "bootstrap.sh")], text=True)
                        assert ready.strip() == str(executable), "package bootstrap did not reuse its binary"
                    env = dict(os.environ, AGENT_DASHBOARD_HOME=str(Path(tmp) / "state"))
                    output = subprocess.check_output([str(executable), "version"], env=env, text=True)
                    assert output.strip() == f"agent-dashboard {version} · {host_os}/{host_arch}"
                    capabilities = json.loads(subprocess.check_output([str(executable), "capabilities"], env=env, text=True))
                    assert capabilities["version"] == version
                    assert {"rex-local-open-v1", "rex-observer-v1", "observation-bindings-v1"} <= set(capabilities["features"])
                    if dynamic:
                        assert {"dynamic-layout-v1", "agent-scope-v1"} <= set(capabilities["features"])
                        scope = Path(tmp) / "scope.json"
                        subprocess.run([str(executable), "scope-init", "--file", str(scope),
                                        "--criterion", "Synthetic public package smoke; no model invoked."], env=env, check=True)
                        subprocess.run([str(executable), "scope-close", "--file", str(scope)], env=env, check=True)
                        closed = json.loads(subprocess.check_output([str(executable), "scope-watch", "--file", str(scope),
                                                                    "--duration", "1"], env=env, text=True))
                        assert closed["closed"] and not closed["requests"], "scope-watch closure failed"
                    examples = sorted((skill / "examples").rglob("*.json"))
                    if dynamic:
                        assert len(examples) == 19, "expected all current bundled layouts"
                    for example in examples:
                        layout = json.loads(example.read_text())
                        if "root" not in layout:
                            layout = {"version": 1, "name": "Release graph validation",
                                      "root": {"kind": "dag", "id": "release-graph", "dag": layout}}
                        subprocess.run([str(executable), "demo", "--json", json.dumps(layout)],
                                       env=env, check=True, stdout=subprocess.DEVNULL)
                    print(f"PASS native layout validation: {len(examples)} bundled examples")
                    subprocess.run([str(executable), "demo"], env=env, check=True, stdout=subprocess.DEVNULL)
                    frame = subprocess.check_output([str(executable), "tui", "--session", "DEMO-layout", "--plain"], env=env, text=True)
                    assert "DEMO" in frame and "SYNTHETIC" in frame
                    print(f"PASS native startup + synthetic plain frame: {target}")
            print(f"PASS anonymous download/hash/package: {name}")
    print(f"PASS release {tag}: all six public assets and archive-only checksums")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", default="shihaonan369/skills")
    parser.add_argument("--tag", default="agent-dashboard-v0.3.0")
    parser.add_argument("--version", default="0.3.0")
    parser.add_argument("--expected-checksums", help="reviewed local archive-only SHA256SUMS to match byte-for-byte")
    args = parser.parse_args()
    try:
        run(args.repo, args.tag, args.version, args.expected_checksums)
    except HTTPError as error:
        print(f"FAIL public release request: HTTP {error.code}", file=sys.stderr)
        sys.exit(1)
