# PowerX 通用能力调用观测指引

本指南覆盖插件在调用 PowerX Gateway 能力时需要关注的日志、指标与 CLI 诊断步骤，帮助研发/运维快速定位失败与限流问题。适用于宿主与 Skeleton 模式，路径引用均以当前仓库为准。

## 1. 关键指标

框架在 `framework/backend/go/observability/capability_metrics.go` 暴露以下 Prometheus 指标：

| 指标 | 说明 | 标签 |
| --- | --- | --- |
| `capability_invocation_duration_ms` | 能力调用耗时直方图 | `capability`（能力 ID）、`tenant`（租户 UUID）、`result`（`success/validation/unauthorized/rate_limited/upstream`） |
| `capability_invocation_total` | 调用总次数计数器 | 同上 |
| `capability_rate_limit_events_total` | 限流事件计数器 | `capability`、`tenant` |
| `capability_registry_duration_ms`/`capability_exposure_activate_rate` | 供能力目录/暴露流程复用的历史指标 | 参见原文件 |

### 采集示例

1. 在宿主或 Skeleton 后端启动 Prometheus Exporter（默认 `/metrics`）。
2. 按照能力维度筛选：
   ```promql
   rate(capability_invocation_total{capability="com.corex.media.assets.manage"}[5m])
   ```
3. 监控限流峰值：
   ```promql
   increase(capability_rate_limit_events_total[10m])
   ```

## 2. 日志事件

`framework/backend/go/internal/services/capabilityinvoker/service.go` 统一输出以下日志：

| 事件 | 触发条件 | 关键字段 |
| --- | --- | --- |
| `capability.invoke.success` | 调用成功 | `capabilityId`、`tenantUUID`、`action`、`traceId`、`durationMs` |
| `capability.invoke.rate_limit` | Gateway 返回 429 | 同上 + `code=RATE_LIMIT`、`statusCode=429` |
| `capability.invoke.validation_failed`/`unauthorized`/`failure` | 400/401/403/其他错误 | 同上，`message` 为底座返回的错误描述 |
| `audit.capability.invocation.denied` | 429 或 401/403 | 用于审计的统一事件，保留 `capabilityId`、`tenantUUID`、`traceId`、`statusCode`、`code`、`message` |

建议在日志聚合系统（如 Loki / Elasticsearch）中按 `traceId` 或 `capabilityId` 检索，配合指标定位问题。

## 3. CLI 诊断

`px-plugin doctor` 已在 `tools/cli/src/executors/doctor.ts` 中新增 Gateway 凭证检查：

1. 默认读取环境变量，若存在 `skeleton/.env.local` 会一并解析。
2. 校验 `PX_GATEWAY_BASE_URL`、`PX_TOOL_TOKEN`（或 `PX_PLUGIN_TOOL_TOKEN`）、`PX_TENANT_UUID` 是否配置。
3. 对 Token 进行 JWT 解析获取过期时间：
   - 已过期：`status=fail`，提示重新执行 `px-plugin login --manifest ./skeleton/plugin.yaml`。
   - 24 小时内过期：`status=warn`，提醒提前刷新。
4. 若未检测到 `.env.local` 亦会提示 Warn，方便 Skeleton 团队补齐样本。

运行示例：

```bash
npm run cli -- doctor --project-dir skeleton \
  && cat skeleton/.doctor/report.json | jq '.checks[] | select(.id=="gateway-credentials")'
```

## 4. 常见问题排查

| 现象 | 建议步骤 |
| --- | --- |
| 调用返回 429 且 `capability_rate_limit_events_total` 激增 | 观察 `capability.invoke.rate_limit` 日志获取 `traceId`、`tenantUUID`，检查是否存在突发流量；必要时调整限流或启用 Mock。 |
| CLI doctor 提示 Token 过期 | 执行 `px-plugin login --manifest ./skeleton/plugin.yaml` 刷新凭证，并将输出写入 `skeleton/.env.local`。 |
| 指标缺失 `tenant` 标签 | 确认前端/调用方携带 `X-PowerX-Tenant`，Skeleton API 会自动传入 `InvokeParams.TenantUUID`。 |
| 日志缺失 `traceId` | 检查 Gateway 响应头是否返回 `X-Trace-Id`，若无可联系 PowerX 核心团队确认链路配置。 |

## 5. 参考文件

- `framework/backend/go/observability/tracing.go`
- `framework/backend/go/observability/capability_metrics.go`
- `framework/backend/go/internal/services/capabilityinvoker/service.go`
- `tools/cli/src/executors/doctor.ts`
- `docs/plan/009-consume-powerx-capability.md`（Skeleton 环境与 CLI 操作文档）
