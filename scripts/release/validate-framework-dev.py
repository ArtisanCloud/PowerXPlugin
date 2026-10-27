#!/usr/bin/env python3
"""Consume the exact dev artifacts in a generated plugin outside the monorepo."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile

MODULE = "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go"


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("bundle", type=Path)
    parser.add_argument("--public-go", action="store_true")
    args = parser.parse_args()
    bundle = args.bundle.resolve()
    refs = json.loads((bundle / "references.json").read_text())
    if refs.get("schema") != "framework_dev_bundle.v2":
        raise SystemExit("FRAMEWORK_DEV_BUNDLE_VERSION_INVALID")
    for name, expected in refs["artifacts"].items():
        if hashlib.sha256((bundle / name).read_bytes()).hexdigest() != expected:
            raise SystemExit("FRAMEWORK_ARTIFACT_CHECKSUM_MISMATCH:" + name)
    versions = refs["versions"]
    packs = {name: json.loads((bundle / (name + "-pack.json")).read_text())
             for name in ["framework-client", "framework-admin"]}
    client_ref = "file:" + str(bundle / packs["framework-client"]["filename"])
    admin_ref = "file:" + str(bundle / packs["framework-admin"]["filename"])
    scratch = Path(tempfile.mkdtemp(prefix="powerx-framework-dev-"))
    consumer = scratch / "plugin"
    env = dict(os.environ, GOWORK="off", GOFLAGS="-modcacherw", GOMODCACHE=str(scratch / "go-mod-cache"),
               GOPROXY=refs["go_proxy"] + "," + os.environ.get("GOPROXY", "https://proxy.golang.org,direct"),
               GONOSUMDB=MODULE, PX_PLUGIN_FRAMEWORK_VERSION=versions["go"],
               PX_PLUGIN_FRAMEWORK_ADMIN_REF=admin_ref, PX_PLUGIN_FRAMEWORK_CLIENT_REF=client_ref,
               NPM_CONFIG_AUDIT="false", NPM_CONFIG_FUND="false")
    env.pop("POWERXPLUGIN_USE_LOCAL_FRONTEND", None)
    if args.public_go:
        env.update(GOENV="off", GOPROXY="https://proxy.golang.org", GOSUMDB="sum.golang.org",
                   GOPRIVATE="", GONOPROXY="", GONOSUMDB="")
    results = {"schema": "framework_dev_validation.v2", "published": False,
               "versions": versions, "consumer_outside_repository": True,
               "go_work": "off", "isolated_go_module_cache": True, "local_replace": False,
               "source_symlinks": False, "checks": {}, "consumer_removed": False, "status": "failed"}
    results["public_go_proxy"] = args.public_go
    results["public_go_checksum_database"] = args.public_go
    try:
        with (bundle / "consumer-validation.log").open("w") as log:
            def run(name, command, cwd):
                log.write("\n" + name + "\n")
                log.flush()
                completed = subprocess.run(command, cwd=cwd, env=env, stdout=log, stderr=subprocess.STDOUT)
                results["checks"][name] = "passed" if completed.returncode == 0 else "failed"
                print(name + ":" + results["checks"][name], flush=True)
                completed.check_returncode()

            run("generated_plugin_install", [str(bundle / "px-plugin"), "init", "com.powerx.plugins.framework-dev-check",
                "--directory", str(consumer), "--backend", "go-gin", "--admin", "nuxt", "--app", "nuxt",
                "--git-init=false", "--init-config=false", "--install-deps"], scratch)
            backend, frontend = consumer / "backend", consumer / "web-admin"
            if any((backend / "bin").rglob("*")):
                raise RuntimeError("TEMPLATE_BUILD_ARTIFACT_NOT_ALLOWED")
            mod = (backend / "go.mod").read_text()
            if versions["go"] not in mod or "replace " + MODULE in mod:
                raise RuntimeError("FRAMEWORK_CONSUMER_GO_REFERENCE_INVALID")
            sums = (backend / "go.sum").read_text().splitlines()
            for suffix, expected in [("", refs["go_checksums"]["sum"]), ("/go.mod", refs["go_checksums"]["go_mod_sum"])]:
                if MODULE + " " + versions["go"] + suffix + " " + expected not in sums:
                    raise RuntimeError("FRAMEWORK_CONSUMER_CHECKSUM_MISMATCH")
            results["checks"]["installed_go_checksums"] = "passed"
            package = json.loads((frontend / "package.json").read_text())
            for name, expected in [("framework-client", client_ref), ("framework-admin", admin_ref)]:
                key = "@artisan-cloud/plugin-" + name
                if package["dependencies"][key] != expected:
                    raise RuntimeError("FRAMEWORK_CONSUMER_NPM_REFERENCE_INVALID:" + key)
                installed = frontend / "node_modules" / key
                if installed.is_symlink() or not installed.is_dir():
                    raise RuntimeError("FRAMEWORK_SOURCE_LINK_NOT_ALLOWED:" + key)
                with tarfile.open(bundle / packs[name]["filename"]) as archive:
                    for item in archive.getmembers():
                        if item.isfile():
                            raw = archive.extractfile(item).read()
                            actual = installed / Path(item.name).relative_to("package")
                            if actual.read_bytes() != raw:
                                raise RuntimeError("FRAMEWORK_INSTALLED_PACKAGE_MISMATCH:" + str(actual))
            results["checks"]["installed_packages_match_tarballs"] = "passed"
            run("manifest_validation", ["go", "run", "./cmd/manifestcheck", "--plugin", "../plugin.yaml", "--plugin-only"], backend)
            run("generated_backend_tests", ["go", "test", "./...", "-count=1"], backend)
            run("packaged_ai_contract_tests", ["go", "test", MODULE + "/runtime/aisettings", MODULE + "/runtime/powerx/ai", "-count=1"], backend)
            run("generated_backend_build", ["go", "build", "-o", str(scratch / "plugin-bin"), "./cmd/plugin"], backend)
            probe = frontend / "framework-exports-check.ts"
            probe.write_text('''import { createAIMediaClient, createPluginSSEClient, createImageDerivativesClient,
  businessNotificationCardContractV2, validateBusinessCardPresentation, createRegressionClient,
  type MediaTask, type ImageDerivatives, type BusinessCardPresentationV2
} from '@artisan-cloud/plugin-framework-client';
import { definePowerXAdminConfig } from '@artisan-cloud/plugin-framework-admin';
export const sdk = { createAIMediaClient, createPluginSSEClient, createImageDerivativesClient,
  businessNotificationCardContractV2, validateBusinessCardPresentation, createRegressionClient, definePowerXAdminConfig };
export type Contracts = [MediaTask, ImageDerivatives, BusinessCardPresentationV2];
''')
            run("installed_typescript_exports", ["node", "node_modules/typescript/bin/tsc", "--noEmit", "--strict", "--skipLibCheck",
                "--target", "ES2022", "--module", "ESNext", "--moduleResolution", "bundler", str(probe)], frontend)
            run("generated_nuxt_production_build", ["npm", "run", "build"], frontend)
            lock = json.loads((frontend / "package-lock.json").read_text())
            results["installed"] = {name: lock["packages"]["node_modules/" + name]["version"]
                                    for name in ["nuxt", "@artisan-cloud/plugin-framework-client", "@artisan-cloud/plugin-framework-admin"]}
            results["toolchain"] = {name: subprocess.check_output(command, text=True).strip()
                                    for name, command in [("go", ["go", "version"]), ("node", ["node", "--version"]), ("npm", ["npm", "--version"])]}
            results["status"] = "passed"
    finally:
        # Go's module cache may contain read-only directories, including when a
        # caller's toolchain ignores -modcacherw during a dependency download.
        for directory, _, _ in os.walk(scratch):
            Path(directory).chmod(0o755)
        shutil.rmtree(scratch)
        results["consumer_removed"] = True
        (bundle / "validation.json").write_text(json.dumps(results, indent=2) + "\n")
    refs["validation"] = "passed"
    (bundle / "references.json").write_text(json.dumps(refs, indent=2) + "\n")


if __name__ == "__main__":
    main()
