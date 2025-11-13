# PowerX Publish Hub · 能力注册与暴露指南

本指南帮助插件开发者通过 px-plugin CLI 在 manifest/contract 中声明能力，并向 PowerX 能力中心注册、授权与灰度发布。

---

## 前置条件

- PowerX CLI (`px-plugin`) ≥ 0.4.0，Node.js ≥ 18，Go ≥ 1.24。
- 已通过 `px-plugin init` 生成脚手架工程，包含 `plugin.yaml`、`contracts/` 与 `.px-plugin/`。
- 拥有 Dev API/能力中心访问凭证：`PX_DEV_API_BASEURL`、`PX_DEV_API_TOKEN`。

---

## 定义能力

- 使用脚手架：`px-plugin capabilities init com.powerx.demo.template.create --method POST --description "创建模板"`。
- 命令会自动更新：
  - `plugin.yaml` 的 `capabilities.provides`、`agent_tools`、`exposure.channels`;
  - `contracts/capabilities/<id>.yaml`;
  - `contracts/schema/input|output/<id>.json`;
  - `backend/internal/handlers/capabilities/...` handler stub。
- 若需手动调整，可编辑上述文件并重新运行 `px-plugin capabilities lint`。

---

## 生成并校验契约

```bash
# 自动生成/校验缺失的 descriptor / schema
node scripts/capabilities/validate-capabilities.mjs --manifest ./plugin.yaml

# 或在 make 中
VALIDATE_MANIFEST=./plugin.yaml make validate
```

`lint` 会检测以下问题：
- 缺少 descriptor 或 JSON Schema；
- `agent_tools` / `exposure.channels` 引用了不存在的 capability；
- descriptor 中的 ID 与 manifest 不一致。

---

## 提交审核

```bash
px-plugin capabilities submit \
  --manifest ./plugin.yaml \
  --base-url https://dev-api.powerx.local \
  --token $PX_DEV_API_TOKEN
```

- CLI 会调用 `POST /internal/plugins/capabilities` 与 `PATCH /internal/plugins/capabilities/{id}/exposure`；
- `.px-plugin/capabilities.json` 会记录最新状态；
- `.px-plugin/audit/*.log` 保留审计轨迹；
- 若状态为 pending/rejected，`px-plugin dist/publish` 将自动阻断。

> 可使用 `--capability-id <id>` 单独提交某个能力。

---

## 租户授权与额度

```bash
px-plugin capabilities quota \
  --capability-id com.powerx.demo.template.create \
  --tenant demo \
  --base-url https://dev-api.powerx.local \
  --token $PX_DEV_API_TOKEN \
  --qps 20 --burst 40 --limits 1000
```

CLI 将调用 `POST /internal/plugins/capabilities/{id}/tenants/{tenantId}/quota`，同时生成 Postman/SDK 示例（未来版本）。

生成的示例位于：

- `dist/capabilities/<capabilityId>/samples/tenant-<tenant>-quota.postman.json`
- `dist/capabilities/<capabilityId>/samples/tenant-<tenant>-quota.http`

可直接导入 Postman 或复制 HTTP 请求供 SDK/测试使用。

---

## Skeleton 模式：仓库内置样板

适合你直接在仓库中的 `skeleton/` 工程（或由它复制出的插件仓库）验证能力。**请确保 `skeleton/` 下已经存在 `plugin.yaml`（可以先运行一次 `px-plugin init` 并将产物放入该目录）。** 以下步骤全部在 `repo-root/skeleton` 内执行，若你在自己的插件根目录，也可照此顺序替换路径。

1. **进入样板根目录**
   ```bash
   cd skeleton
   ```
2. **扫描既有 handler → 生成能力清单**  
   该命令会读取 `backend/internal/transport/http/admin/templates/template_handler.go` 中的 `// capability:` 标记，输出需要注册的能力。
   ```bash
   # 在 skeleton/ 下执行
   node ../scripts/capabilities/discover-handlers.mjs \
     --plugin . \
     --handlers backend/internal/transport/http/admin/templates
   ```
3. **声明/更新能力契约**  
   ```bash
   # 在 skeleton/ 下执行
   npx --yes tsx ../tools/cli/src/commands/capabilities/init.ts \
     --manifest ./plugin.yaml \
     --capability-id com.powerx.skeleton.template.create \
     --method POST \
     --description "Skeleton 模板能力"
   ```
4. **运行 lint / schema 校验**
   ```bash
   npx --yes tsx ../tools/cli/src/commands/capabilities/lint.ts --manifest ./plugin.yaml
   CAP_MANIFEST=./plugin.yaml npm test
   ```
5. **向能力中心提交 + 同步暴露配置**
   ```bash
   npx --yes tsx ../tools/cli/src/commands/capabilities/submit.ts \
     --manifest ./plugin.yaml \
     --base-url https://dev-api.powerx.local \
     --token $PX_DEV_API_TOKEN \
     --root-dir .
   cat .px-plugin/capabilities.json
   ```
6. **分配租户额度 + 生成 Postman/HTTP 样例**
   ```bash
   npx --yes tsx ../tools/cli/src/commands/capabilities/quota.ts \
     --capability-id com.powerx.skeleton.template.create \
     --tenant sandbox \
     --base-url https://dev-api.powerx.local \
     --token $PX_DEV_API_TOKEN
   ls dist/capabilities/com.powerx.skeleton.template.create/samples/
   ```
7. **Go handler / service 自测**
   ```bash
   cd backend
   GOFLAGS=-mod=mod GOWORK=off go test ./...
   cd ..
   ```
8. **需要安装验证时执行 dist + local-install**
   ```bash
   VERSION=0.2.0 make dist
   make local-install
   ```
9. **生成能力 Diff + 灰度计划**
   ```bash
   npx --yes tsx ../tools/cli/src/commands/capabilities/diff.ts \
     --from HEAD~1:plugin.yaml \
     --to plugin.yaml \
     --output release/capabilities-change-report.md
   ```

## Init 输出工程：新建插件调试能力

当你通过 `px-plugin init` 生成全新插件仓库时，可按照下列顺序在新目录中验证能力。假设插件位于 `~/workspace/com.example.capability`。

1. **初始化并进入插件目录**
   ```bash
   PX_PLUGIN_BIN=${PX_PLUGIN_BIN:-$(pwd)/bin/px-plugin}
   [ -x "$PX_PLUGIN_BIN" ] || go build -o "$PX_PLUGIN_BIN" ./tools/cli/cmd/px-plugin
   "$PX_PLUGIN_BIN" init com.example.capability --backend go-gin --admin nuxt --force
   cd com.example.capability
   npm install
   go mod tidy ./backend
   ```
2. **使用 handler 扫描（可选，若已有注释）**
   ```bash
   node ../scripts/capabilities/discover-handlers.mjs --plugin .
   ```
3. **声明能力 / 生成契约**
   ```bash
   npx --yes tsx ../tools/cli/src/commands/capabilities/init.ts \
     --manifest ./plugin.yaml \
     --capability-id com.example.capability.createItem \
     --method POST \
     --description "Init 示例能力"
   ```
4. **Lint 与校验**
   ```bash
   npx --yes tsx ../tools/cli/src/commands/capabilities/lint.ts --manifest ./plugin.yaml
   CAP_MANIFEST=./plugin.yaml npm test
   ```
5. **提交能力 + 暴露配置**
   ```bash
   npx --yes tsx ../tools/cli/src/commands/capabilities/submit.ts \
     --manifest ./plugin.yaml \
     --base-url https://dev-api.powerx.local \
     --token $PX_DEV_API_TOKEN \
     --root-dir .
   ```
6. **设置租户 quota 并获取样例**
   ```bash
   npx --yes tsx ../tools/cli/src/commands/capabilities/quota.ts \
     --capability-id com.example.capability.createItem \
     --tenant demo \
     --base-url https://dev-api.powerx.local \
     --token $PX_DEV_API_TOKEN
   ```
7. **运行后端/handler 单测**
   ```bash
   cd backend
   GOFLAGS=-mod=mod GOWORK=off go test ./...
   cd ..
   ```
8. **构建 artefact + 本地安装（如需）**
   ```bash
   VERSION=0.1.0 make dist
   make local-install
   ```
9. **输出 Diff 报告 / 灰度计划**
   ```bash
   npx --yes tsx ../tools/cli/src/commands/capabilities/diff.ts \
     --from HEAD:plugin.yaml \
     --to ./plugin.yaml \
     --output release/capabilities-change-report.md
   ```

## 版本差异与灰度计划

使用 diff 命令即可比对任意两个 manifest/contract 版本（既可指向本地文件，也可用 Git 引用）：

```bash
# 与上一提交相比
px-plugin capabilities diff --to ./plugin.yaml

# 或显式指定两个文件
px-plugin capabilities diff \
  --from ./snapshots/manifest.prev.yaml \
  --to ./snapshots/manifest.next.yaml

# 支持 Git 语法：<ref>:<path>
px-plugin capabilities diff \
  --from HEAD~1:plugin.yaml \
  --to HEAD:plugin.yaml
```

- 默认输出写入 `release/capabilities-change-report.md`，内容涵盖：
  - 每个 capability 的版本/descriptor/RBAC/Schema/Exposure/Agent Tool 差异；
  - Schema 字段新增/删除列表与哈希指纹；
  - 灰度 & 通知计划 YAML 模板，可直接交付审核/运维团队；
- 可通过 `--output <path>` 自定义报告路径（例如 `dist/capabilities/report.md`）。
- 若当前目录在 Git 仓库中且未指定 `--from`，命令会自动与 `HEAD~1:plugin.yaml` 比较；纯离线目录请显式提供 `--from`。

---

## 常见问题

| 问题 | 说明 | 解决方案 |
|------|------|----------|
| `px-plugin dist` 报告 capability 阻断 | `.px-plugin/capabilities.json` 中存在 pending/rejected | 重新提交并等待审批通过 |
| `capabilities lint` 找不到 descriptor/schema | 运行 `scripts/capabilities/validate-capabilities.mjs` 自动生成 |
| Dev API 403 | 检查 `PX_DEV_API_TOKEN`，确保角色具备 capability registry 权限 |
| 多人协作状态丢失 | 将 `.px-plugin/` 纳入仓库（或文档化共享流程），避免不同成员重复提交 |

---

## 相关链接

- [Quickstart · Capability 流程]()（待补）
- [docs/development/t099-capability-scaffold.md](../development/t099-capability-scaffold.md)
- [docs/development/t100-capability-contracts.md](../development/t100-capability-contracts.md)
- [docs/development/t103-capability-audit.md](../development/t103-capability-audit.md)
