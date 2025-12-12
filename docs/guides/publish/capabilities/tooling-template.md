# 开发者操作：Template 示例 & 批量工具

聚焦日常开发：如何在 skeleton 内批量生成能力、如何在新插件中快速落地，以及如何利用扫描脚本减轻重复劳动。始终以 `template` 模型为案例。

## Skeleton 模式：仓库内置样板

> 前提：`skeleton/plugin.yaml` 为唯一的开发态 manifest（根目录仅保留指向该文件的 symlink）。以下命令均在 `repo-root/skeleton` 中执行。

1. **扫描 handler → 输出能力清单**
   ```bash
   node ../scripts/capabilities/discover-handlers.mjs \
     --plugin . \
     --handlers backend/internal/transport/http/admin/templates \
     --manifest ./plugin.yaml \
     --output tmp/template-capabilities.json
   ```
   - 解析 `// capability:` 注解并写入 `tmp/template-capabilities.json`；
   - 如果在其他目录执行，可通过 `--plugin <dir> --manifest <path>` 调整根路径。

2. **初始化或更新能力契约**
   ```bash
   npx --yes tsx ../tools/cli/src/commands/capabilities/init.ts \
     --manifest ./plugin.yaml \
     --batch tmp/template-capabilities.json
   ```
   - `tmp/template-capabilities.json` 即上一步的扫描结果，可手动编辑以覆盖多模型；
   - 若只想生成单个能力，可改用 `--capability-id <id> --method <VERB>`。

3. **Lint / Schema 校验**
   ```bash
   npx --yes tsx ../tools/cli/src/commands/capabilities/lint.ts --manifest ./plugin.yaml
   CAP_MANIFEST=./plugin.yaml npm test
   ```
   - `lint`：检查 catalog + contracts；
  - `npm test`：验证 `scripts/capabilities/run-from-package.mjs`，默认读取 `./skeleton/plugin.yaml`。

4. **提交 + 暴露**
   ```bash
   npx --yes tsx ../tools/cli/src/commands/capabilities/submit.ts --manifest ./plugin.yaml --base-url $PX_DEV_API_BASE --token $PX_DEV_API_TOKEN
   npx --yes tsx ../tools/cli/src/commands/capabilities/quota.ts --capability-id com.powerx.skeleton.template.create --tenant sandbox --base-url $PX_DEV_API_BASE --token $PX_DEV_API_TOKEN
   ```
   - `submit`：向能力中心注册；
   - `quota`：为租户分配额度，生成 Postman/HTTP 示例便于测试。

5. **Diff & 灰度计划（可选）**
   ```bash
   npx --yes tsx ../tools/cli/src/commands/capabilities/diff.ts \
     --from HEAD~1:plugin.yaml \
     --to plugin.yaml \
     --output release/capabilities-change-report.md
   ```
   结合 `docs/guides/publish/capabilities/registration.md` 的生命周期章节，即可产出灰度计划。

## 新插件仓库（px-plugin init 输出）

1. `px-plugin init com.example.capability --backend go-gin --admin nuxt --force`
2. `cd com.example.capability && npm install && go mod tidy ./backend`
3. `node ../scripts/capabilities/discover-handlers.mjs --plugin .`（若已有注解）
4. 按步骤运行 `init → lint → export → submit → quota`，流程与 skeleton 相同。

## 批量生成 / 自定义脚本

- **批量描述文件**：`scripts/capabilities/template-crud.json`（自建）可列出多个能力定义，`init.ts --batch` 会依次生成。
- **Go AST 扫描**：若不想手动标注，可在自定义脚本中用 `go/packages` 读取 Gin 路由，匹配 REST 动词 + handler 名，再拼装 `--capability-id`。生成 `capabilities/catalog.json` 后可复用 `validate`/`export`。
- **自定义模板**：`scaffold/templates/backend/go-gin/...` 中已包含 CapRoute/handler stub 模板，可根据插件需求扩展；运行 `tools/cli` 子命令即可更新脚手架默认产物。

## 常用命令速查

| 场景 | 命令 |
|------|------|
| 扫描 handler | `node scripts/capabilities/discover-handlers.mjs --plugin . --handlers <path>` |
| 初始化/覆盖能力 | `px-plugin capabilities init <id> --method POST ...` |
| 批量初始化 | `npx --yes tsx ../tools/cli/src/commands/capabilities/init.ts --batch ./config.json` |
| 校验契约 | `node scripts/capabilities/validate-capabilities.mjs --manifest ./plugin.yaml` |
| 导出多协议 | `npm --prefix scripts/capabilities run export` |
| 提交能力 | `px-plugin capabilities submit --manifest ./plugin.yaml --base-url ...` |
| 配额授权 | `px-plugin capabilities quota --capability-id ... --tenant ...` |
| 生成 Diff | `px-plugin capabilities diff --from <ref>:plugin.yaml --to plugin.yaml` |

依照以上流程，开发者可以单个或批量地将模板能力（或任何业务模型）映射为 PowerX 可调用的能力集。EOF
