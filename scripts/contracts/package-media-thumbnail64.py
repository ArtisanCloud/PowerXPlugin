#!/usr/bin/env python3
"""Build an unpublished, content-addressed SDK/CLI bundle and consume it locally."""
import hashlib
import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess
import tarfile
import zipfile

ROOT = Path(__file__).resolve().parents[2]
MODULE = "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go"
ESCAPED_MODULE = "github.com/!artisan!cloud/!power!x!plugin/framework/backend/go"
AI_CONTRACT_FILES = (
    "ai-media-v2.md", "ai-media-v2.openapi.yaml", "ai-media-v2-source.json",
    "fixtures/ai-media-v2.json", "fixtures/ai-media-v2-registration.json",
    "fixtures/capability-version-upgrade-errors.v1.json",
)


def files_under(directory):
    return {p.relative_to(directory).as_posix(): p.read_bytes()
            for p in sorted(directory.rglob("*")) if p.is_file()
            and not any(part.startswith(".") for part in p.relative_to(directory).parts)}


def digest(files):
    value = hashlib.sha256()
    for name, raw in sorted(files.items()):
        value.update(name.encode() + b"\0" + raw)
    return value.hexdigest()


def run(args, **kwargs):
    return subprocess.run(args, check=True, **kwargs)


def manifest_bindings(path):
    result = run(["node", "--input-type=module", "-e",
                  "import fs from 'node:fs'; import YAML from 'yaml'; process.stdout.write(JSON.stringify(YAML.parse(fs.readFileSync(process.argv[1], 'utf8')).notification_presentations));",
                  str(path)], cwd=ROOT, capture_output=True, text=True)
    return json.loads(result.stdout)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--profile", choices=["thumbnail64", "ai-settings"], default="thumbnail64")
    profile = parser.parse_args().profile
    version_label = profile.replace("-", "")
    framework = files_under(ROOT / "framework/backend/go")
    client_root = ROOT / "framework/frontend/nuxt/framework-client"
    client_package = json.loads((client_root / "package.json").read_text())
    client = {name: (client_root / name).read_bytes() for name in client_package["files"]}
    cli = {}
    for name in ["cmd", "internal/templates/data", "internal/contracts", "internal/manifest", "internal/package"]:
        for path, raw in files_under(ROOT / "tools/cli" / name).items():
            if name == "cmd" and path.startswith("px-plugin/"):
                continue
            cli[name + "/" + path] = raw
    cli["bundle-builder.py"] = Path(__file__).read_bytes()
    hashes = {"go": digest(framework), "nuxt": digest(client), "generator": digest(cli),
              "contracts": hashlib.sha256((ROOT / "docs/contracts/media-thumbnail64-source.json").read_bytes()).hexdigest()}
    if profile == "ai-settings":
        hashes["ai_contracts"] = digest({name: (ROOT / "docs/contracts" / name).read_bytes() for name in AI_CONTRACT_FILES})
    bundle_id = hashlib.sha256(json.dumps(hashes, sort_keys=True).encode()).hexdigest()[:16]
    bundle = ROOT / "artifacts" / ("framework-release-" + profile + "-20261022-" + bundle_id)
    if bundle.exists():
        raise SystemExit("MEDIA_BUNDLE_ALREADY_EXISTS:" + str(bundle))
    bundle.mkdir(parents=True)
    version = "v0.0.27-dev." + version_label + "." + hashes["go"][:16]
    client_version = "0.0.14-dev." + version_label + "." + hashes["nuxt"][:16]
    cli_version = "v0.0.0-" + version_label + "." + hashes["generator"][:16]
    proxy = bundle / "go-proxy" / ESCAPED_MODULE / "@v"
    proxy.mkdir(parents=True)
    (proxy / (version + ".mod")).write_bytes(framework["go.mod"])
    (proxy / (version + ".info")).write_text(json.dumps({"Version": version, "Time": "2026-10-22T00:00:00Z"}))
    (proxy / "list").write_text(version + "\n")
    with zipfile.ZipFile(proxy / (version + ".zip"), "w", compression=zipfile.ZIP_DEFLATED) as archive:
        for name, raw in sorted(framework.items()):
            archive.writestr(MODULE + "@" + version + "/" + name, raw)
    staging = bundle / "client"
    staging.mkdir()
    for name, raw in client.items():
        (staging / name).write_bytes(raw)
    client_package["version"] = client_version
    (staging / "package.json").write_text(json.dumps(client_package, indent=2) + "\n")
    packed = run(["npm", "pack", "--json", "--pack-destination", str(bundle)], cwd=staging, capture_output=True, text=True)
    tarball = bundle / json.loads(packed.stdout)[0]["filename"]
    main_file = bundle / "build-main.go"
    main_file.write_text('''package main
import ("fmt"; "os"; "errors"; "github.com/powerx-plugin/cli/cmd")
func main() { if err:=cmd.Execute(os.Args[1:]);err!=nil {fmt.Fprintln(os.Stderr,err);code:=1;var coded interface{ExitCode()int};if errors.As(err,&coded){code=coded.ExitCode()};os.Exit(code)} }
''')
    binary = bundle / "px-plugin"
    run(["go", "build", "-ldflags", "-X github.com/powerx-plugin/cli/cmd.Version=" + cli_version,
         "-o", str(binary), str(main_file)], cwd=ROOT)
    source = json.loads((ROOT / "docs/contracts/media-thumbnail64-source.json").read_text())
    for name in list(source["sha256"]) + ["media-thumbnail64-source.json", "fixtures/notification-presentations.v2.yaml"]:
        target = bundle / "contracts" / name
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(ROOT / "docs/contracts" / name, target)
    if profile == "ai-settings":
        for name in AI_CONTRACT_FILES:
            target = bundle / "contracts" / name
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(ROOT / "docs/contracts" / name, target)
    consumer = bundle / "generated-consumer"
    env = dict(os.environ, GOWORK="off", GOPROXY="file://" + str(bundle / "go-proxy") + "," + os.environ.get("GOPROXY", "https://proxy.golang.org,direct"), GONOSUMDB=MODULE)
    with (bundle / "consumer-validation.log").open("w") as log:
        run([str(binary), "init", "com.powerx.plugins.thumbnail64-fixture", "--directory", str(consumer),
             "--backend", "go-gin", "--admin", "nuxt", "--app", "nuxt", "--git-init=false", "--init-config=false",
             "--notification-presentations", str(bundle / "contracts/fixtures/notification-presentations.v2.yaml"),
             "--framework-version", version, "--framework-client-ref", "file:" + str(tarball)],
            cwd=ROOT, stdout=log, stderr=subprocess.STDOUT)
        go_mod = (consumer / "backend/go.mod").read_text()
        assert version in go_mod and "replace " + MODULE not in go_mod
        bindings = manifest_bindings(consumer / "plugin.yaml")
        assert bindings == manifest_bindings(bundle / "contracts/fixtures/notification-presentations.v2.yaml")
        assert {(item["business_type"], item["contract"]) for item in bindings} == {
            (business, "business_notification_card." + contract)
            for business in ["sample_track", "bulk_order"] for contract in ["v1", "v2"]}
        assert len(bindings) == 4
        package = json.loads((consumer / "web-admin/package.json").read_text())
        assert package["dependencies"]["@artisan-cloud/plugin-framework-client"] == "file:" + str(tarball)
        run(["go", "run", "-mod=mod", "./cmd/manifestcheck", "--plugin", "../plugin.yaml", "--plugin-only"], cwd=consumer / "backend", env=env, stdout=log, stderr=subprocess.STDOUT)
        run(["go", "test", "-mod=mod", "./cmd/plugin", "./cmd/manifestcheck", "./internal/config",
             "./internal/services/runtimeexample", "./internal/transport/http/admin/host_contract",
             "./internal/services/ai_settings", "./internal/transport/http/admin/ai_settings"],
            cwd=consumer / "backend", env=env, stdout=log, stderr=subprocess.STDOUT)
        for name in ["business-notification-card.v1.json", "business-notification-card.v2.json", "media-image-derivatives.v2.json"]:
            assert (consumer / "docs/contracts/fixtures" / name).read_bytes() == (ROOT / "docs/contracts/fixtures" / name).read_bytes()
        for name in ["business-notification-card.openapi.yaml", "image-derivatives.openapi.yaml", "media-host.openapi.yaml"]:
            assert (consumer / "docs/contracts" / name).read_bytes() == (ROOT / "tools/cli/internal/contracts/data/media-contracts" / name).read_bytes()
        if profile == "ai-settings":
            run(["go", "test", "-mod=mod", MODULE + "/runtime/powerx/ai", "-run",
                 "TestMediaCatalogAroundLegacyTenantRegistrationUpgrade|TestUpgrade"],
                cwd=consumer / "backend", env=env, stdout=log, stderr=subprocess.STDOUT)
            for name in AI_CONTRACT_FILES:
                assert (consumer / "docs/contracts" / name).read_bytes() == (ROOT / "docs/contracts" / name).read_bytes()
            ai_source = json.loads((consumer / "docs/contracts/ai-media-v2-source.json").read_text())
            assert len(ai_source["sha256"]) == 4
            for name, expected in ai_source["sha256"].items():
                assert hashlib.sha256((consumer / "docs/contracts" / name).read_bytes()).hexdigest() == expected
            assert (consumer / "web-admin/app/utils/aiSettingsTesting.ts").is_file()
        tsc = ROOT / "skeleton/web-admin/nuxt/node_modules/typescript/bin/tsc"
        run(["node", str(tsc), "--noEmit", "--strict", "--skipLibCheck", "--target", "ES2022", "--module", "ESNext",
             "--moduleResolution", "bundler", str(staging / "notifications.ts"), str(staging / "image-derivatives.ts")], cwd=ROOT, stdout=log, stderr=subprocess.STDOUT)
    with tarfile.open(tarball) as archive:
        assert archive.getmember("package/image-derivatives.ts")
        assert archive.getmember("package/notifications.ts")
    refs = {"bundle": str(bundle), "profile": profile, "published": False, "changes_committed": False,
            "base_commit": run(["git", "rev-parse", "HEAD"], cwd=ROOT, capture_output=True, text=True).stdout.strip(),
            "source_sha256": hashes, "go_module": MODULE, "go_sdk_version": version, "go_proxy": "file://" + str(bundle / "go-proxy"),
            "nuxt_client_version": client_version, "nuxt_client_ref": "file:" + str(tarball), "nuxt_client_tarball": str(tarball),
            "generator_version": cli_version, "generator_binary": str(binary),
            "consumer": {"path": str(consumer), "go_work": "off", "local_replace": False, "four_bindings": "passed",
                         "manifest_validation": "passed", "backend_build": "passed", "shared_fixtures": "passed", "packed_typescript": "passed"},
            "artifact_sha256": {str(p.relative_to(bundle)): hashlib.sha256(p.read_bytes()).hexdigest()
                                for p in [binary, tarball, proxy / (version + ".zip")]}}
    (bundle / "references.json").write_text(json.dumps(refs, indent=2) + "\n")
    print(json.dumps(refs, indent=2))


if __name__ == "__main__":
    main()
