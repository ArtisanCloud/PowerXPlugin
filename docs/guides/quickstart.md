# PowerXPlugin Quickstart

欢迎来到 PowerXPlugin 指南中心！这里汇总仓库当前最重要的入门路径，帮助你在最短时间内启动 Standalone 环境、运行 CLI 并理解各模块之间的协作关系。仓库目前聚焦 **Go + Nuxt** 技术栈，其他语言仍在规划阶段。

## 入门路线图

1. **准备运行环境**  
   - 安装 Go 1.24+（启用 `GOWORK=on`）与 Node.js 18+ / npm 9+。  
   - 建议阅读 `docs/init-project.md` 了解更完整的环境说明。

2. **Standalone 骨架演练**  
   - 按照《[PowerXPlugin Standalone 启动教程](./develop/standalone-mode.md)》同步依赖并启动 Skeleton 后端与管理端。  
   - 使用多租户 Header 验证 Templates CRUD 示例（默认租户为 `X-Tenant-ID: 1`）：  
     ```bash
     # 列表/创建/更新/删除示例
     curl -s -H 'X-Tenant-ID: 1' http://localhost:8080/api/v1/templates | jq
     curl -s -X POST -H 'X-Tenant-ID: 1' -H 'Content-Type: application/json' \
       -d '{"name":"Demo","description":"From Quickstart","content":"Hello"}' \
       http://localhost:8080/api/v1/templates | jq
     ```  
     创建后刷新 `http://localhost:3000/_p/com.powerx.sample/admin/templates/crud`，确认前端可读取并编辑记录。  
   - 使用 `curl -w 'time_total: %{time_total}\n'` 记录 CRUD API 延迟（目标 <1s），必要时在 docs/research 中登记结果。  
   - 验证 `GET /api/v1/ping` 与 Starter 页面，熟悉框架 App 生命周期。

3. **CLI 模板生成与自测**  
   - 参考《[使用 CLI 生成并运行插件骨架](./develop/cli-plugin-tutorial.md)》构建 `px-plugin`、执行 `./bin/px-plugin init <plugin-id>` 生成骨架。  
   - 在新项目中运行 `go test ./...`、`npm run lint` 并复用上述 CRUD/延迟脚本，确认 CLI 输出与 Skeleton 行为一致。  
   - 检查 `plugin.yaml` 与契约文件，确认 CLI 模板与仓库保持一致。

4. **Dev API 热更新与 Doctor 诊断**  
   - 启动本地 Dev API（`make devapi`），并在示例插件目录执行：  
     ```bash
     go build -o ./bin/px-plugin ./tools/cli/cmd/px-plugin
     ./bin/px-plugin dev --watch \
       --entry examples/starter/go-admin \
       --tenant demo \
       --dev-api http://127.0.0.1:8077
     ```  
     观察 `Initial build complete. Watching for changes...`，随后修改任意源码文件，确保终端输出 `Reload applied` 且耗时 ≤2s。  
   - 另开终端执行 `./bin/px-plugin dev --logs <session-id>`，校验 SSE 日志能实时显示 `buildSucceeded/reloadApplied`。  
   - 最后运行 `./bin/px-plugin doctor --entry examples/starter/go-admin`，确认 `.doctor/report.json` 中 Toolchain / mTLS / Dev API / Watcher 状态均为 `pass`，便于新成员快速验证环境。

完成以上三个步骤，即可获得开发 PowerX 插件所需的核心能力：框架运行、模板生成与快速验证。

## 进一步阅读

- `docs/release.md`：了解版本发布与实验性 `package/dist/publish` 命令的最新状态。
- `docs/test/testing_usage.md`：掌握 `make test-smoke` / `make test-regression` 的测试流程。
- `docs/guide/bootstrap-context.md`：理解 `bootstrap.Context` 抽象与跨框架适配方式。
- `docs/backlog/multi-language.md`：关注多语言支持与未来路线图。

> 遇到问题时，可优先检查 Go/Node 版本，确认依赖安装无误，再回到两个教程逐步排查。欢迎在 README 中列出的渠道反馈问题与改进建议。
