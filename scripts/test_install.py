#!/usr/bin/env python3
"""Exercise installer trust boundaries with synthetic downloads and isolated paths."""

import hashlib
import io
import json
import os
from pathlib import Path
import shlex
import subprocess
import sys
import tarfile
import tempfile
import unittest


SOURCE = Path(__file__).with_name("install.sh").read_text()


class InstallerSecurityTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="domain-installer-test-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.tools = self.root / "tools"
        self.tools.mkdir()
        self.target = self.root / "install"
        self.target.mkdir()
        self.profile = self.root / "zshrc"
        self.fixture = self.root / "downloads"
        self.fixture.mkdir()
        self.version = "26.10.21.26"
        self.archive = f"domain_{self.version}_darwin_arm64.tar.gz"
        self.python_tool("uname", 'print("Darwin" if sys.argv[1] == "-s" else "arm64")')
        self.python_tool("plutil", 'print(json.loads(Path(sys.argv[-1]).read_text())["tag_name"])')
        self.python_tool("curl", '''
assert sys.argv[1:6] == ["-q", "--proto", "=https", "--proto-redir", "=https"]
assert "--max-time" in sys.argv and "--max-filesize" in sys.argv
url = sys.argv[-1]
assert url.startswith("https://")
destination = Path(sys.argv[sys.argv.index("-o")+1])
source = Path(os.environ["DOMAIN_INSTALL_TEST_FIXTURES"]) / url.rsplit("/", 1)[-1]
if url.endswith("/releases/latest"):
    source = source.with_name("release.json")
destination.write_bytes(source.read_bytes())
''')
        # Only the test copy redirects tools and installation paths. The actual
        # installer retains its fixed macOS PATH and never receives a fake HOME.
        script = SOURCE.replace(
            "PATH=/usr/bin:/bin:/usr/sbin:/sbin",
            f"PATH={shlex.quote(str(self.tools))}:/usr/bin:/bin:/usr/sbin:/sbin",
        ).replace('"$HOME/.local/bin"', f'"{self.target}"').replace('"$HOME/.local/bin/', f'"{self.target}/')
        script = script.replace('profile="${ZDOTDIR:-$HOME}/.zshrc"', f"profile={shlex.quote(str(self.profile))}")
        script = script.replace("/usr/bin/plutil", str(self.tools / "plutil"))
        if sys.platform != "darwin":
            script = script.replace('mv -fh "$pending"', 'mv -fT "$pending"')
        self.script = self.root / "install.sh"
        self.script.write_text(script)
        self.env = dict(os.environ, DOMAIN_INSTALL_TEST_FIXTURES=str(self.fixture), TMPDIR=str(self.root))
        self.make_archive()

    def python_tool(self, name, code):
        tool = self.tools / name
        tool.write_text(f"#!{sys.executable}\nimport sys, os, json\nfrom pathlib import Path\n" + code + "\n")
        tool.chmod(0o755)

    def make_archive(self, attack=None):
        (self.fixture / "release.json").write_text(json.dumps({"tag_name": "v" + self.version}))
        with tarfile.open(self.fixture / self.archive, "w:gz") as archive:
            for name in ["domain", "README.md", "CHANGELOG.md", "LICENSE", "VERSION", "SECURITY.md"]:
                info = tarfile.TarInfo(name)
                body = b"synthetic executable" if name == "domain" else b"documentation"
                info.size = len(body)
                info.mode = 0o755 if name == "domain" else 0o644
                if name == "domain" and attack == "symlink":
                    info.type, info.linkname, info.size = tarfile.SYMTYPE, "../outside", 0
                    archive.addfile(info)
                elif name == "domain" and attack == "hardlink":
                    info.type, info.linkname, info.size = tarfile.LNKTYPE, "README.md", 0
                    archive.addfile(info)
                else:
                    archive.addfile(info, io.BytesIO(body))
            if attack in ("traversal", "duplicate", "nested-doc"):
                extra = tarfile.TarInfo({"traversal": "../outside", "duplicate": "domain", "nested-doc": "README/md"}[attack])
                extra.size = 6
                archive.addfile(extra, io.BytesIO(b"attack"))
        digest = hashlib.sha256((self.fixture / self.archive).read_bytes()).hexdigest()
        (self.fixture / f"checksums_{self.version}.txt").write_text(f"{digest}  {self.archive}\n")

    def run_installer(self, *args, success=True):
        result = subprocess.run(["/bin/sh", str(self.script), *args], cwd=self.root, env=self.env,
                                capture_output=True, text=True, timeout=15)
        self.assertEqual(result.returncode == 0, success, result.stdout + result.stderr)
        self.assertFalse(list(self.target.glob(".domain-install.*")), "temporary executable leaked")
        return result

    def test_valid_download_and_repeatable_path(self):
        self.run_installer()
        self.run_installer()
        self.assertEqual((self.target / "domain").read_bytes(), b"synthetic executable")
        self.assertEqual(self.profile.read_text().count('export PATH="$HOME/.local/bin:$PATH"'), 1)

    def test_untrusted_archives_preserve_previous_installation(self):
        for attack in ["traversal", "duplicate", "symlink", "hardlink", "nested-doc"]:
            with self.subTest(attack=attack):
                self.make_archive(attack)
                (self.target / "domain").write_bytes(b"previous executable")
                self.run_installer(success=False)
                self.assertEqual((self.target / "domain").read_bytes(), b"previous executable")
                self.assertFalse((self.root / "outside").exists())
                self.assertFalse(self.profile.exists())

    def test_bad_checksum_and_duplicate_manifest(self):
        checksum = self.fixture / f"checksums_{self.version}.txt"
        good = checksum.read_text()
        for body in ["0" * 64 + f"  {self.archive}\n", good + good, f"$(id)  {self.archive}\n", good + f"invalid  {self.archive}\n"]:
            with self.subTest(manifest=body[:20]):
                checksum.write_text(body)
                self.run_installer(success=False)
                self.assertFalse((self.target / "domain").exists())

    def test_release_tag_cannot_inject_paths_or_commands(self):
        for tag in ["v../../outside", "v26.10.21.1$(touch marker)", "--output=outside", "v26.10.21.1\nother"]:
            with self.subTest(tag=tag):
                (self.fixture / "release.json").write_text(json.dumps({"tag_name": tag}))
                self.run_installer(success=False)
                self.assertFalse((self.root / "marker").exists())

    def test_source_option_name_and_destination_symlinks(self):
        source = self.root / "-m"
        source.write_bytes(b"source executable")
        outside = self.root / "unrelated"
        outside.write_bytes(b"untouched")
        (self.target / "domain").symlink_to(outside)
        self.run_installer("-m")
        self.assertEqual(outside.read_bytes(), b"untouched")
        self.assertFalse((self.target / "domain").is_symlink())
        (self.target / "domain").unlink()
        directory = self.root / "unrelated-directory"
        directory.mkdir()
        (self.target / "domain").symlink_to(directory)
        self.run_installer("-m")
        self.assertEqual(list(directory.iterdir()), [])

    def test_failed_source_install_and_directory_target(self):
        destination = self.target / "domain"
        destination.write_bytes(b"previous executable")
        self.run_installer("missing-source", success=False)
        self.assertEqual(destination.read_bytes(), b"previous executable")
        destination.unlink()
        destination.mkdir()
        source = self.root / "source"
        source.write_bytes(b"source executable")
        self.run_installer("source", success=False)
        self.assertEqual(list(destination.iterdir()), [])


if __name__ == "__main__":
    unittest.main()
