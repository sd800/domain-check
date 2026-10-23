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
        self.temp = tempfile.TemporaryDirectory(prefix="domain-check-installer-test-")
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
        self.archive = f"domain-check_{self.version}_darwin_arm64.tar.gz"
        self.python_tool("uname", 'print("Darwin" if sys.argv[1] == "-s" else "arm64")')
        self.python_tool("sw_vers", 'print("13.0")')
        self.python_tool("plutil", 'value=json.loads(Path(sys.argv[-1]).read_text())[sys.argv[2]]; print(json.dumps(value) if isinstance(value, (list, dict)) else value)')
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
            script = script.replace('mv -fh ', 'mv -fT ')
        self.script = self.root / "install.sh"
        self.script.write_text(script)
        self.env = dict(os.environ, DOMAIN_INSTALL_TEST_FIXTURES=str(self.fixture), TMPDIR=str(self.root),
                        DOMAIN_CONFIG_DIR=str(self.root / "configuration"),
                        DOMAIN_CACHE_DIR=str(self.root / "cache"))
        self.make_archive()

    def python_tool(self, name, code):
        tool = self.tools / name
        tool.write_text(f"#!{sys.executable}\nimport sys, os, json\nfrom pathlib import Path\n" + code + "\n")
        tool.chmod(0o755)

    def make_archive(self, attack=None):
        entry = 'domain' if attack == 'legacy-flat' else 'domain-check'
        self.archive = f"{entry}_{self.version}_darwin_arm64.tar.gz"
        (self.fixture / "release.json").write_text(json.dumps({"tag_name": "v" + self.version, 'assets': [{'name': self.archive}]}))
        with tarfile.open(self.fixture / self.archive, "w:gz") as archive:
            names = [entry, "README.md", "CHANGELOG.md", "LICENSE", "VERSION"]
            if attack != "legacy-flat":
                directory = tarfile.TarInfo("docs/")
                directory.type = tarfile.DIRTYPE
                directory.mode = 0o755
                archive.addfile(directory)
                names += ["docs/technical.md", "docs/domains.md", "docs/security.md",
                          "docs/quality-assurance.md", "docs/references.md"]
            for name in names:
                info = tarfile.TarInfo(name)
                body = b"synthetic executable" if name == entry else b"documentation"
                info.size = len(body)
                info.mode = 0o755 if name == entry else 0o644
                if name == entry and attack == "symlink":
                    info.type, info.linkname, info.size = tarfile.SYMTYPE, "../outside", 0
                    archive.addfile(info)
                elif name == entry and attack == "hardlink":
                    info.type, info.linkname, info.size = tarfile.LNKTYPE, "README.md", 0
                    archive.addfile(info)
                else:
                    archive.addfile(info, io.BytesIO(body))
            if attack in ("traversal", "duplicate", "nested-doc", "both-names"):
                extra = tarfile.TarInfo({"traversal": "../outside", "duplicate": entry, "nested-doc": "README/md", 'both-names': 'domain'}[attack])
                extra.size = 6
                archive.addfile(extra, io.BytesIO(b"attack"))
        digest = hashlib.sha256((self.fixture / self.archive).read_bytes()).hexdigest()
        (self.fixture / f"checksums_{self.version}.txt").write_text(f"{digest}  {self.archive}\n")

    def run_installer(self, *args, success=True):
        result = subprocess.run(["/bin/sh", str(self.script), *args], cwd=self.root, env=self.env,
                                capture_output=True, text=True, timeout=15)
        self.assertEqual(result.returncode == 0, success, result.stdout + result.stderr)
        self.assertFalse(list(self.target.glob(".domain-check-install.*")), "temporary executable leaked")
        return result

    def test_valid_download_and_repeatable_path(self):
        self.run_installer()
        self.run_installer()
        self.assertEqual((self.target / "domain").read_bytes(), b"synthetic executable")
        self.assertEqual((self.target / 'domain').readlink(), Path('domain-check'))
        self.assertFalse((self.target / 'domain-check').is_symlink())
        self.assertEqual(self.profile.read_text().count('export PATH="$HOME/.local/bin:$PATH"'), 1)

    def test_previous_flat_release_remains_installable(self):
        self.make_archive("legacy-flat")
        self.run_installer()
        self.assertEqual((self.target / "domain").read_bytes(), b"synthetic executable")

    def test_installed_executable_refreshes_its_own_web_service(self):
        source = self.root / "source"
        marker = self.root / "refresh-calls"
        source.write_text("#!/bin/sh\nprintf '%s\\n' \"$1\" >> " + shlex.quote(str(marker)) + "\n")
        self.run_installer(str(source))
        self.assertEqual(marker.read_text().splitlines(), ["--version", "--web-refresh"])

    def test_failed_web_refresh_preserves_installation_and_path_setup(self):
        source = self.root / "source"
        source.write_text('#!/bin/sh\n[ "$1" != "--web-refresh" ]\n')
        result = self.run_installer(str(source))
        self.assertIn('domain setup', result.stderr)
        self.assertEqual((self.target / "domain").read_bytes(), source.read_bytes())
        self.assertEqual(self.profile.read_text().count('export PATH="$HOME/.local/bin:$PATH"'), 1)

    def test_unsupported_macos_never_replaces_the_executable(self):
        destination = self.target / "domain"
        destination.write_bytes(b"previous executable")
        for version in ["12.7.6", "10.15.7", "unknown"]:
            with self.subTest(macos=version):
                self.python_tool("sw_vers", f"print({version!r})")
                self.run_installer(success=False)
                self.assertEqual(destination.read_bytes(), b"previous executable")
                self.assertFalse(self.profile.exists())

    def test_untrusted_archives_preserve_previous_installation(self):
        for attack in ["traversal", "duplicate", "symlink", "hardlink", "nested-doc", "both-names"]:
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
        self.assertEqual((self.target / "domain").readlink(), Path('domain-check'))
        (self.target / "domain").unlink()
        directory = self.root / "unrelated-directory"
        directory.mkdir()
        (self.target / "domain").symlink_to(directory)
        self.run_installer("-m")
        self.assertEqual(list(directory.iterdir()), [])

    def test_real_executable_symlink_is_replaced_without_following(self):
        source = self.root / 'source'
        source.write_bytes(b'source executable')
        outside = self.root / 'unrelated'
        outside.write_bytes(b'untouched')
        (self.target / 'domain-check').symlink_to(outside)
        self.run_installer(str(source))
        self.assertEqual(outside.read_bytes(), b'untouched')
        self.assertFalse((self.target / 'domain-check').is_symlink())
        self.assertEqual((self.target / 'domain').read_bytes(), source.read_bytes())

    def test_binary_only_install_preserves_shell_and_service_configuration(self):
        source = self.root / 'source'
        source.write_bytes(b'source executable')
        self.run_installer('--binary-only', str(source), str(self.target))
        self.assertEqual((self.target / 'domain').readlink(), Path('domain-check'))
        self.assertEqual((self.target / 'domain-check').read_bytes(), source.read_bytes())
        self.assertFalse(self.profile.exists())
        self.assertFalse((self.root / 'configuration').exists())

    def test_permission_failure_never_requests_elevation(self):
        source = self.root / 'source'
        source.write_bytes(b'source executable')
        previous = self.target / 'domain'
        previous.write_bytes(b'previous executable')
        marker = self.root / 'privilege-request'
        for tool in ['sudo', 'osascript', 'pkexec']:
            self.python_tool(tool, f'Path({str(marker)!r}).write_text("requested"); sys.exit(1)')
        self.python_tool('install', 'raise PermissionError("synthetic permission denied")')
        self.run_installer(str(source), success=False)
        self.assertFalse(marker.exists(), 'installer requested elevated privileges')
        self.assertEqual(previous.read_bytes(), b'previous executable')
        self.assertFalse((self.target / 'domain-check').exists())
        self.assertFalse(self.profile.exists())

    def test_failed_source_install_and_directory_target(self):
        destination = self.target / "domain"
        destination.write_bytes(b"previous executable")
        self.run_installer("missing-source", success=False)
        self.assertEqual(destination.read_bytes(), b"previous executable")
        destination.unlink()
        source = self.root / "source"
        source.write_bytes(b"source executable")
        for name in ['domain', 'domain-check']:
            with self.subTest(destination=name):
                destination = self.target / name
                destination.mkdir()
                self.run_installer("source", success=False)
                self.assertEqual(list(destination.iterdir()), [])
                destination.rmdir()


if __name__ == "__main__":
    unittest.main()
