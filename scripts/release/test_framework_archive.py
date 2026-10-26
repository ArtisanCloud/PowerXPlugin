import json
from pathlib import Path
import re
import subprocess
import tempfile
import unittest

from framework_archive import check_embedded_inputs, check_locked_checksums, committed_files, module_checksums, source_files, write_module_zip


class FrameworkArchiveTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="powerx-archive-fixture-")
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.git("init", "--quiet")
        (self.root / ".gitignore").write_text("*.out\nbin/\n*.log\n.DS_Store\n__pycache__/\n")
        self.sdk = self.root / "sdk"
        (self.sdk / "iam").mkdir(parents=True)
        (self.sdk / "iam/.gitkeep").write_bytes(b"")
        (self.sdk / "go.mod").write_text("module example.invalid/framework\n\ngo 1.20\n")
        (self.sdk / "sdk.go").write_text("package framework\n")
        self.git("add", ".")
        self.git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid",
                 "commit", "--quiet", "-m", "fixture")

    def git(self, *args):
        return subprocess.check_output(["git", "-C", str(self.root), *args], stderr=subprocess.PIPE)

    def proxy(self, name, files, module="example.invalid/framework", version="v0.0.1-dev.fixture"):
        root = self.root / name
        escaped = "".join("!" + c.lower() if "A" <= c <= "Z" else c for c in module)
        directory = root / escaped / "@v"
        directory.mkdir(parents=True)
        (directory / (version + ".mod")).write_bytes(files["go.mod"])
        (directory / (version + ".info")).write_text(json.dumps({"Version": version, "Time": "2026-10-26T00:00:00Z"}))
        write_module_zip(directory / (version + ".zip"), module, version, files)
        return root, version

    def test_hidden_git_inputs_are_kept_and_coverage_is_excluded(self):
        (self.sdk / "coverage.out").write_text("ignored coverage\n")
        (self.sdk / "bin").mkdir()
        (self.sdk / "bin/plugin").write_bytes(b"ignored binary")
        actual = source_files(self.sdk)
        self.assertIn("iam/.gitkeep", actual)
        self.assertNotIn("coverage.out", actual)
        self.assertNotIn("bin/plugin", actual)
        self.assertEqual(actual, committed_files(self.root, "HEAD", "sdk"))

    def test_uncommitted_source_changes_are_preserved(self):
        (self.sdk / "sdk.go").write_text("package framework\nconst Revision = 2\n")
        (self.sdk / "new.go").write_text("package framework\n")
        actual = source_files(self.sdk)
        self.assertEqual(actual["sdk.go"], (self.sdk / "sdk.go").read_bytes())
        self.assertIn("new.go", actual)
        self.assertNotEqual(actual, committed_files(self.root, "HEAD", "sdk"))

    def test_go_checksums_match_git_archive_despite_ignored_files(self):
        (self.sdk / "coverage.out").write_text("ignored coverage\n")
        working_proxy, version = self.proxy("working-proxy", source_files(self.sdk))
        git_proxy, _ = self.proxy("git-proxy", committed_files(self.root, "HEAD", "sdk"))
        actual = module_checksums(working_proxy, "example.invalid/framework", version)
        expected = module_checksums(git_proxy, "example.invalid/framework", version)
        self.assertEqual(actual, expected)
        self.assertTrue(actual["sum"].startswith("h1:"))
        self.assertTrue(actual["go_mod_sum"].startswith("h1:"))

    def test_ignored_files_cannot_be_embedded_in_cli(self):
        embedded = self.root / "templates"
        embedded.mkdir()
        (embedded / "view.tmpl").write_text("{{ .PluginID }}")
        self.git("add", "templates/view.tmpl")
        check_embedded_inputs(embedded)
        for name in ["coverage.out", ".DS_Store", "audit.log", "__pycache__/compiled.pyc"]:
            path = embedded / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(b"ignored")
            with self.assertRaisesRegex(ValueError, "TEMPLATE_IGNORED_INPUT_NOT_ALLOWED"):
                check_embedded_inputs(embedded)
            path.unlink()

    def test_repository_sdk_matches_cli_checksum_lock(self):
        repository = Path(__file__).resolve().parents[2]
        module = "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go"
        manifest = (repository / "tools/cli/go.mod").read_text()
        version = re.search(re.escape(module) + r"\s+(\S+)", manifest)[1]
        proxy, _ = self.proxy("repository-proxy", source_files(repository / "framework/backend/go"), module, version)
        checksums = module_checksums(proxy, module, version)
        check_locked_checksums(repository / "tools/cli/go.sum", module, version, checksums)


if __name__ == "__main__":
    unittest.main()
