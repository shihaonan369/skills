#!/usr/bin/env python3
"""Developer-only PowerShell contracts. Cross-host runs are not Windows validation."""
import hashlib
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest
import warnings
import zipfile

SKILL = Path(__file__).resolve().parents[1]
PWSH = os.environ.get("AGENT_DASHBOARD_TEST_PWSH") or shutil.which("pwsh") or shutil.which("powershell")

# Replace only the network boundary in memory; the shipped script is unchanged.
# Pin parsing, locking, ZIP handling, no-overwrite move, and cleanup run in .NET.
HARNESS = r'''
$ErrorActionPreference = 'Stop'
$path = Join-Path $PSScriptRoot 'bootstrap.ps1'
$tokens = $null
$errors = $null
$ast = [Management.Automation.Language.Parser]::ParseFile($path, [ref] $tokens, [ref] $errors)
if ($errors.Count -ne 0) { throw ($errors | Out-String) }
$download = $ast.Find({ param($node) $node -is [Management.Automation.Language.FunctionDefinitionAst] -and $node.Name -eq 'Receive-Asset' }, $true)
$mock = @'
function Receive-Asset([Uri] $Uri, [string] $Path, [long] $Limit) {
    if (-not $Uri.AbsoluteUri.StartsWith('https://github.com/shihaonan369/skills/releases/download/agent-dashboard-v0.1.0/')) {
        throw 'Unexpected download source.'
    }
    [IO.File]::AppendAllText($env:BOOTSTRAP_LOG, $Uri.AbsoluteUri + "`n")
    if ($env:DOWNLOAD_FAIL -eq '1') { throw 'Synthetic HTTP failure.' }
    if ($env:DOWNLOAD_DELAY) { Start-Sleep -Milliseconds ([int] $env:DOWNLOAD_DELAY) }
    $name = $Uri.Segments[-1]
    [IO.File]::Copy((Join-Path $env:BOOTSTRAP_ASSETS $name), $Path)
}
'@
$text = [IO.File]::ReadAllText($path)
$text = $text.Remove($download.Extent.StartOffset, $download.Extent.EndOffset - $download.Extent.StartOffset).Insert($download.Extent.StartOffset, $mock)
$text = $text.Replace('$PSScriptRoot', "'" + $PSScriptRoot.Replace("'", "''") + "'")
& ([scriptblock]::Create($text))
'''


@unittest.skipUnless(PWSH, "PowerShell developer tool unavailable; native Windows remains unverified")
class PowerShellBootstrapTests(unittest.TestCase):
    def setUp(self):
        output = SKILL / "target"
        output.mkdir(exist_ok=True)
        self.tmp = tempfile.TemporaryDirectory(prefix="bootstrap-powershell.", dir=output)
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name).resolve()
        self.skill = self.root / "skill with spaces"
        self.skill.mkdir()
        shutil.copyfile(SKILL / "bootstrap.ps1", self.skill / "bootstrap.ps1")
        (self.skill / "SKILL.md").write_text("unchanged skill instructions\n")
        (self.skill / "README.md").write_text("unchanged documentation\n")
        self.harness = self.skill / "harness.ps1"
        self.harness.write_text(HARNESS)
        self.assets = self.root / "assets"
        self.assets.mkdir()
        self.log = self.root / "requests.log"
        self.env = dict(os.environ, OS="Windows_NT", PROCESSOR_ARCHITECTURE="AMD64",
                        PROCESSOR_ARCHITEW6432="", BOOTSTRAP_ASSETS=str(self.assets),
                        BOOTSTRAP_LOG=str(self.log))
        self.archive()

    def archive(self, target="windows-amd64", members=None):
        self.asset = f"agent-dashboard-0.1.0-{target}.zip"
        self.payload = b"synthetic executable; must not be run"
        self.member = f"agent-dashboard/bin/{target}/agent-dashboard.exe"
        archive = self.assets / self.asset
        with warnings.catch_warnings():
            warnings.simplefilter("ignore", UserWarning)
            with zipfile.ZipFile(archive, "w") as z:
                for path, content in members or [(self.member, self.payload),
                                                 ("../../outside", b"never extract"),
                                                 ("agent-dashboard/SKILL.md", b"never replace")]:
                    z.writestr(path, content)
        self.digest = hashlib.sha256(archive.read_bytes()).hexdigest()
        manifest = f"{self.digest}  {self.asset}\n"
        (self.skill / "bootstrap.sha256").write_text(manifest)
        (self.assets / "SHA256SUMS").write_text(manifest)
        self.dest = self.skill / "bin" / target / "agent-dashboard.exe"

    def command(self):
        return [str(PWSH), "-NoProfile", "-NonInteractive", "-File", str(self.harness)]

    def run_bootstrap(self, **env):
        return subprocess.run(self.command(), env=dict(self.env, **env), text=True,
                              capture_output=True, timeout=30)

    def assert_clean(self):
        self.assertFalse([p for p in self.skill.glob("bin/*/.bootstrap.*") if p.is_dir()])
        self.assertEqual((self.skill / "SKILL.md").read_text(), "unchanged skill instructions\n")
        self.assertEqual((self.skill / "README.md").read_text(), "unchanged documentation\n")
        self.assertFalse((self.root / "outside").exists())

    def test_verified_download_and_existing_binary_offline(self):
        result = self.run_bootstrap()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.strip(), str(self.dest))
        self.assertEqual(self.dest.read_bytes(), self.payload)
        before = self.log.read_text()
        self.assertEqual(len(before.splitlines()), 2)
        result = self.run_bootstrap(DOWNLOAD_FAIL="1")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.log.read_text(), before)
        self.assert_clean()

    def test_native_architecture_wins_over_emulated_shell(self):
        self.archive("windows-arm64")
        result = self.run_bootstrap(PROCESSOR_ARCHITECTURE="x86", PROCESSOR_ARCHITEW6432="ARM64")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.strip(), str(self.dest))
        self.assertEqual(self.dest.read_bytes(), self.payload)
        self.assert_clean()

    def test_checksum_mismatch_installs_nothing(self):
        (self.assets / self.asset).write_bytes(b"tampered")
        result = self.run_bootstrap()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("checksum mismatch", result.stderr.lower())
        self.assertFalse(self.dest.exists())
        self.assert_clean()

    def test_remote_checksum_cannot_replace_pin(self):
        (self.assets / "SHA256SUMS").write_text(f"{'0' * 64}  {self.asset}\n")
        result = self.run_bootstrap()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("bundled pin", result.stderr)
        self.assertFalse(self.dest.exists())
        self.assert_clean()

    def test_missing_duplicate_and_malformed_checksums(self):
        for manifest in ("", f"{self.digest}  {self.asset}\n" * 2,
                         f"{self.digest}\t{self.asset}\n", f"invalid  {self.asset}\n"):
            with self.subTest(manifest=manifest):
                (self.assets / "SHA256SUMS").write_text(manifest)
                result = self.run_bootstrap()
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse(self.dest.exists())
                self.assert_clean()

    def test_missing_duplicate_and_empty_members(self):
        for members in ([("wrong.exe", self.payload)],
                        [(self.member, self.payload), (self.member, self.payload)],
                        [(self.member, b"")]):
            with self.subTest(members=members):
                self.archive(members=members)
                result = self.run_bootstrap()
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse(self.dest.exists())
                self.assert_clean()

    def test_missing_bundled_pin_stops_before_network(self):
        (self.skill / "bootstrap.sha256").unlink()
        result = self.run_bootstrap()
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(self.log.exists())
        self.assertFalse(self.dest.exists())
        self.assert_clean()

    def test_bin_link_is_not_followed(self):
        outside = self.root / "outside-bin"
        outside.mkdir()
        try:
            (self.skill / "bin").symlink_to(outside, target_is_directory=True)
        except OSError as error:
            self.skipTest(f"developer host cannot create test symlinks: {error}")
        result = self.run_bootstrap()
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(self.log.exists())
        self.assertFalse(list(outside.iterdir()))

    def test_download_failure_cleans_stage(self):
        result = self.run_bootstrap(DOWNLOAD_FAIL="1")
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(self.dest.exists())
        self.assert_clean()

    def test_unsupported_architecture_and_host_do_not_download(self):
        for env in ({"PROCESSOR_ARCHITECTURE": "x86"}, {"OS": "Linux"}):
            with self.subTest(env=env):
                result = self.run_bootstrap(**env)
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse(self.log.exists())

    def test_existing_directory_is_not_replaced(self):
        self.dest.mkdir(parents=True)
        result = self.run_bootstrap()
        self.assertNotEqual(result.returncode, 0)
        self.assertTrue(self.dest.is_dir())
        self.assertFalse(self.log.exists())
        self.assert_clean()

    def test_abandoned_empty_lock_file_does_not_block(self):
        self.dest.parent.mkdir(parents=True)
        (self.dest.parent / ".bootstrap.lock").touch()
        result = self.run_bootstrap()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.dest.read_bytes(), self.payload)
        self.assert_clean()

    def test_parallel_first_use_publishes_once(self):
        env = dict(self.env, DOWNLOAD_DELAY="300")
        processes = [subprocess.Popen(self.command(), env=env, text=True,
                                      stdout=subprocess.PIPE, stderr=subprocess.PIPE) for _ in range(2)]
        try:
            for process in processes:
                out, err = process.communicate(timeout=30)
                self.assertEqual(process.returncode, 0, err)
                self.assertEqual(out.strip(), str(self.dest))
        finally:
            for process in processes:
                if process.poll() is None:
                    process.kill()
                    process.communicate()
        self.assertEqual(len(self.log.read_text().splitlines()), 2)
        self.assertEqual(self.dest.read_bytes(), self.payload)
        self.assert_clean()


if __name__ == "__main__":
    unittest.main()
