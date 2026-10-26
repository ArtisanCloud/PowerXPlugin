#!/usr/bin/env python3
"""Package an unpublished Framework/CLI snapshot without source links or publishing."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import shlex
import subprocess
import zipfile

ROOT = Path(__file__).resolve().parents[2]
MODULE = "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go"
ESCAPED = "github.com/!artisan!cloud/!power!x!plugin/framework/backend/go"
EXCLUDED = {"node_modules", "bin", "obj", "vendor", "__pycache__", "test-results"}


def source_files(directory):
    return {p.relative_to(directory).as_posix(): p.read_bytes()
            for p in sorted(directory.rglob("*"))
            if p.is_file() and not p.is_symlink()
            and not any(part in EXCLUDED or part.startswith(".")
                        for part in p.relative_to(directory).parts[:-1])
            and (not p.name.startswith(".") or p.name.endswith(".tmpl"))
            and p.name != "px-plugin"}


def digest(files):
    sha = hashlib.sha256()
    for name, raw in sorted(files.items()):
        sha.update(name.encode() + b"\0" + raw)
    return sha.hexdigest()


def run(args, **kwargs):
    return subprocess.run(args, check=True, capture_output=True, text=True, **kwargs).stdout


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cli-version", required=True)
    parser.add_argument("--output", default="artifacts/framework-release-dev-20261026")
    args = parser.parse_args()
    embedded = ROOT / "tools/cli/internal/templates/data"
    for path in embedded.rglob("*"):
        if path.is_file() and any(part in {"bin", "node_modules", ".nuxt", ".output"}
                                  for part in path.relative_to(embedded).parts):
            raise SystemExit("TEMPLATE_BUILD_ARTIFACT_NOT_ALLOWED:" + str(path))
    init = (ROOT / "tools/cli/cmd/init.go").read_text()
    version = re.search(r'defaultFrameworkVersion\s*=\s*"([^"]+)"', init)[1]
    packages = {name: json.loads((ROOT / "framework/frontend/nuxt" / name / "package.json").read_text())
                for name in ["framework-client", "framework-admin"]}
    for value in [version, args.cli_version, *[p["version"] for p in packages.values()]]:
        if "-dev." not in value:
            raise SystemExit("FRAMEWORK_DEV_VERSION_REQUIRED:" + value)
    client_version = packages["framework-client"]["version"]
    admin_version = packages["framework-admin"]["version"]
    if packages["framework-admin"]["peerDependencies"][packages["framework-client"]["name"]] != client_version:
        raise SystemExit("FRAMEWORK_PEER_VERSION_MISMATCH")
    for constant, expected in [("defaultClientVersion", client_version), ("defaultAdminVersion", admin_version)]:
        if re.search(constant + r'\s*=\s*"([^"]+)"', init)[1] != expected:
            raise SystemExit("FRAMEWORK_DEFAULT_VERSION_MISMATCH:" + constant)

    trees = {
        "go": source_files(ROOT / "framework/backend/go"),
        "client": source_files(ROOT / "framework/frontend/nuxt/framework-client"),
        "admin": source_files(ROOT / "framework/frontend/nuxt/framework-admin"),
        "cli": source_files(ROOT / "tools/cli"),
    }
    trees["builder"] = {"prepare-framework-dev.py": Path(__file__).read_bytes()}
    hashes = {name: digest(files) for name, files in trees.items()}
    bundle_id = hashlib.sha256(json.dumps(hashes, sort_keys=True).encode()).hexdigest()[:16]
    bundle = (ROOT / args.output / bundle_id).resolve()
    if bundle.exists():
        raise SystemExit("FRAMEWORK_DEV_BUNDLE_EXISTS:" + str(bundle))
    bundle.mkdir(parents=True)
    proxy = bundle / "go-proxy" / ESCAPED / "@v"
    proxy.mkdir(parents=True)
    (proxy / (version + ".mod")).write_bytes(trees["go"]["go.mod"])
    (proxy / (version + ".info")).write_text(json.dumps({"Version": version, "Time": "2026-10-26T00:00:00Z"}))
    (proxy / "list").write_text(version + "\n")
    module_zip = proxy / (version + ".zip")
    with zipfile.ZipFile(module_zip, "w", compression=zipfile.ZIP_DEFLATED) as archive:
        for name, raw in sorted(trees["go"].items()):
            info = zipfile.ZipInfo(MODULE + "@" + version + "/" + name, (2026, 10, 26, 0, 0, 0))
            info.compress_type = zipfile.ZIP_DEFLATED
            info.external_attr = 0o100644 << 16
            archive.writestr(info, raw)

    tarballs = {}
    for name, package in packages.items():
        packed = json.loads(run(["npm", "pack", "--json", "--ignore-scripts", "--pack-destination", str(bundle)],
                                cwd=ROOT / "framework/frontend/nuxt" / name))[0]
        tarballs[name] = bundle / packed["filename"]
        (bundle / (name + "-pack.json")).write_text(json.dumps(packed, indent=2) + "\n")
    binary = bundle / "px-plugin"
    run(["go", "build", "-ldflags", "-X main.version=" + args.cli_version,
         "-o", str(binary), "./tools/cli/cmd/px-plugin"], cwd=ROOT)

    exports = {
        "GOWORK": "off",
        "PX_PLUGIN_FRAMEWORK_VERSION": version,
        "PX_PLUGIN_FRAMEWORK_CLIENT_REF": "file:" + str(tarballs["framework-client"]),
        "PX_PLUGIN_FRAMEWORK_ADMIN_REF": "file:" + str(tarballs["framework-admin"]),
    }
    activation = ["#!/bin/sh"] + ["export " + key + "=" + shlex.quote(value) for key, value in exports.items()]
    activation += [
        "export GOPROXY=" + shlex.quote("file://" + str(bundle / "go-proxy") + ",") + '"${GOPROXY:-https://proxy.golang.org,direct}"',
        "export GONOSUMDB=" + shlex.quote(MODULE) + '${GONOSUMDB:+,$GONOSUMDB}',
        "export PATH=" + shlex.quote(str(bundle) + ":") + '"$PATH"',
    ]
    (bundle / "activate.sh").write_text("\n".join(activation) + "\n")
    (bundle / "init.sh").write_text("#!/bin/sh\nset -eu\n. " + shlex.quote(str(bundle / "activate.sh")) +
                                    "\nexec " + shlex.quote(str(binary)) + ' init "$@"\n')
    (bundle / "init.sh").chmod(0o755)
    references = {
        "schema": "framework_dev_bundle.v1", "published": False,
        "versions": {"go": version, "client": client_version, "admin": admin_version, "cli": args.cli_version},
        "base_commit": run(["git", "rev-parse", "HEAD"], cwd=ROOT).strip(),
        "source": "current_worktree_including_uncommitted_files", "source_sha256": hashes,
        "go_module": MODULE, "go_proxy": "file://" + str(bundle / "go-proxy"),
        "artifacts": {str(p.relative_to(bundle)): hashlib.sha256(p.read_bytes()).hexdigest()
                      for p in [module_zip, *tarballs.values(), binary]},
        "activation": str(bundle / "activate.sh"), "init": str(bundle / "init.sh"),
        "validation": "pending",
    }
    (bundle / "references.json").write_text(json.dumps(references, indent=2) + "\n")
    manifest = {name: {path: hashlib.sha256(raw).hexdigest() for path, raw in files.items()}
                for name, files in trees.items()}
    (bundle / "source-manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")
    print(json.dumps({"bundle": str(bundle), "versions": references["versions"]}))


if __name__ == "__main__":
    main()
