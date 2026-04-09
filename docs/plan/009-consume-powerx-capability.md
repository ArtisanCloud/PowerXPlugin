# 009-插件侧调用 PowerX 通用开放能力方案

> 📘 此方案在仓库 README 的「PowerX 通用能力消费方案」章节也做了概述，并链接到本文件与 quickstart；若你从 README 跳转而来，可继续按下文执行；若在本页需要回到总览，可返回 README 该章节获取更多上下文。

## 背景与目标
> 本文搭配 `docs/quickstart.md`、`specs/009-consume-powerx-capability/quickstart.md` 食用更佳：若需要快速演练 Skeleton/宿主场景，请优先阅读 quickstart 中的配置与命令示例。
- 参考《PowerX Capability Exposure Plan》，底座通过 Integration Gateway + MCP 汇聚 Media、事件总线、Scheduler、Workflow、Knowledge 等能力，并统一输出 HTTP/OpenAPI 与 gRPC 契约。
- 插件（无论宿主模式还是 Skeleton 本地独立调试）需要用一致的方式领取能力、鉴权并发起调用，避免再依赖 Admin API 的内部路由。
- 本文给出插件侧可执行的配合方案，涵盖能力发现、Manifest 申领、调用编排、观测治理与落地任务，帮助快速复用 PowerX 已开放的通用系统能力。

## 统一接入原则
1. **能力发现与注册**
   - 通过 Capability Registry `/tenant/capabilities` 获取 `source=corex` 条目（Media/Event/Scheduler/Workflow/Knowledge 等）。
   - 插件在 `skeleton/plugin.yaml` 与宿主部署清单中声明依赖的 Capability ID（如 `com.corex.media.assets.manage`），CI 使用 `px-plugin capabilities plan|apply --manifest ./skeleton/plugin.yaml` 做静态校验。
2. **鉴权与凭证**
   - 统一利用 Tool Grant + STS，但由 framework 在运行时执行模式分流：
     - `delegated`：仅允许 Bearer（`PX_PLUGIN_TOOL_TOKEN`/平台注入 token）
     - `standalone local`：同样使用 Bearer（`PX_PLUGIN_TOOL_TOKEN`）
   - 运行策略冲突（如 `auth_scheme != bearer`）必须 fail-fast，并在启动日志输出诊断信息。
3. **调用入口**
   - REST：`POST {GatewayOrigin}/tenant/invocations`，Body 必须包含 `capabilityId/action/preferred_protocol` 与 **完整的协议描述**（`payload.method`、`payload.endpoint`、`payload.query/body/headers`），Gateway 才能根据 `capability_id + method + endpoint` 匹配对应 Adapter；缺少字段会直接在插件后端被拒绝。
   - gRPC：`IntegrationGatewayTenantService.InvokeCapability`，或直接指向模块契约（如 `powerx.media.v1.MediaAssetAdminService`），同样需在 payload 中指明 `preferred_protocol="grpc"` 以及服务/方法名称。
4. **观测与限流**
   - 复用 FR-001~FR-015 追踪要求：插件需在调用端注入 `X-Request-ID`，并捕获 Gateway 返回的 `traceId`，写入插件日志。
   - 对于自研能力，需要在 Registry 中补充 `rateLimit` 与 `quota`，以免共用通道被抢占。

## 宿主模式落地路径
| 步骤 | 说明 |
| --- | --- |
| 1. 能力申领 | 在 Admin 界面或 `px-plugin capabilities apply` 中勾选 `source=corex` 的能力；Pipeline 校验 Manifest 中的 `requiredCapabilities`。 |
| 2. SDK 初始化 | 在 `packages/admin` / `packages/backend` 里通过 `@artisan-cloud/plugin-framework-client` 注入 Gateway Client，并读取宿主注入的 `PX_PLUGIN_ENV`、`PX_PLUGIN_TOOL_TOKEN`。 |
| 3. 调用封装 | 后端统一使用 framework Host Capability Client（`integration.NewClient().Invoke(ctx, capabilityId, payload)`），并在 HTTP 入口统一接入 `RequireCapabilityGateway` Guard；保持能力 ID 常量化，便于限流配置。 |
| 4. 多环境切换 | `PX_GATEWAY_BASE_URL` 在宿主部署中由运维注入；所有调用通过该域名转发，避免直接访问内部微服务。 |
| 5. 观测回传 | 在插件日志、Metric（如 `plugin_capability_call_duration`）中使用 Gateway 返回的 `traceId`，并通过 `IntegrationGatewayHook` 上报审计事件。 |

### 宿主示例（Go 后端）
```go
payload := map[string]any{
    "assetName": "invoice.pdf",
    "contentType": "application/pdf",
}
resp, err := gatewayClient.Invoke(ctx, integration.InvokePayload{
    CapabilityID: "com.corex.media.assets.manage",
    Action:       "Create",
    Payload:      payload,
})
if err != nil { /* 记录 traceId & 错误 */ }
```

## Skeleton 本地开发模式
1. **本地凭证**：执行 `px-plugin login --manifest ./skeleton/plugin.yaml`，生成 `~/.powerx/credentials`，并在 `skeleton/.env.local` 暴露 `PX_GATEWAY_BASE_URL`, `PX_PLUGIN_TOOL_TOKEN`。
   - CLI 阶段截图（文本）：
     ```text
     $ px-plugin login --manifest ./skeleton/plugin.yaml --tenant demo-tenant
     ✔ Loading manifest skeleton/plugin.yaml
     ✔ Authenticate against PowerX Dev Gateway
     ✔ Tool token issued (tenant=demo-tenant-uuid, expires_in=24h)
     ℹ Credentials saved to ~/.powerx/credentials
     ```
   - `.env.local` 样例（供 Skeleton web-admin/backend 共用）：
     ```ini
     PX_GATEWAY_BASE_URL=https://gateway.powerx.dev/_tenant
     PX_PLUGIN_TOOL_TOKEN=sts-dev-xxxxxx
     PX_USE_MOCK=media # Dev Gateway 不可达时可选
     POWERX_PROXY=0
     ```
2. **Mock & 回退**：在 Gateway 不可达时，仍可以 `npm run dev -- --use-mock=media` 走内存实现；真正调用 Core 能力前，需在 `docs/use_cases` 标记 `powerxCapability: true`，方便 QA 构建测试矩阵。
3. **调用方式**：保持与宿主一致，Skeleton 后端（Go Gin）通过 `integration.Client` 调用 Gateway，并在 `/api/powerx/capabilities/*` 等路径暴露受控 API；前端 Nuxt 仅访问这些插件后端 API，不直接携带 `PX_*` 凭证。
   - 默认情况下，`@artisan-cloud/plugin-framework-admin` 会将 `/_p/<pluginId>/api/v1` 写入 `runtimeConfig.public.powerx.apiBase`；如需自定义，可在 `.env` 设置 `NUXT_PUBLIC_POWERX_API_BASE`、`NUXT_PUBLIC_POWERX_CAPABILITY_ENDPOINT`。
4. **调试工具**：使用 `scripts/capabilities/run-from-package.mjs --manifest ./skeleton/plugin.yaml --cap com.corex.media.assets.manage` 进行单条调用验证；配合 `make dev-proxy` 将本地请求透传到 Dev Gateway。

### Skeleton 示例（前端调用插件后端 API）
```ts
export const usePowerXCapabilityBridge = () => {
  return async (capabilityId: string, action: string, payload: Record<string, any>) => {
    return await $fetch('/api/powerx/capabilities/invoke', {
      method: 'POST',
      body: { capabilityId, action, payload }
    })
  }
}
```

### Skeleton 能力调试页面（Capability Lab）

为便于在本地验证 CoreX 能力与 Mock/真实链路，Skeleton web-admin 将新增 `pages/powerx/capability-lab.vue`，具备以下特性：

1. **配置面板**：提供 Capability ID 下拉（读取 manifest `requiredCapabilities`）、Action 输入框、JSON Payload 编辑器，并允许选择 `tenant_uuid`、切换 `PX_USE_MOCK`。选择 REST Action 时会提示必须填写 `preferred_protocol + method + endpoint`，并可一键插入模板（含 query/headers/body 字段）；gRPC Action 则提示 service/method。
2. **请求预览**：在发送前展示目标 URL（默认 `/api/v1/integration/capabilities/invoke`）、Headers（脱敏 Authorization）、请求体，方便复制。
3. **调用结果**：以卡片形式输出状态、TraceId（可复制）、响应 JSON、耗时；若服务器返回 `warnings`/`X-PowerX-Contract-Status`，则在页面顶部提示契约升级或 Mock 告警。
4. **历史记录**：保留最近若干次请求/响应，支持展开比对与复制为 `.http` 或 `curl`。
5. **权限控制**：仅 `IsRoot` 或具备系统管理员角色的开发者可见，通过 web-admin 菜单新增“开放能力调试”入口，并在文档中标注使用方式，避免普通用户误用 PowerX 能力。

页面底层复用 `powerx-capability.client.ts`，因此与正式业务调用共享同一封装，可真实触发 Gateway 链路；若启用 Mock，则直接透传 `PX_USE_MOCK` 供后端网关客户端切换。实现完成后将本计划、Spec、Tasks 与 `docs/guides/develop/consume-powerx-capability/README.md` 同步。

> **能力目录数据源**：Skeleton `/api/v1/admin/capabilities` 默认读取插件本地 `capabilities/catalog.json`；当请求携带 `source=corex` 时，会透过 Gateway `GET /tenant/capabilities?source=corex` 拉取 PowerX 底座的实时能力清单（含 REST/gRPC 通道），并转换为 catalog 格式返回。Capability Lab 前端始终使用 `?source=corex`，从而保证页面展示的 ID/协议与 PowerX 文档一致，而其他页面仍可使用本地 manifest 数据。

## 限流与配额配置流程
1. **Preparation**：运维先登录 Dev API，导出 `PX_DEV_API_BASEURL` 与 `PX_DEV_API_TOKEN`（或通过 Secret 管理）；在本地/CI 均可复用。  
2. **CLI 授权命令**：`px-plugin capabilities quota` 现已支持 `--manifest`，会读取 `capabilities.provides` 验证目标 ID，并默认将样例输出到 manifest 所在目录，避免手动传 `rootDir`。
   ```bash
   px-plugin capabilities quota \
     --manifest ./skeleton/plugin.yaml \
     --capability-id com.powerx.plugins.base.template.create \
     --tenant sandbox-tenant \
     --base-url "${PX_DEV_API_BASEURL}" \
     --token "${PX_DEV_API_TOKEN}" \
     --qps 50 --burst 100 --limits 5000
   ```
   - 若 manifest 仅声明单一能力，可省略 `--capability-id`，CLI 会自动推断，并在 manifest 缺少该能力时给出报错提示。
3. **产物与验证**：命令会调用 `POST /internal/plugins/capabilities/{id}/tenants/{tenant}/quota`，并在 `dist/capabilities/<id>/samples/` 下生成 Postman 与 `.http` 示例，便于 QA/运维在 Gateway/Dev API 再次回放；Telemetry 会记录 `capability.cli.quota_total`，可在 Grafana/ClickHouse 查询。
4. **运行时告警**：当 Skeleton 或宿主调用触发 `rateLimitExceeded` 时，可结合上一步配置的额度确认是否需要扩容；CLI 输出会提示 `quota` JSON 片段，可直接粘贴给租户管理员备案。

## 能力使用速查
| Capability ID | 典型插件场景 | 推荐调用动作 |
| --- | --- | --- |
| `com.corex.media.assets.read` | 媒资列表、详情 | `action=List`、`action=Get`，REST `GET {APIPrefix}/media/assets` |
| `com.corex.media.assets.manage` | 上传/预签名 | `action=Create`、`action=Presign`，REST `POST {APIPrefix}/media/assets/{uuid}/presign` |
| `com.corex.eventfabric.publish` | 统一事件广播 | gRPC `EventDeliveryService/PublishEvent` 或 REST `POST /tenant/invocations` with `action=Publish` |
| `com.corex.scheduler.jobs` | 定时/Workflow 触发 | gRPC `WorkflowService/StartInstance`；对 Skeleton 可透过 Gateway HTTP 触发 |
| `com.corex.workflow.builder` | 模板查询/发布 | gRPC `WorkflowService/ListDefinitions`、`PublishDefinition` |
| `com.corex.knowledge.space` | 知识库同步 | gRPC `KnowledgeSpaceAdminService/TriggerIngestion` |

## 任务与依赖
1. **Manifest 对齐**：更新 `skeleton/plugin.yaml` 模板，增加 `requiredCapabilities` 示例；`docs/guides/manifest.md` 同步说明。
2. **Gateway Client 模块化**：
   - Go：在 `packages/backend` 提供 `pkg/powerx/gateway/client.go` 及默认 HTTP handler，封装 `Invoke`、`PresignMedia` 等常用函数并通过插件后端 API 暴露；
   - 增加运行时策略层：`detectRuntimeGatewayMode + enforceGatewayCredentialPolicy`，并提供统一 `RequireCapabilityGateway`；
   - Nuxt：在 `packages/admin` 增加 `plugins/powerx-capability.client.ts`，仅依赖插件后端 API Base，禁止直接读取 `PX_*` 凭证。
3. **脚手架命令**：扩展 `scripts/capabilities/run-from-package.mjs`，支持 `--mode skeleton|host`，自动拼装凭证。
4. **观测对齐**：在 `docs/operations/observability.md` 增加 Gateway trace 采集示例；Go 端默认注入 `log.WithField("traceId", resp.TraceID)`。
5. **QA 验收**：补充 `tests/capabilities/media_invocation_test.go`（可使用 `PX_GATEWAY_BASE_URL` 的 stub server）与 Playwright 场景，确保 Admin UI 触发开放能力链路。

## 风险与缓解
- **鉴权失配**：采用统一 Token 管理脚本（`scripts/auth/refresh-tool-token.sh`），并在客户端启动时检测 Token 是否即将过期，提前刷新。
- **环境差异**：Skeleton 默认指向 Dev Gateway，宿主可通过 `PX_GATEWAY_BASE_URL` 切换；提供 `npm run env:doctor` 检查变量。
- **契约变更**：CI 挂载 `make contracts-test`，比对 `specs/<module>` 下的 OpenAPI/gRPC 契约，防止插件侧调用参数过期。

## 下一步
1. 与 PowerX Core 团队确认 Registry `source=corex` 能力列表与限流默认值。
2. 在插件示例场景（Media 上传、Scheduler 调度）中落地上述调用封装并写入 `docs/use_cases`。
3. 结合 007 Standalone IAM 方案，验证 Delegated 模式只读/代发调用链是否满足审核要求。
