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
import sys

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "scripts/release"))
from framework_archive import source_files as files_under, write_module_zip, module_checksums, check_embedded_inputs
MODULE = "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go"
ESCAPED_MODULE = "github.com/!artisan!cloud/!power!x!plugin/framework/backend/go"
AI_CONTRACT_FILES = (
    "ai-online-model-catalog.md", "ai-settings-llm-catalog.v1.md", "ai-online-model-catalog-source.json",
    "ai-local-model-catalog.v1.openapi.yaml", "ai-local-model-catalog.v1.md", "fixtures/ai-local-model-catalog.v1.json",
    "ai-settings-connection-test.md", "ai-connection-test.v1.openapi.yaml", "ai-connection-test-source.json", "fixtures/ai-connection-test.v1.json",
    "ai-media-v2.md", "ai-media-v2.openapi.yaml", "ai-media-v2-source.json", "ai-media-model-limits-framework.md",
    "fixtures/ai-media-v2.json", "fixtures/ai-media-v2-registration.json", "fixtures/ai-media-model-limits.v1.json",
    "fixtures/capability-version-upgrade-errors.v1.json",
    "plugin-service-authorization.md", "plugin-service-authorization.openapi.yaml",
    "plugin-authorization-source.json", "plugin-authorization-framework.md",
    "fixtures/plugin-authorization.v1.json", "fixtures/plugin-service-errors-stream.v1.json",
)


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



def validate_packed_media_limits(bundle, tarball):
    packed_dir = bundle / "packed-client"
    packed_dir.mkdir()
    with tarfile.open(tarball) as archive:
        for name in json.loads((bundle / "client/package.json").read_text())["files"] + ["package.json"]:
            (packed_dir / name).write_bytes(archive.extractfile("package/" + name).read())
    fixture = json.loads((bundle / "contracts/fixtures/ai-media-model-limits.v1.json").read_text())
    assert fixture["fixture_owner"] == "Framework"
    fields = ["max_image_bytes", "max_total_image_bytes", "max_image_pixels"]
    assert all(field + "?: number" in (packed_dir / "ai-media.ts").read_text() for field in fields)
    spec = (ROOT / "skeleton/web-admin/nuxt/tests/unit/framework-ai-media.spec.ts").read_text()
    spec = spec.replace("../../../../../framework/frontend/nuxt/framework-client", "./packed-client/index")
    spec = spec.replace("../../../../../docs/contracts/fixtures/", "./contracts/fixtures/")
    (bundle / "packed-ai-media.spec.ts").write_text(spec)
    types = bundle / "packed-limits-types.ts"
    types.write_text('''import { createAIMediaClient, type MediaModelOperation } from './packed-client/index';
declare const operation: MediaModelOperation;
const values: Array<number | undefined> = [operation.max_image_bytes, operation.max_total_image_bytes, operation.max_image_pixels];
type Catalog = Awaited<ReturnType<ReturnType<typeof createAIMediaClient>['listModels']>>;
declare const catalog: Catalog;
const bytes: number | undefined = catalog.items[0]?.operations.image_generation?.max_image_bytes;
void values; void bytes;
''')
    nuxt = ROOT / "skeleton/web-admin/nuxt"
    with (bundle / "packed-client-validation.log").open("w") as log:
        run(["node", str(nuxt / "node_modules/vitest/dist/cli.js"), "run", "--config", str(nuxt / "vitest.config.ts"),
             "--root", str(bundle), "packed-ai-media.spec.ts"], cwd=ROOT, stdout=log, stderr=subprocess.STDOUT)
        run(["node", str(nuxt / "node_modules/typescript/bin/tsc"), "--noEmit", "--strict", "--skipLibCheck",
             "--target", "ES2022", "--module", "ESNext", "--moduleResolution", "bundler", str(types)],
            cwd=ROOT, stdout=log, stderr=subprocess.STDOUT)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--profile", choices=["thumbnail64", "ai-settings", "plugin-service", "media-limits"], default="thumbnail64")
    profile = parser.parse_args().profile
    includes_ai_contracts = profile in {"ai-settings", "plugin-service", "media-limits"}
    version_label = profile.replace("-", "")
    check_embedded_inputs(ROOT / "tools/cli/internal/contracts/data")
    check_embedded_inputs(ROOT / "tools/cli/internal/templates/data")
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
    cli["framework_archive.py"] = (ROOT / "scripts/release/framework_archive.py").read_bytes()
    hashes = {"go": digest(framework), "nuxt": digest(client), "generator": digest(cli),
              "contracts": hashlib.sha256((ROOT / "docs/contracts/media-thumbnail64-source.json").read_bytes()).hexdigest()}
    if includes_ai_contracts:
        hashes["ai_contracts"] = digest({name: (ROOT / "docs/contracts" / name).read_bytes() for name in AI_CONTRACT_FILES})
    bundle_id = hashlib.sha256(json.dumps(hashes, sort_keys=True).encode()).hexdigest()[:16]
    bundle_date = "20261107" if profile == "media-limits" else ("20261102" if profile == "plugin-service" else "20261022")
    bundle = ROOT / "artifacts" / ("framework-release-" + profile + "-" + bundle_date + "-" + bundle_id)
    if bundle.exists():
        raise SystemExit("MEDIA_BUNDLE_ALREADY_EXISTS:" + str(bundle))
    bundle.mkdir(parents=True)
    version_base = "v0.0.31-dev." if profile == "media-limits" else ("v0.0.29-dev." if profile == "plugin-service" else "v0.0.27-dev.")
    version = version_base + version_label + "." + hashes["go"][:16]
    client_version = ("0.0.15-dev." if profile == "media-limits" else "0.0.14-dev.") + version_label + "." + hashes["nuxt"][:16]
    cli_version = "v0.0.0-" + version_label + "." + hashes["generator"][:16]
    proxy = bundle / "go-proxy" / ESCAPED_MODULE / "@v"
    proxy.mkdir(parents=True)
    (proxy / (version + ".mod")).write_bytes(framework["go.mod"])
    snapshot_time = "2026-11-07T00:00:00Z" if profile == "media-limits" else ("2026-11-02T00:00:00Z" if profile == "plugin-service" else "2026-10-22T00:00:00Z")
    (proxy / (version + ".info")).write_text(json.dumps({"Version": version, "Time": snapshot_time}))
    (proxy / "list").write_text(version + "\n")
    write_module_zip(proxy / (version + ".zip"), MODULE, version, framework)
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
    if includes_ai_contracts:
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
        if includes_ai_contracts:
            run(["go", "test", "-mod=mod", MODULE + "/runtime/ai/modeldiscovery"],
                cwd=consumer / "backend", env=env, stdout=log, stderr=subprocess.STDOUT)
            assert "com.corex.ai.connection.test" in (consumer / "plugin.d/capabilities.yaml").read_text()
            run(["go", "test", "-mod=mod", MODULE + "/runtime/aisettings", "-run", "TestConnection"],
                cwd=consumer / "backend", env=env, stdout=log, stderr=subprocess.STDOUT)
            run(["go", "test", "-mod=mod", MODULE + "/runtime/powerx/ai", "-run",
                 "TestMediaCatalogAroundLegacyTenantRegistrationUpgrade|TestUpgrade|TestMediaModelLimits"],
                cwd=consumer / "backend", env=env, stdout=log, stderr=subprocess.STDOUT)
            for name in AI_CONTRACT_FILES:
                assert (consumer / "docs/contracts" / name).read_bytes() == (ROOT / "docs/contracts" / name).read_bytes()
            connection_source = json.loads((consumer / "docs/contracts/ai-connection-test-source.json").read_text())
            for name, expected in connection_source["sha256"].items():
                assert hashlib.sha256((consumer / "docs/contracts" / name).read_bytes()).hexdigest() == expected
            ai_source = json.loads((consumer / "docs/contracts/ai-media-v2-source.json").read_text())
            assert len(ai_source["sha256"]) == 4
            for name, expected in ai_source["sha256"].items():
                assert hashlib.sha256((consumer / "docs/contracts" / name).read_bytes()).hexdigest() == expected
            assert (consumer / "web-admin/app/utils/aiSettingsTesting.ts").is_file()
        if profile == "plugin-service":
            run(["go", "test", "-mod=mod", MODULE + "/gateway", MODULE + "/runtime/powerx/hostcontract",
                 MODULE + "/runtime/powerx/capability", MODULE + "/runtime/powerx/ai", MODULE + "/runtime/capability",
                 MODULE + "/runtime/ai"], cwd=consumer / "backend", env=env, stdout=log, stderr=subprocess.STDOUT)
            service_source = json.loads((consumer / "docs/contracts/plugin-authorization-source.json").read_text())
            for name, expected in service_source["sha256"].items():
                assert hashlib.sha256((consumer / "docs/contracts" / name).read_bytes()).hexdigest() == expected
            service_fixture = service_source["framework_service_fixture"]
            assert hashlib.sha256((consumer / "docs/contracts" / service_fixture["file"]).read_bytes()).hexdigest() == service_fixture["sha256"]
        tsc = ROOT / "skeleton/web-admin/nuxt/node_modules/typescript/bin/tsc"
        run(["node", str(tsc), "--noEmit", "--strict", "--skipLibCheck", "--target", "ES2022", "--module", "ESNext",
             "--moduleResolution", "bundler", str(staging / "notifications.ts"), str(staging / "image-derivatives.ts"), str(staging / "ai-connection.ts"), str(staging / "ai-model-catalog.ts"), str(staging / "ai-media.ts")], cwd=ROOT, stdout=log, stderr=subprocess.STDOUT)
    with tarfile.open(tarball) as archive:
        assert archive.getmember("package/image-derivatives.ts")
        assert archive.getmember("package/notifications.ts")
        assert archive.getmember("package/ai-connection.ts")
        assert archive.getmember("package/ai-model-catalog.ts")
        assert archive.getmember("package/ai-media.ts")
    if profile == "media-limits":
        validate_packed_media_limits(bundle, tarball)
    checksums = module_checksums(bundle / "go-proxy", MODULE, version)
    refs = {"bundle": str(bundle), "profile": profile, "published": False, "changes_committed": False,
            "base_commit": run(["git", "rev-parse", "HEAD"], cwd=ROOT, capture_output=True, text=True).stdout.strip(),
            "source_sha256": hashes, "go_module": MODULE, "go_sdk_version": version, "go_proxy": "file://" + str(bundle / "go-proxy"),
            "nuxt_client_version": client_version, "nuxt_client_ref": "file:" + str(tarball), "nuxt_client_tarball": str(tarball),
            "generator_version": cli_version, "generator_binary": str(binary),
            "go_checksums": checksums,
            "consumer": {"path": str(consumer), "go_work": "off", "local_replace": False, "four_bindings": "passed",
                         "manifest_validation": "passed", "backend_build": "passed", "shared_fixtures": "passed", "packed_typescript": "passed"},
            "artifact_sha256": {str(p.relative_to(bundle)): hashlib.sha256(p.read_bytes()).hexdigest()
                                for p in [binary, tarball, proxy / (version + ".zip")]}}
    (bundle / "references.json").write_text(json.dumps(refs, indent=2) + "\n")
    if profile == "media-limits":
        refs["consumer"]["packed_media_limits_runtime_and_types"] = "passed"
        refs["nuxt_client_integrity"] = json.loads(packed.stdout)[0]["integrity"]
        refs["capability_protocol_changes"] = False
        (bundle / "references.json").write_text(json.dumps(refs, indent=2) + "\n")
        (ROOT / "artifacts/ai-media-model-limits-latest.json").write_text(json.dumps(refs, indent=2) + "\n")
    print(json.dumps(refs, indent=2))


if __name__ == "__main__":
    main()
