#!/usr/bin/env python3
"""Package a Framework/CLI release candidate without source links or publishing."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import shlex
import subprocess
from framework_archive import check_embedded_inputs, check_locked_checksums, committed_files, module_checksums, source_files, write_module_zip

ROOT = Path(__file__).resolve().parents[2]
MODULE = "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go"
ESCAPED = "github.com/!artisan!cloud/!power!x!plugin/framework/backend/go"


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
    parser.add_argument("--output", default="artifacts/framework-release-dev-20261026-2")
    parser.add_argument("--canonical-revision")
    parser.add_argument("--stable", action="store_true")
    args = parser.parse_args()
    embedded = ROOT / "tools/cli/internal/templates/data"
    check_embedded_inputs(embedded)
    init = (ROOT / "tools/cli/cmd/init.go").read_text()
    version = re.search(r'defaultFrameworkVersion\s*=\s*"([^"]+)"', init)[1]
    packages = {name: json.loads((ROOT / "framework/frontend/nuxt" / name / "package.json").read_text())
                for name in ["framework-client", "framework-admin"]}
    for value in [version, args.cli_version, *[p["version"] for p in packages.values()]]:
        if args.stable and not re.fullmatch(r"v?\d+\.\d+\.\d+", value):
            raise SystemExit("FRAMEWORK_STABLE_VERSION_REQUIRED:" + value)
        if not args.stable and "-dev." not in value:
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
    canonical_revision = None
    if args.canonical_revision:
        canonical_revision = run(["git", "rev-parse", args.canonical_revision], cwd=ROOT).strip()
        if trees["go"] != committed_files(ROOT, canonical_revision, "framework/backend/go"):
            raise SystemExit("FRAMEWORK_CANONICAL_SOURCE_MISMATCH")
    trees["builder"] = {name: (ROOT / "scripts/release" / name).read_bytes()
                        for name in ["prepare-framework-dev.py", "framework_archive.py", "validate-framework-dev.py"]}
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
    write_module_zip(module_zip, MODULE, version, trees["go"])
    checksums = module_checksums(bundle / "go-proxy", MODULE, version)
    check_locked_checksums(ROOT / "tools/cli/go.sum", MODULE, version, checksums)

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
        "schema": "framework_dev_bundle.v2", "published": False,
        "versions": {"go": version, "client": client_version, "admin": admin_version, "cli": args.cli_version},
        "base_commit": run(["git", "rev-parse", "HEAD"], cwd=ROOT).strip(),
        "source": "current_worktree_including_uncommitted_files", "source_sha256": hashes,
        "go_module": MODULE, "go_proxy": "file://" + str(bundle / "go-proxy"),
        "go_checksums": checksums, "go_canonical_git_revision": canonical_revision,
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
