"""Run the Framework SDK against Core's owned PostgreSQL, TaskBus and MinIO fixture."""

import argparse
import hashlib
import json
import os
import shutil
from pathlib import Path
import subprocess
import sys
import tarfile
from tempfile import TemporaryDirectory
from zipfile import ZipFile



def build_fixture_package(root, fixture, framework_root, env):
    fixture.mkdir()
    (fixture / "go.mod").write_text("module powerx.storage.dataset.fixture\n\ngo 1.24.0\n")
    for target in ("plugin", "database"):
        cmd = fixture / "cmd" / target
        cmd.mkdir(parents=True)
        shutil.copyfile(root / "testdata/package_main.go.fixture", cmd / "main.go")
    assets = fixture / "contracts/storage-datasets"
    assets.mkdir(parents=True)
    raw = json.dumps({
        "dataset_key": "reports", "display_name_i18n": {"en": "Archive fixture", "zh-CN": "归档样本"},
        "contract_version": 1, "schema_contract_version": 1, "tenant_scope": "tenant",
        "source_binding_ref": "database", "adapter_key": "reports", "adapter_version": 1,
        "allowed_actions": ["observe", "preview", "archive"],
        "retention_bounds": {"minimum_days": 7, "suggested_days": 30, "maximum_days": 90},
    }).encode()
    (assets / "reports.json").write_bytes(raw)
    digest = hashlib.sha256(raw).hexdigest()
    (fixture / "plugin.yaml").write_text(f"""id: com.powerx.plugins.storage-package-fixture
name: Storage archive fixture
version: 1.0.0
runtime:
  kind: process
  entry: backend/bin/plugin
  env:
    POWERX_BIND_ADDR: "127.0.0.1:__POWERX_DYNAMIC_PORT__"
backend:
  entry: backend/bin/plugin
  port: 0
  health: /healthz
endpoints:
  http_base_path: /api/v1
frontend:
  admin:
    kind: static
    static_dir: web-admin/.output
migrations:
  driver: go
  entry: backend/bin/migrate
  args: ["migrate"]
  once: true
  timeout: 15s
capabilities:
  required:
    - com.corex.storage.datasets.register
    - com.corex.storage.datasets.preview_worker
    - com.corex.storage.datasets.execution_worker
    - com.corex.runtime.taskqueue.service_consume
data_lifecycle:
  reports:
    sha256: "{digest}"
    observation_tables: [reports]
""")
    skeleton = framework_root / "skeleton"
    (fixture / "make-files").mkdir()
    shutil.copyfile(skeleton / "make-files/build.mk", fixture / "make-files/build.mk")
    scripts = fixture / "scripts/manifest"
    scripts.mkdir(parents=True)
    for name in ("dist-target.mjs", "dist-target.locales.json"):
        shutil.copyfile(skeleton / "scripts/manifest" / name, scripts / name)
    frontend = fixture / "web-admin"
    frontend.mkdir()
    (frontend / "package.json").write_text("{}")
    local_deps = frontend / "node_modules"
    local_deps.mkdir()
    yaml = framework_root / "node_modules/yaml"
    if not yaml.exists():
        yaml = skeleton / "web-admin/nuxt/node_modules/yaml"
    if not yaml.exists():
        raise RuntimeError("Framework build must have its declared yaml package installed")
    (local_deps / "yaml").symlink_to(yaml.resolve())
    (fixture / "Makefile").write_text("""PLUGIN_ID := com.powerx.plugins.storage-package-fixture
VERSION := 1.0.0
GO_MODULE := powerx.storage.dataset.fixture
PLUGIN_MANIFEST := plugin.yaml
DIST_BACKEND_BIN = $(DIST_DIR)/backend/bin
DIST_VERIFY := 0
include make-files/build.mk

# The fixture backend uses the real Go build. Its frontend is one static file.
frontend-build:
	@mkdir -p web-admin/.output
	@printf 'dataset archive fixture' > web-admin/.output/index.html

plugin-yaml-check plugin-permission-declaration-check:
	@test -s plugin.yaml
""")
    subprocess.run(["go", "work", "use", str(fixture)], env=env, cwd=fixture, check=True)
    subprocess.run(["make", "package"], env=env, cwd=fixture, check=True)
    archive = fixture / "com.powerx.plugins.storage-package-fixture-1.0.0.zip"
    with ZipFile(archive) as package:
        if package.read("1.0.0/contracts/storage-datasets/reports.json") != raw:
            raise RuntimeError("dataset declaration changed during packaging")
        for binary in ("plugin", "migrate"):
            package.getinfo(f"1.0.0/backend/bin/{binary}")
    # The management upload accepts tar.gz or a directory. Keep the byte-for-byte
    # ZIP assertion above, then use the same verified dist tree for the upload.
    upload = fixture / f"{archive.stem}.tar.gz"
    with tarfile.open(upload, "w:gz") as package:
        package.add(fixture / "dist/1.0.0", arcname="1.0.0")
    return upload


def packaged_test_overlay(source):
    source = source.replace(
        'external != "" {\n\t\tt.Run("independent-postgres"',
        'external != "" && os.Getenv("POWERX_STORAGE_DATASET_TEST_PACKAGE") == "" {\n\t\tt.Run("independent-postgres"', 1,
    ).replace(
        "const pluginID =",
        """var installedPackage pm.Manager
        var packageConfig *config.Config
        var packageSupervisor *pluginSupervisor.Supervisor
        packagePath := os.Getenv("POWERX_STORAGE_DATASET_TEST_PACKAGE")
        const pluginID =""", 1,
    ).replace(
        "if id != pluginID {",
        """if installedPackage != nil && id == packagedDatasetID {
            return installedPackage.Get(ctx, id)
        }
        if id != pluginID {""", 1,
    ).replace(
        "Installed: func(_ context.Context, id string)",
        "Installed: func(ctx context.Context, id string)", 1,
    ).replace(
        "Binding: func(pm.Plugin) (uuid.UUID, string, string, error) { return binding, pluginSchema, pluginRole, nil },",
        """Binding: func(p pm.Plugin) (uuid.UUID, string, string, error) {
            if p.ID == packagedDatasetID {
                return packageManager.ManagedDatasetBinding(p, string(packageConfig.Deployment.Env), packageConfig.Database)
            }
            return binding, pluginSchema, pluginRole, nil
        },""", 1,
    ).replace(
        "UserID: 1, IsRoot: true, TenantUUID: tenant.UUID.String()",
        "UserID: 1, UserUUID: uuid.NewString(), IsRoot: true, TenantUUID: tenant.UUID.String()", 1,
    ).replace(
        "server := httptest.NewServer(engine)",
        """server := httptest.NewUnstartedServer(engine)
        if packagePath != "" && external == "" {
            installedPackage, packageConfig, packageSupervisor = configurePackagedDatasetFixture(t, ctx, dsn, engine, group,
                server, deps, queue, credentials, credentialOptions, tenant.UUID.String(), listener.Addr().String(), secret)
        }
        server.Start()""", 1,
    ).replace(
        'op := func(key string)',
        """if installedPackage != nil {
            exercisePackagedDataset(t, ctx, db, call, server, installedPackage, packageConfig, packageSupervisor,
                packagePath, tenant.UUID.String(), otherTenant.UUID.String(), root, archiveTarget)
            return
        }
        op := func(key string)""", 1,
    )
    return source.replace("import (", """import (
        "mime/multipart"
        packageManager "github.com/ArtisanCloud/PowerX/internal/infra/plugin/manager"
        pluginRouter "github.com/ArtisanCloud/PowerX/internal/infra/plugin/manager/router"
        pluginSupervisor "github.com/ArtisanCloud/PowerX/internal/infra/plugin/manager/supervisor"
        pluginHTTP "github.com/ArtisanCloud/PowerX/internal/transport/http/admin/plugin"
        pluginInstall "github.com/ArtisanCloud/PowerX/internal/service/plugin_install"
        pluginService "github.com/ArtisanCloud/PowerX/internal/service/plugin"
        installModel "github.com/ArtisanCloud/PowerX/pkg/corex/db/persistence/model/plugin_install"
        corexdb "github.com/ArtisanCloud/PowerX/pkg/corex/db"
    """, 1)

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("core", type=Path, help="Core checkout with the dataset live fixture")
    parser.add_argument("--with-package", action="store_true", help="test the real package upload, migration, and supervisor startup")
    args = parser.parse_args()
    framework = Path(__file__).resolve().parents[2] / "framework/backend/go"
    core = args.core.resolve()
    contracts = core / "specs/027-monitor-center/contracts"
    for name in ("plugin-dataset-host.v1.schema.json", "plugin-dataset-execution.v1.schema.json"):
        embedded = framework / "runtime/storagedataset/contracts" / name
        if embedded.read_bytes() != (contracts / name).read_bytes():
            parser.error(f"Framework and Core contract differ: {name}")
    live_test = core / "backend/internal/bootstrap/storage_datasets_live_test.go"
    original = live_test.read_text()
    start = original.index("\t\tpreviewPath = exerciseStorageDatasetPreview(")
    end = original.index("\n\t}\n\tcheck := base", start)
    consumer = """
        clusterID, sourceDSN := "owned", dsn
        if external != "" {
            clusterID, sourceDSN = "owned-independent-postgres", external
        }
        previewPath = exerciseFrameworkDataset(t, ctx, sourceDB, call, server, tables,
            installation, pluginSchema, pluginRole, clusterID, sourceDSN, base+"/"+fixture.DatasetUUID.String(),
            tenant.UUID.String(), sts, exchange(otherTenantID, otherTenantSecret), root, archiveTarget, binding, fixture, func() {
                const executionControl = "/admin/ops/storage/execution-control"
                call("POST", executionControl+"/changes", root, gov.ExecutionChange{
                    RequestID: uuid.NewString(), Action: "pause", Confirm: true, Reason: "Isolated Framework reconciliation",
                }, 200)
                pending, err := queue.Dequeue(ctx, "platform:storage-control", "core.storage.control."+scope.ID(),
                    "owned-dataset-config", "test", 1, time.Second)
                require.NoError(t, err)
                require.Len(t, pending, 1)
                require.NoError(t, control.ApplyDelivery(ctx, pending[0]))
                require.Contains(t, string(call("GET", executionControl, root, nil, 200)), `"mode":"paused"`)
            })
"""
    consumer_imports = """
        "sync"
        datasetfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/storagedataset"
        fwgateway "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
        fwqueue "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/taskqueue"
        "github.com/jackc/pgx/v5/stdlib"
"""
    fixture = framework / "runtime/storagedataset/testdata/core_worker.go.fixture"
    combined = original[:start] + consumer + original[end:]
    combined = combined.replace("import (", "import (" + consumer_imports, 1) + "\n" + fixture.read_text()
    with TemporaryDirectory(prefix="powerx-framework-dataset-", dir="/tmp") as tmp:
        private = Path(tmp).resolve()
        test = private / "storage_datasets_live_test.go"
        if args.with_package:
            combined = packaged_test_overlay(combined) + (fixture.parent / "package_core.go.fixture").read_text()
        test.write_text(combined)
        subprocess.run(["gofmt", "-w", str(test)], check=True)
        overlay = private / "overlay.json"
        overlay.write_text(json.dumps({"Replace": {str(live_test): str(test)}}))
        env = dict(os.environ, GOWORK="off", GOFLAGS="")
        subprocess.run(["go", "work", "init", str(core / "backend"), str(framework)],
                       cwd=private, env=env, check=True)
        env["GOWORK"] = str(private / "go.work")
        if args.with_package:
            archive = build_fixture_package(fixture.parent.parent, private / "fixture", framework.parents[2], env)
            env["POWERX_STORAGE_DATASET_TEST_PACKAGE"] = str(archive)
        env["GOFLAGS"] = f"-overlay={overlay}"
        subprocess.run([sys.executable, str(core / "scripts/dev/check-storage-datasets.py"), "--with-minio"],
                       cwd=core, env=env, check=True)


if __name__ == "__main__":
    main()
