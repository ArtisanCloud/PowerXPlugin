# 插件能力注册与本地调试手册

> 适用范围：`/capabilities/register` 页面以及插件在 **独立运行/未接入 PowerX 宿主** 时的能力联调。目标是在同一页面完成「能力建模 + 直接调用插件后端接口」，而非走 PowerX Gateway。

## 1. 场景 & 目标

| 场景 | 说明 |
| --- | --- |
| 插件独立运行 | 通过 `make dev` 启动 `skeleton/web-admin` 与 `skeleton/backend`，仅依赖插件本身的接口（端口 `8078`/`8080` 等）。 |
| 能力建模 | 在表单里填写 `capability_id`、协议矩阵、示例与描述，生成 `capabilities/catalog.json` 条目。 |
| 即时调试 | 在右侧调试面板中读取当前协议，直接向插件后端暴露的 REST/gRPC/Workflow endpoint 发送请求，验证业务实现。 |

策略：参照 `/powerx/capability-lab` 的交互，但请求永远落在插件自己的 HTTP/gRPC 入口，不依赖宿主权限、租户授权或 PowerX Gateway。

## 2. 启动与准备

1. 在仓库根目录执行：
   ```bash
   make dev # 或分别启动 backend + web-admin
   node scripts/capabilities/validate-capabilities.mjs --manifest ./skeleton/plugin.yaml
   ```
2. 打开 `http://127.0.0.1:3131/capabilities/register`，使用 Root/Admin 账号登录。
3. 插件后端默认开放以下端点，可按需替换：
   - REST：`http://127.0.0.1:8078/api/v1/<resource>`
   - gRPC：`127.0.0.1:9090`（`powerx.*` / `com.powerx.plugins.*` service）
   - Workflow / Agent Tool：`contracts/exposure/workflow/*.json`、`agent-streams/*.yaml`

> **提示**：即使未在“能力曝光/租户授权”中配置，调试面板也能直连插件接口；无需经过 Registry 或宿主授权。

## 3. 表单与调试面板

1. **基础区块**：namespace/resource/action、场景、输入输出 Schema、示例等（提交后写入 catalog）。
2. **协议矩阵**：
   - REST：method + path（示例 `/api/v1/media/assets`）
   - gRPC：`service/method`（示例 `powerx.media.v1.MediaAssetAdminService/ListMediaAssets`）
   - Workflow/Agent：模板文件或 channel
3. **调试面板**（右侧）：
   - `API Base`：默认 `http://127.0.0.1:8078`，可改为其它环境。
   - `Protocol`：根据当前表单自动推导 REST/gRPC/Workflow。
   - `Action`：按协议生成，如 REST 的 `List/Create/Delete`、gRPC 的 `ListMediaAssets`。
   - `Payload`：JSON 编辑器，加载协议模板，可手动调整 `headers/query/body`/`rpc` 参数。
   - Request Preview：展示真实的 HTTP/gRPC 调用配置，方便复制到 CLI。
   - Result：显示 status、traceId、warnings、raw response、历史记录。

## 4. 本地调试步骤

1. **填写协议** → 点击“调试草稿”。Modal 打开后自动补全 `capabilityId`、Action、Payload。
2. **设置 Base URL**：
   - REST：`http://127.0.0.1:8078` + 表单中填写的 path。
   - gRPC：只需填 `grpc://127.0.0.1:9090`（UI 里填写 service/method 即可）。
3. **编辑 Payload**：
   - REST 模式：`{ "method": "GET", "endpoint": "/api/v1/media/assets", "query": {}, "body": {} }`
   - gRPC 模式：`{ "endpoint": "powerx.media.v1.MediaAssetAdminService", "rpc": "ListMediaAssets", "metadata": {}, "body": {} }`
   - Workflow：`{ "workflow": { "template": "contracts/exposure/workflow/template-quality-distribute.json" }, "payload": {} }`
4. **Invoke**：点击“开始调试”，请求会直接发送到插件后端：
   - REST：通过 `$fetch`/Axios 调用本地 API，不依赖 `/api/v1/integration/capabilities/invoke`。
   - gRPC：使用前端 gRPC 客户端（Web/gRPC proxy 或 dev 工具）直连插件 gRPC 服务。
   - Workflow：本地伪执行（调用插件 Workflow handler 或模拟 Workflow Engine 的 HTTP 入口）。
5. **查看结果**：成功/失败都会写入历史列表，可复制 Trace ID、Raw Response 进行排查。

### 4.1 MCP 会话调试（BETA）

`/capabilities/register` 右上角新增“**MCP 会话调试**”入口，无需离开插件后台即可按文档完成 REGISTER → ACK → Heartbeat → Invoke → SSE 全流程本地联调（更详尽的协议说明仍参考《mcp-guide.md》）。

> 入口位置：页面顶部“刷新 / 注册能力”按钮旁的 `MCP 会话调试`。

功能要点：

1. **会话控制**：在表单里填写 `runtime_assignment_id`、`jwt_id`、`capabilities_hash`，点击“注册会话”即可调用 `POST /api/v1/admin/runtime/sessions/register`。同一面板还提供 ACK、Heartbeat、Close 操作，自动携带当前 Session ID。
2. **Invoke 调试**：第二个卡片用于构造 Integration Envelope（Trace/Correlation/IssuedAt/ToolScope/intent/payload_ref），“执行 Invoke” 会命中 `POST /api/v1/admin/runtime/sessions/{id}/invoke`，并展示响应体、错误详情与最近 5 条历史记录。文本框接受 JSON，会在发送前自动 `JSON.stringify`，无需手动转义。
3. **SSE 订阅**：第三个卡片封装了 `/mcp/sse?session_id=...`，注册/ACK/Invoke/Close 都会实时推送到列表，便于观察 MCP 生命周期事件。可随时断开或清空记录。
4. **与草稿联动**：点击“引用当前能力”即可把当前草稿/表单里的 Capability ID 同步到 MCP Invoke 区域，省去重复输入；Tenant/Session/Intent 会自动写入 Metadata。

本地流程示例：

```text
1. 打开 MCP 会话调试 → “生成 Runtime Assignment ID” → “注册会话”；
2. 查看下方 Session 信息与 SSE 日志，确认 `session.registered` 事件已推送；
3. 输入 Tool Scope/Intent/Payload，点击“执行 Invoke”；如需多场景复用，可重复修改 Intent；
4. SSE 卡片会看到 `invoke.completed`，Invoke 历史也会保存 Trace/返回值；
5. 完成后点击“关闭会话”，事件会显示 `session.closed`。
```

这样即可在同一个页面完成 REST/gRPC/Workflow/MCP 的回归验证，减少手动敲 curl 的成本；当需要了解更底层的 Envelope/事件结构，可继续查阅《docs/guides/publish/capabilities/mcp-guide.md》。

## 5. 与 `/powerx/capability-lab` 的关系

| 项目 | `/powerx/capability-lab` | `/capabilities/register` 调试面板 |
| --- | --- | --- |
| 目标 | 调试 **PowerX Core** 已发布能力 | 调试 **插件自身** 尚未发布的能力 |
| 数据源 | PowerX 能力目录（source=corex） | `capabilities/catalog.json` + 当前表单 |
| 请求目标 | PowerX Gateway (`8077/tenant/invocations`) | 插件后端 REST/gRPC/Workflow 接口（本地端口） |
| 授权 | 需要宿主租户权限 | 不需要宿主，仅需插件 dev token |
| 适用阶段 | 插件上线前验证宿主能力 | 插件开发/联调阶段验证自身能力 |

可在本地先调通插件 API，再将同样的协议/示例同步给宿主；当插件安装到 PowerX 后，再通过 Capability Lab or Gateway 验证宿主链路。

## 6. 常见问题

1. **为什么不用 `/api/v1/integration/capabilities/invoke`？**  
   因为插件独立运行时并没有宿主的授权/Registry；调试面板直接命中插件自己的 HTTP/gRPC 端口即可。

2. **出现 4xx/5xx？**  
   - 检查 `API Base`、path 或 gRPC service 是否与 `skeleton/backend` 中的路由一致。  
   - 若接口需要额外 Header/Token，可在 Payload 或 Header 区域补充。

3. **Workflow/Agent 如何调试？**  
   - Workflow：调试面板会把模板与 payload 发送到 `Workflow Stub`（位于插件后端 `/api/v1/workflow/trigger` 等模拟入口）。  
   - Agent Tool：可在 Payload 里直接填充 `tool_id` + `input`，调用插件提供的 Agent handler。

4. **需要落地到宿主时怎么办？**  
   先在本地调试完成，再触发 `scripts/capabilities run catalog` → 提交 PR；部署至宿主后，再用 `/powerx/capability-lab` 或 `tenant/invocations` 验证宿主链路。

## 7. TODO

1. 支持在调试面板中切换“插件本地”与“宿主代理”两种模式，方便在同一 UI 比对结果。
2. 提供一键导出 `.http` / `curl` / `ghz`（gRPC）示例，方便 CLI 回归。
3. 对 Workflow/Agent 提供更完整的伪执行器（含多节点 DAG 调试、SSE 模拟）。

---

通过本手册，研发可以在 `/capabilities/register` 页面完成「能力建模 → 本地接口调试」的闭环，确保在提交 PR 或与宿主对接前就已经验证插件 API 的真实性能。若需要调试 PowerX 底座能力，再转到 `docs/guides/develop/consume-powerx-capability/README.md` 与 `/powerx/capability-lab`。 
