"""Select Git source inputs and validate module archives with the Go toolchain."""
import io
import json
import os
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile
import zipfile


def source_files(directory):
    directory = Path(directory).resolve()
    raw = subprocess.check_output([
        "git", "-C", str(directory), "ls-files", "--cached", "--others",
        "--exclude-standard", "-z", "--", ".",
    ])
    files = {}
    for name in sorted(set(os.fsdecode(value) for value in raw.split(b"\0") if value)):
        path = directory / name
        if path.is_file() and not path.is_symlink():
            files[name] = path.read_bytes()
    return files


def committed_files(repository, revision, subdirectory):
    raw = subprocess.check_output([
        "git", "-C", str(repository), "archive", "--format=tar",
        revision + ":" + subdirectory,
    ])
    with tarfile.open(fileobj=io.BytesIO(raw)) as archive:
        return {item.name: archive.extractfile(item).read()
                for item in archive.getmembers() if item.isfile()}


def check_embedded_inputs(directory):
    selected = source_files(directory)
    for path in Path(directory).rglob("*"):
        if not path.is_file():
            continue
        name = path.relative_to(directory).as_posix()
        if any(part in {"bin", "node_modules", ".nuxt", ".output"} for part in Path(name).parts):
            raise ValueError("TEMPLATE_BUILD_ARTIFACT_NOT_ALLOWED:" + name)
        if name not in selected:
            raise ValueError("TEMPLATE_IGNORED_INPUT_NOT_ALLOWED:" + name)


def write_module_zip(destination, module, version, files):
    with zipfile.ZipFile(destination, "w", compression=zipfile.ZIP_DEFLATED) as archive:
        for name, raw in sorted(files.items()):
            info = zipfile.ZipInfo(module + "@" + version + "/" + name, (2026, 10, 26, 0, 0, 0))
            info.compress_type = zipfile.ZIP_DEFLATED
            info.external_attr = 0o100644 << 16
            archive.writestr(info, raw)


def check_locked_checksums(lockfile, module, version, checksums):
    lines = Path(lockfile).read_text().splitlines()
    for suffix, key in [("", "sum"), ("/go.mod", "go_mod_sum")]:
        if module + " " + version + suffix + " " + checksums[key] not in lines:
            raise ValueError("FRAMEWORK_LOCKED_CHECKSUM_MISMATCH:" + module + "@" + version + suffix)


def module_checksums(proxy, module, version):
    scratch = Path(tempfile.mkdtemp(prefix="powerx-module-checksum-"))
    try:
        (scratch / "go.mod").write_text("module example.invalid/framework-archive-check\n\ngo 1.24.0\n")
        env = dict(os.environ, GOWORK="off", GOTOOLCHAIN="local", GOFLAGS="-modcacherw",
                   GOMODCACHE=str(scratch / "cache"), GOPROXY=Path(proxy).resolve().as_uri(),
                   GONOSUMDB=module)
        result = subprocess.run(["go", "mod", "download", "-json", module + "@" + version],
                                cwd=scratch, env=env, capture_output=True, text=True)
        if result.returncode:
            raise ValueError("FRAMEWORK_MODULE_CHECKSUM_FAILED:" + result.stdout + result.stderr)
        data = json.loads(result.stdout)
        if data.get("Path") != module or data.get("Version") != version or not data.get("Sum") or not data.get("GoModSum"):
            raise ValueError("FRAMEWORK_MODULE_CHECKSUM_INVALID")
        return {"sum": data["Sum"], "go_mod_sum": data["GoModSum"]}
    finally:
        for directory, _, _ in os.walk(scratch):
            Path(directory).chmod(0o755)
        shutil.rmtree(scratch)
