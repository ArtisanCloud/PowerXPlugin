# PowerXPlugin Quickstart

欢迎来到 PowerXPlugin 指南中心！这里汇总仓库当前最重要的入门路径，帮助你在最短时间内启动 Standalone 环境、运行 CLI 并理解各模块之间的协作关系。仓库目前聚焦 **Go + Nuxt** 技术栈，其他语言仍在规划阶段。

## 入门路线图

1. **准备运行环境**  
   - 安装 Go 1.24+（启用 `GOWORK=on`）与 Node.js 18+ / npm 9+。  
   - 建议阅读 `docs/init-project.md` 了解更完整的环境说明。

2. **Standalone 骨架演练**  
   - 按照《[PowerXPlugin Standalone 启动教程](./develop/standalone-mode.md)》同步依赖并启动 Skeleton 后端与管理端。  
   - 使用多租户 Header 验证 Templates CRUD 示例（默认租户为 `X-Tenant-UUID: 1`）：  
     ```bash
     # 列表/创建/更新/删除示例
     curl -s -H 'X-Tenant-UUID: 1' http://localhost:8080/api/v1/templates | jq
     curl -s -X POST -H 'X-Tenant-UUID: 1' -H 'Content-Type: application/json' \
       -d '{"name":"Demo","description":"From Quickstart","content":"Hello"}' \
       http://localhost:8080/api/v1/templates | jq
     ```  
     创建后刷新 `http://localhost:3000/_p/com.powerx.sample/admin/templates/crud`，确认前端可读取并编辑记录。  
   - 使用 `curl -w 'time_total: %{time_total}\n'` 记录 CRUD API 延迟（目标 <1s），必要时在 docs/research 中登记结果。  
   - 验证 `GET /api/v1/ping` 与 Starter 页面，熟悉框架 App 生命周期。

3. **CLI 模板生成与自测**  
   - 参考《[使用 CLI 生成并运行插件骨架](./develop/cli-plugin-tutorial.md)》构建 `px-plugin`、执行 `./bin/px-plugin init <plugin-id>` 生成骨架。  
   - 在新项目中运行 `go test ./...`、`npm run lint` 并复用上述 CRUD/延迟脚本，确认 CLI 输出与 Skeleton 行为一致。  
   - 检查 `plugin.yaml` 与契约文件，确认 CLI 模板与仓库保持一致。**注意**：本仓库的 manifest 真源在 `skeleton/plugin.yaml`，根目录的 `plugin.yaml` 为 symlink，运行相关命令时建议在 `skeleton/` 内执行或显式传入 `--manifest ./skeleton/plugin.yaml`。  
  - 如需调用 PowerX 通用能力，请在 `skeleton/plugin.yaml` 的 `capabilities.required`（即 manifest 的 `requiredCapabilities` 区块）填入授权 ID，并运行：  
     ```bash
     px-plugin capabilities plan --manifest ./skeleton/plugin.yaml
     px-plugin capabilities apply --manifest ./skeleton/plugin.yaml  # 将核准的能力写回 Registry
     ```  
     其中：
     1. `plan` 会比对 `capabilities.required` 与 Registry，若缺少授权会列出差异，阻止继续构建；
     2. `apply`/`lint|submit` 用于将审批通过的能力同步回 manifest，常在 CI/CD 中执行；
     3. 未传入 `--manifest` 时脚本现已自动回退到 `./skeleton/plugin.yaml`，建议总是显式指定以避免误操作。
     详细流程与示例请参考《[009-插件侧调用 PowerX 通用开放能力方案](../plan/009-consume-powerx-capability.md)》以及 `specs/009-consume-powerx-capability/quickstart.md`。
   - 运行 `npm run test`（默认使用 `./skeleton/plugin.yaml`，若不存在会自动回退到根目录清单），以及 `make capabilities-export`，确保能力目录与多协议资产均已生成；导出的 `contracts/exposure/*` 与 `dist/agent-sdk/manifest.json` 将用于 Workflow / Agent 注册。
   - 若需暴露能力，请同步阅读《[能力注册与暴露指南](./publish/capabilities.md)》，并在提交发布之前运行 `px-plugin capabilities init/lint/submit`，避免发布阶段被能力审核阻断。

4. **Dev API 热更新与 Doctor 诊断**  
   - 启动本地 Dev API（`make devapi`），并在示例插件目录执行：  
     ```bash
     go build -o ./bin/px-plugin ./tools/cli/cmd/px-plugin
     ./bin/px-plugin dev --watch \
       --entry examples/starter/go-admin \
       --tenant demo \
       --dev-api http://127.0.0.1:8077/api/v1
     ```  
     观察 `Initial build complete. Watching for changes...`，随后修改任意源码文件，确保终端输出 `Reload applied` 且耗时 ≤2s。  
   - 另开终端执行 `./bin/px-plugin dev --logs <session-id>`，校验 SSE 日志能实时显示 `buildSucceeded/reloadApplied`。  
   - 最后运行 `./bin/px-plugin doctor --entry examples/starter/go-admin`，确认 `.doctor/report.json` 中 Toolchain / mTLS / Dev API / Watcher 状态均为 `pass`，便于新成员快速验证环境。

完成以上三个步骤，即可获得开发 PowerX 插件所需的核心能力：框架运行、模板生成与快速验证。

- **观测提示**：安装自测后，PowerX 会自动暴露 `capability.catalog.sync_status` 与 `capability.workflow.async_duration` 指标（Prometheus 名称分别为 `powerx_capability_catalog_sync_status` 与 `powerx_capability_workflow_async_duration_seconds`）。通过宿主监控面板即可确认目录同步与复合 Workflow 耗时，排查安装或审核瓶颈。

## 进一步阅读

- `docs/release.md`：了解版本发布与实验性 `package/dist/publish` 命令的最新状态。
- `docs/test/testing_usage.md`：掌握 `make test-smoke` / `make test-regression` 的测试流程。
- `docs/guide/bootstrap-context.md`：理解 `bootstrap.Context` 抽象与跨框架适配方式。
- `docs/guides/publish/local-install.md`：学习如何使用模板内置 `Makefile`、`make local-install`/`make pack` 快速在 PowerX 安装插件。
- `docs/backlog/multi-language.md`：关注多语言支持与未来路线图。

> 遇到问题时，可优先检查 Go/Node 版本，确认依赖安装无误，再回到两个教程逐步排查。欢迎在 README 中列出的渠道反馈问题与改进建议。
