v# 009-插件侧调用 PowerX 通用开放能力方案

## 背景与目标
- 参考《PowerX Capability Exposure Plan》，底座通过 Integration Gateway + MCP 汇聚 Media、事件总线、Scheduler、Workflow、Knowledge 等能力，并统一输出 HTTP/OpenAPI 与 gRPC 契约。
- 插件（无论宿主模式还是 Skeleton 本地独立调试）需要用一致的方式领取能力、鉴权并发起调用，避免再依赖 Admin API 的内部路由。
- 本文给出插件侧可执行的配合方案，涵盖能力发现、Manifest 申领、调用编排、观测治理与落地任务，帮助快速复用 PowerX 已开放的通用系统能力。

## 统一接入原则
1. **能力发现与注册**
   - 通过 Capability Registry `/tenant/capabilities` 获取 `source=corex` 条目（Media/Event/Scheduler/Workflow/Knowledge 等）。
   - 插件在 `skeleton/plugin.yaml` 与宿主部署清单中声明依赖的 Capability ID（如 `com.corex.media.assets.manage`），CI 使用 `px-plugin capabilities plan|apply --manifest ./skeleton/plugin.yaml` 做静态校验。
2. **鉴权与凭证**
   - 统一利用 Tool Grant + STS。宿主模式由 Admin 控台自动注入 `PX_PLUGIN_TOOL_TOKEN`，Skeleton 模式通过 `px-plugin login` 获取 `~/.powerx/credentials` 并在本地代理 `PX_TOOL_TOKEN`。
   - 调用 Integration Gateway HTTP 接口时：`Authorization: Bearer <tool-grant-token>` + `X-Tenant-UUID`; gRPC 场景通过 Gateway 颁发的 mTLS 证书或互斥 Token。
3. **调用入口**
   - REST：`POST/GET {GatewayOrigin}/tenant/invocations`，Body 携带 `capabilityId`、`action`、`payload`，由 Gateway 路由至底座服务。
   - gRPC：`IntegrationGatewayTenantService.InvokeCapability`，或直接指向模块契约（如 `powerx.media.v1.MediaAssetAdminService`），前提是 Registry 中已对插件授权。
4. **观测与限流**
   - 复用 FR-001~FR-015 追踪要求：插件需在调用端注入 `X-Request-ID`，并捕获 Gateway 返回的 `traceId`，写入插件日志。
   - 对于自研能力，需要在 Registry 中补充 `rateLimit` 与 `quota`，以免共用通道被抢占。

## 宿主模式落地路径
| 步骤 | 说明 |
| --- | --- |
| 1. 能力申领 | 在 Admin 界面或 `px-plugin capabilities apply` 中勾选 `source=corex` 的能力；Pipeline 校验 Manifest 中的 `requiredCapabilities`。 |
| 2. SDK 初始化 | 在 `packages/admin` / `packages/backend` 里通过 `@artisan-cloud/plugin-framework-client` 注入 Gateway Client，读取宿主注入的 `PX_PLUGIN_ENV`、`PX_PLUGIN_TOOL_TOKEN`。 |
| 3. 调用封装 | 后端推荐使用 Go `powerx/integration` SDK（`integration.NewClient().Invoke(ctx, capabilityId, payload)`），前端使用 `$fetch('/tenant/invocations')` 封装；保持能力 ID 常量化，便于限流配置。 |
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
1. **本地凭证**：执行 `px-plugin login --manifest ./skeleton/plugin.yaml`，生成 `~/.powerx/credentials`，并在 `skeleton/.env.local` 暴露 `PX_GATEWAY_BASE_URL`, `PX_TOOL_TOKEN`。
2. **Mock & 回退**：在 Gateway 不可达时，仍可以 `npm run dev -- --use-mock=media` 走内存实现；真正调用 Core 能力前，需在 `docs/use_cases` 标记 `powerxCapability: true`，方便 QA 构建测试矩阵。
3. **调用方式**：保持与宿主一致，Skeleton 后端（Go Gin）通过 `integration.Client`，前端 Nuxt 通过 `usePowerXCapability(action)` 自定义 Composable，对应 HTTP 请求发往 `PX_GATEWAY_BASE_URL`。
4. **调试工具**：使用 `scripts/capabilities/run-from-package.mjs --manifest ./skeleton/plugin.yaml --cap com.corex.media.assets.manage` 进行单条调用验证；配合 `make dev-proxy` 将本地请求透传到 Dev Gateway。

### Skeleton 示例（Composable）
```ts
export const usePowerXCapability = () => {
  const config = useRuntimeConfig()
  return async (capabilityId: string, action: string, payload: Record<string, any>) => {
    return await $fetch('/tenant/invocations', {
      baseURL: config.gatewayBaseUrl,
      method: 'POST',
      headers: {
        Authorization: `Bearer ${config.powerx.toolToken}`,
        'X-Tenant-UUID': config.powerx.tenantUuid,
      },
      body: { capabilityId, action, payload },
    })
  }
}
```

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
   - Go：在 `packages/backend` 提供 `pkg/powerx/gateway/client.go`，封装 `Invoke`、`PresignMedia` 等常用函数；
   - Nuxt：在 `packages/admin` 增加 `plugins/powerx-gateway.client.ts`，读取 `PX_*` 环境变量。
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
