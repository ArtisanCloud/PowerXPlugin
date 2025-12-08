# 能力注册流程

本节描述如何从零开始为 `template` CRUD 能力生成契约、导出多协议 artefacts、提交审核并完成租户授权。

## 0. 前置条件

- PowerX CLI (`px-plugin`) ≥ 0.4.0，Node.js ≥ 18，Go ≥ 1.24。
- 已通过 `px-plugin init` 生成脚手架工程，包含 `plugin.yaml`、`contracts/` 与 `.px-plugin/` 目录。
- 拥有 Dev API 凭证：`PX_DEV_API_BASE`、`PX_DEV_API_TOKEN`。

## 1. 声明能力

使用 CLI 初始化能力契约：

```bash
px-plugin capabilities init com.powerx.demo.template.create \
  --method POST \
  --description "创建模板"
```

命令会：
- 更新 `plugin.yaml` 的 `capabilities.imports/provides`、`exposure.channels`、`agent_tools`；
- 写入 `contracts/capabilities/com.powerx.demo.template.create.yaml`；
- 生成 `contracts/schema/input|output/com.powerx.demo.template.create.json`；
- 在 `backend/internal/handlers/capabilities` 生成/更新 handler stub（若使用脚手架默认模板）。

## 2. 校验契约

```bash
node scripts/capabilities/validate-capabilities.mjs --manifest ./plugin.yaml
# 或
px-plugin capabilities lint --manifest ./plugin.yaml
```

校验会检查 catalog 与 artefacts 是否一致、JSON Schema 是否存在、ID 是否命名正确。`npm run test` 也会默认加载 `./backend/etc/plugin.yaml`（如果不存在则回退到根目录），可通过 `CAP_MANIFEST=<path>` 覆盖。

## 3. 导出多协议资产

```bash
npm --prefix scripts/capabilities run export
# 等价于 make capabilities-export
```

导出脚本会读取 `capabilities/catalog.json` 与 `contracts/exposure/exposure-packages.json`，并生成：

| 目录 | 内容 | 场景 |
|------|------|------|
| `contracts/exposure/openapi.yaml` | REST/Webhook 摘要 | API Portal / Gateway |
| `contracts/exposure/proto/*.proto` | gRPC Service | gRPC Gateway、SDK |
| `contracts/exposure/workflow/*.json` | Workflow Step | Workflow Builder |
| `contracts/exposure/agent-streams/*.yaml` | Agent SSE/推送 | Agent Hub |
| `contracts/exposure/mcp-tools.json` | MCP Manifest | Agent 工具目录 |
| `dist/agent-sdk/manifest.json` | SDK/工具清单 | SDK 包/测试工具 |

若能力是复合任务，还需维护 `contracts/exposure/composites/*.json`，导出脚本会据此同步 Workflow/Agent Stream 产物。

## 4. 提交审核 / 同步暴露配置

```bash
px-plugin capabilities submit \
  --manifest ./plugin.yaml \
  --base-url "$PX_DEV_API_BASE" \
  --token "$PX_DEV_API_TOKEN"
```

CLI 会：
- 调用能力中心 API (`POST /internal/plugins/capabilities`、`PATCH /internal/plugins/capabilities/{id}/exposure`)；
- 更新 `.px-plugin/capabilities.json`、`.px-plugin/audit/*.log`；
- 在状态为 `pending/rejected` 时阻断 `px-plugin dist/publish`。

> 可使用 `--capability-id` 仅提交部分能力。

## 5. 租户授权与额度

```bash
px-plugin capabilities quota \
  --capability-id com.powerx.demo.template.create \
  --tenant demo \
  --base-url "$PX_DEV_API_BASE" \
  --token "$PX_DEV_API_TOKEN" \
  --qps 20 --burst 40 --limits 1000
```

该命令会调用 `POST /internal/plugins/capabilities/{id}/tenants/{tenantId}/quota` 并生成示例：

- `dist/capabilities/<id>/samples/tenant-<tenant>-quota.postman.json`
- `dist/capabilities/<id>/samples/tenant-<tenant>-quota.http`

## 6. Skeleton 插件验证（可选）

在 `repo-root/skeleton` 目录可按以下顺序验证：

1. `node ../scripts/capabilities/discover-handlers.mjs --plugin . --handlers backend/internal/transport/http/admin/templates`
2. `npx --yes tsx ../tools/cli/src/commands/capabilities/init.ts --manifest ./plugin.yaml --capability-id com.powerx.skeleton.template.create ...`
3. `npx --yes tsx ../tools/cli/src/commands/capabilities/lint.ts --manifest ./plugin.yaml`
4. `CAP_MANIFEST=./plugin.yaml npm test`
5. `npx --yes tsx ../tools/cli/src/commands/capabilities/submit.ts --manifest ./plugin.yaml ...`
6. `npx --yes tsx ../tools/cli/src/commands/capabilities/quota.ts --capability-id com.powerx.skeleton.template.create --tenant sandbox ...`
7. `cd backend && GOFLAGS=-mod=mod GOWORK=off go test ./...`
8. `VERSION=0.2.0 make dist && make local-install`
9. `npx --yes tsx ../tools/cli/src/commands/capabilities/diff.ts --from HEAD~1:plugin.yaml --to plugin.yaml --output release/capabilities-change-report.md`

## 7. Init 输出工程（全新插件）

当你使用 `px-plugin init` 生成新插件仓库时，可在仓库根目录执行：

1. `px-plugin init com.example.capability --backend go-gin --admin nuxt --force`
2. `npm install && go mod tidy ./backend`
3. `node ../scripts/capabilities/discover-handlers.mjs --plugin .`（可选）
4. `npx --yes tsx ../tools/cli/src/commands/capabilities/init.ts --manifest ./plugin.yaml --capability-id com.example.capability.createItem ...`
5. `npx --yes tsx ../tools/cli/src/commands/capabilities/lint.ts --manifest ./plugin.yaml`
6. `CAP_MANIFEST=./plugin.yaml npm test`
7. `npx --yes tsx ../tools/cli/src/commands/capabilities/submit.ts ...`
8. `npx --yes tsx ../tools/cli/src/commands/capabilities/quota.ts ...`
9. `npx --yes tsx ../tools/cli/src/commands/capabilities/diff.ts ...`

完成后即可在 PowerX 中安装插件并验证模板能力。
