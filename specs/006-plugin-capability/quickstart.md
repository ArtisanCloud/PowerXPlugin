# Quickstart — 插件能力注册与暴露治理闭环

1. **准备工作**
   - 安装 Go 1.24、Node 20、npm 9、Buf CLI、OpenAPI Generator。
   - 运行 `make deps && npm install`，确保 `px-plugin` CLI 可用。

2. **声明能力**
   - 执行 `px-plugin capabilities new --id com.powerx.demo.template.create`，在 `capabilities/` 生成能力文件与 Schema stub。
   - 完成 Handler (`backend/internal/handlers/capabilities/...`) 与 Service 逻辑，更新 `contracts/schema/input|output`。

3. **生成协议资产**
   - 运行 `npm run capabilities:lint`（或 `make capabilities-lint`）校验 imports/Schema。
   - 执行 `npm run capabilities:export` 生成 OpenAPI、Proto、Workflow Step、MCP manifest、Agent SSE、SDK bundle（产物位于 `contracts/exposure/**` 及 `dist/agent-sdk/`）。

4. **同步目录到 PowerX**
   - 安装/升级插件前执行 `px-plugin capabilities submit --env dev`，验证 Capabilities Manager 可以列出全部能力。
   - 通过 `make package-release` 生成 `.pxp` 包；部署到宿主时，Cap Manager 自动调用 PowerX `/admin/capability_registry`、Workflow Builder、Agent Hub 接口，同步能力目录。

5. **验证**
   - 在 PowerX Workflow Builder 中确认新的插件节点可拖拽。
   - 在 Agent Hub 中确认 MCP 工具已注册且能触发原子能力/复合 Workflow。
   - 触发一次暴露配置（REST + gRPC）并检查 3 分钟内是否同步至 API Gateway。

6. **监控与回滚**
   - 关注 `capability.catalog.sync_status` 指标；如失败会强制回滚。
   - 对 async 能力设置 SSE/回调通道并配置超时策略；同步能力默认阻塞直到完成。
