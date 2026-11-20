#!/usr/bin/env python3
"""Developer-only bootstrap contract tests; Python is not a product dependency."""
import hashlib
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest
import warnings
import zipfile

SKILL = Path(__file__).resolve().parents[1]
RELEASE = "https://github.com/shihaonan369/skills/releases/download/agent-dashboard-v0.1.0"
PAYLOAD = b"#!/bin/sh\necho 'synthetic executable; bootstrap must not run this'\nexit 99\n"


class BootstrapTests(unittest.TestCase):
    def setUp(self):
        output = SKILL / "target"
        output.mkdir(exist_ok=True)
        self.tmp = tempfile.TemporaryDirectory(prefix="bootstrap-test.", dir=output)
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name).resolve()
        self.skill = self.root / "skill with spaces"
        self.skill.mkdir()
        self.script = self.skill / "bootstrap.sh"
        # A missing implementation is an intentional red test, not a skip.
        shutil.copyfile(SKILL / "bootstrap.sh", self.script)
        (self.skill / "SKILL.md").write_text("unchanged skill instructions\n")
        (self.skill / "README.md").write_text("unchanged documentation\n")
        self.assets = self.root / "assets"
        self.assets.mkdir()
        self.tools = self.root / "tools"
        self.tools.mkdir()
        self.log = self.root / "requests.log"
        self.env = dict(os.environ, BOOTSTRAP_ASSETS=str(self.assets),
                        BOOTSTRAP_LOG=str(self.log), TEST_OS="Darwin", TEST_ARCH="arm64")
        # Keep the test's utility availability deterministic, including fallback tests.
        for name in ("dirname", "mkdir", "mktemp", "rm", "rmdir", "sleep", "wc", "awk",
                     "tr", "chmod", "ln", "uname", "curl", "shasum", "sha256sum", "unzip", "tar"):
            path = shutil.which(name)
            if path:
                (self.tools / name).symlink_to(path)
        self.env["PATH"] = str(self.tools)
        self.tool("uname", '#!/bin/sh\ncase "$1" in -s) echo "$TEST_OS";; -m) echo "$TEST_ARCH";; esac\n')
        downloader = f'''#!{sys.executable}
import os
from pathlib import Path
import shutil
import sys
import time
args = sys.argv[1:]
url = next(a for a in args if a.startswith("https://"))
assert url.startswith({RELEASE!r} + "/"), url
with open(os.environ["BOOTSTRAP_LOG"], "a") as log:
    log.write(url + "\\n")
if os.environ.get("DOWNLOAD_FAIL"):
    sys.exit(22)
time.sleep(float(os.environ.get("DOWNLOAD_DELAY", "0")))
flag = "--output" if "--output" in args else "-O"
shutil.copyfile(Path(os.environ["BOOTSTRAP_ASSETS"]) / url.rsplit("/", 1)[1], args[args.index(flag) + 1])
'''
        self.tool("curl", downloader)
        self.downloader = downloader
        self.archive()

    def tool(self, name, content):
        path = self.tools / name
        if path.is_symlink():
            path.unlink()
        path.write_text(content)
        path.chmod(0o755)

    def archive(self, target="darwin-arm64", members=None):
        binary = "agent-dashboard.exe" if target.startswith("windows-") else "agent-dashboard"
        name = f"agent-dashboard-0.1.0-{target}.zip"
        archive = self.assets / name
        member = f"agent-dashboard/bin/{target}/{binary}"
        with warnings.catch_warnings():
            warnings.simplefilter("ignore", UserWarning)
            with zipfile.ZipFile(archive, "w") as z:
                for path, content in members or [(member, PAYLOAD), ("agent-dashboard/README.md", b"do not install this")]:
                    z.writestr(path, content)
        self.asset = name
        self.digest = hashlib.sha256(archive.read_bytes()).hexdigest()
        manifest = f"{self.digest}  {name}\n"
        (self.assets / "SHA256SUMS").write_text(manifest)
        (self.skill / "bootstrap.sha256").write_text(manifest)
        self.dest = self.skill / "bin" / target / binary

    def run_bootstrap(self, **env):
        return subprocess.run(["/bin/sh", str(self.script)], env=dict(self.env, **env),
                              text=True, capture_output=True, timeout=20)

    def assert_clean(self):
        self.assertFalse(list(self.skill.glob("bin/*/.bootstrap*")))
        self.assertEqual((self.skill / "SKILL.md").read_text(), "unchanged skill instructions\n")
        self.assertEqual((self.skill / "README.md").read_text(), "unchanged documentation\n")

    def test_verified_download_installs_only_binary_and_reuses_offline(self):
        result = self.run_bootstrap()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.strip(), str(self.dest))
        self.assertEqual(self.dest.read_bytes(), PAYLOAD)
        self.assertTrue(os.access(self.dest, os.X_OK))
        self.assertEqual(len(self.log.read_text().splitlines()), 2)
        # The deliberately failing payload must never have been executed by bootstrap.
        self.assertNotIn("synthetic executable", result.stdout)
        self.assert_clean()
        before = self.log.read_text()
        result = self.run_bootstrap(DOWNLOAD_FAIL="1")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.log.read_text(), before)
        self.assert_clean()

    def test_checksum_mismatch_never_installs(self):
        (self.assets / self.asset).write_bytes(b"tampered archive")
        result = self.run_bootstrap()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("checksum", result.stderr.lower())
        self.assertFalse(self.dest.exists())
        self.assert_clean()

    def test_missing_or_duplicate_checksum_never_installs(self):
        for manifest in ("", f"{self.digest}  wrong.zip\n",
                         f"{self.digest}  {self.asset}\n" * 2,
                         f"not-a-sha256  {self.asset}\n",
                         f"{self.digest}\t{self.asset}\n",
                         f"{self.digest}  {self.asset} extra\n"):
            with self.subTest(manifest=manifest):
                (self.assets / "SHA256SUMS").write_text(manifest)
                result = self.run_bootstrap()
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse(self.dest.exists())
                self.assert_clean()

    def test_missing_duplicate_and_empty_archive_member_are_rejected(self):
        member = "agent-dashboard/bin/darwin-arm64/agent-dashboard"
        for members in ([("elsewhere/agent-dashboard", PAYLOAD)],
                        [(member, PAYLOAD), (member, PAYLOAD)], [(member, b"")]):
            with self.subTest(members=members):
                self.archive(members=members)
                result = self.run_bootstrap()
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse(self.dest.exists())
                self.assert_clean()

    def test_unrelated_archive_paths_are_not_extracted(self):
        member = "agent-dashboard/bin/darwin-arm64/agent-dashboard"
        self.archive(members=[(member, PAYLOAD), ("../../escape", b"unsafe"),
                              ("agent-dashboard/SKILL.md", b"replacement")])
        result = self.run_bootstrap()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse((self.root / "escape").exists())
        self.assertCountEqual([p.relative_to(self.skill).as_posix() for p in self.skill.rglob("*") if p.is_file()],
                         ["bootstrap.sh", "bootstrap.sha256", "README.md", "SKILL.md", "bin/darwin-arm64/agent-dashboard"])
        self.assert_clean()

    def test_linux_arch_aliases_and_intel_mac(self):
        for os_name, arch, target in (("Linux", "aarch64", "linux-arm64"),
                                      ("Linux", "x86_64", "linux-amd64"),
                                      ("Darwin", "x86_64", "darwin-amd64")):
            with self.subTest(target=target):
                self.archive(target)
                result = self.run_bootstrap(TEST_OS=os_name, TEST_ARCH=arch)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(result.stdout.strip(), str(self.dest))
                self.assertEqual(self.dest.read_bytes(), PAYLOAD)
                self.assert_clean()

    def test_unsupported_platform_does_not_download(self):
        for os_name, arch in (("FreeBSD", "amd64"), ("Linux", "i686"), ("MINGW64_NT", "x86_64")):
            with self.subTest(os_name=os_name, arch=arch):
                result = self.run_bootstrap(TEST_OS=os_name, TEST_ARCH=arch)
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse(self.log.exists())

    def test_changed_archive_and_remote_manifest_cannot_replace_bundled_pin(self):
        (self.assets / self.asset).write_bytes(b"attacker-controlled archive")
        digest = hashlib.sha256((self.assets / self.asset).read_bytes()).hexdigest()
        (self.assets / "SHA256SUMS").write_text(f"{digest}  {self.asset}\n")
        result = self.run_bootstrap()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("bundled pin", result.stderr)
        self.assertEqual(len(self.log.read_text().splitlines()), 1)
        self.assertFalse(self.dest.exists())
        self.assert_clean()

    def test_missing_bundled_pin_stops_before_network(self):
        (self.skill / "bootstrap.sha256").unlink()
        result = self.run_bootstrap()
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(self.log.exists())
        self.assertFalse(self.dest.exists())
        self.assert_clean()

    def test_prebundled_package_never_downloads_a_fallback(self):
        (self.skill / ".prebundled").write_text("platform package\n")
        result = self.run_bootstrap()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("prebundled package", result.stderr)
        self.assertFalse(self.log.exists())
        self.assertFalse(self.dest.exists())
        self.dest.parent.mkdir(parents=True)
        self.dest.write_bytes(PAYLOAD)
        self.dest.chmod(0o755)
        result = self.run_bootstrap(DOWNLOAD_FAIL="1")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.strip(), str(self.dest))
        self.assertFalse(self.log.exists())
        self.assert_clean()

    def test_failed_download_cleans_temporary_files(self):
        result = self.run_bootstrap(DOWNLOAD_FAIL="1")
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(self.dest.exists())
        self.assert_clean()

    def test_existing_binary_and_bin_symlinks_are_not_followed(self):
        outside = self.root / "outside"
        outside.write_bytes(b"do not replace")
        self.dest.parent.mkdir(parents=True)
        self.dest.symlink_to(outside)
        result = self.run_bootstrap()
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(outside.read_bytes(), b"do not replace")
        self.assertFalse(self.log.exists())
        self.dest.unlink()
        shutil.rmtree(self.skill / "bin")
        (self.skill / "bin").symlink_to(self.root / "outside-bin")
        result = self.run_bootstrap()
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse((self.root / "outside-bin").exists())

    def test_foreign_destination_is_preserved(self):
        self.dest.parent.mkdir(parents=True)
        self.dest.mkdir()
        result = self.run_bootstrap()
        self.assertNotEqual(result.returncode, 0)
        self.assertTrue(self.dest.is_dir())
        self.assertFalse(self.log.exists())
        self.assert_clean()

    def test_publication_does_not_follow_a_racing_destination_directory(self):
        real_ln = shutil.which("ln")
        self.tool("ln", f'#!/bin/sh\nmkdir "$TEST_DEST"\nexec "{real_ln}" "$@"\n')
        result = self.run_bootstrap(TEST_DEST=str(self.dest))
        self.assertNotEqual(result.returncode, 0)
        self.assertTrue(self.dest.is_dir())
        self.assertFalse(list(self.dest.iterdir()))
        self.assert_clean()

    def test_parallel_first_use_publishes_once(self):
        env = dict(self.env, DOWNLOAD_DELAY="0.15")
        a = subprocess.Popen(["/bin/sh", str(self.script)], env=env, text=True,
                             stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        b = subprocess.Popen(["/bin/sh", str(self.script)], env=env, text=True,
                             stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        try:
            aout, aerr = a.communicate(timeout=20)
            bout, berr = b.communicate(timeout=20)
        finally:
            for p in (a, b):
                if p.poll() is None:
                    p.kill()
                    p.communicate()
        self.assertEqual((a.returncode, b.returncode), (0, 0), aerr + berr)
        self.assertEqual(aout.strip(), str(self.dest))
        self.assertEqual(bout.strip(), str(self.dest))
        self.assertEqual(len(self.log.read_text().splitlines()), 2)
        self.assert_clean()

    def test_zip_capable_tar_fallback(self):
        tar = self.tools / "tar"
        if not tar.exists() or subprocess.run([str(tar), "-tf", str(self.assets / self.asset)],
                                             capture_output=True).returncode:
            self.skipTest("developer host tar does not support ZIP")
        (self.tools / "unzip").unlink()
        result = self.run_bootstrap()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.dest.read_bytes(), PAYLOAD)
        self.assert_clean()

    def test_missing_system_tools_are_explicit_blockers(self):
        (self.tools / "curl").unlink()
        result = self.run_bootstrap()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("no HTTPS downloader", result.stderr)
        self.assertFalse(self.log.exists())
        self.assertFalse(self.dest.exists())

    def test_rosetta_prefers_native_apple_silicon(self):
        self.tool("sysctl", '#!/bin/sh\necho 1\n')
        result = self.run_bootstrap(TEST_ARCH="x86_64")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.strip(), str(self.dest))
        self.assert_clean()

    def test_skill_symlink_resolves_to_canonical_source(self):
        link = self.root / "discovered skill"
        link.symlink_to(self.skill, target_is_directory=True)
        result = subprocess.run(["/bin/sh", str(link / "bootstrap.sh")], env=self.env,
                                text=True, capture_output=True, timeout=20)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.strip(), str(self.dest))
        self.assert_clean()

    def test_wget_and_sha256sum_fallbacks(self):
        (self.tools / "curl").unlink()
        self.tool("wget", self.downloader)
        if (self.tools / "shasum").exists():
            (self.tools / "shasum").unlink()
        if not (self.tools / "sha256sum").exists():
            self.skipTest("sha256sum unavailable on developer host")
        result = self.run_bootstrap()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.dest.read_bytes(), PAYLOAD)
        self.assert_clean()


if __name__ == "__main__":
    unittest.main()
