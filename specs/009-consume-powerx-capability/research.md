# Research Log - PowerX 通用能力插件消费

## 决策 1：统一 Gateway Client 形态
- **Decision**: Go backend 与 Nuxt 前端均提供框架层 `InvokeCapability(capabilityId, action, payload)` 封装，内部自动注入 Tool Token、`tenant_uuid`、`X-Request-ID`，并以 `/tenant/invocations` 为唯一 HTTP 入口（gRPC 走 `IntegrationGatewayTenantService`）。
- **Rationale**: 消除插件自行拼装请求导致的安全/兼容性问题，使升级 Gateway 契约时只需更新框架层。
- **Alternatives considered**: 允许各业务直接调用 Gateway（风险高，无法统一治理）；由 CoreX 下发 SDK（会与插件框架重复，放弃）。

## 决策 2：凭证与环境变量管理
- **Decision**: 引入 Delegated Gateway Contract v1：宿主模式仅接受 `PX_GATEWAY_BASE_URL` + `PX_PLUGIN_TOOL_TOKEN` + `PX_GATEWAY_AUTH_SCHEME=bearer`，插件侧不再读取 `PX_TOOL_TOKEN` 与 `PX_GATEWAY_API_KEY` 作为 delegated 凭证；缺失即 fail-fast。
- **Rationale**: 消除多变量 fallback 导致的歧义，快速定位问题归属（宿主注入 vs 插件实现），保证安装/启用后行为一致。
- **Alternatives considered**: 保留兼容期（排障复杂、长期双轨）；插件本地兜底推断 delegated 凭证（与宿主契约冲突）。

## 决策 2.1：宿主安装/启用注入与探活
- **Decision**: PowerX 在插件启用链路必须完成变量注入，并在 PostEnable 执行一次插件进程内凭证探活；失败标记 `enable_failed_missing_gateway_credential`。
- **Rationale**: 防止“表面启用成功，但首次调用才失败”的体验割裂。
- **Alternatives considered**: 仅依赖插件启动日志人工排查（反馈慢）；仅靠配置中心推送成功标记（无法保证进程内可见）。

## 决策 3：Mock & 降级策略
- **Decision**: Skeleton 允许通过 `--use-mock=<module>` 参数启用内存 Mock；当 Gateway 不可达时 2 秒超时并提示，若未启用 Mock 则直接抛错。
- **Rationale**: 降低 Dev Gateway 依赖，保证离线开发；同时避免静默返回假数据。
- **Alternatives considered**: 永远使用真实 Gateway（本地可能不可用）；全自动切换 Mock（难以察觉真实调用失败）。

## 决策 4：观测与限流
- **Decision**: 在调用封装中注入日志（含 capabilityId/tenantUUID/traceId）、指标（成功率/耗时/限流事件）与审计事件；提供 `scripts/capabilities/run-from-package.mjs` 供 QA/开发按能力调试，并输出 traceId。
- **Rationale**: 满足宪章的可观测与零信任要求，快速定位问题并验证 quotas。
- **Alternatives considered**: 仅在宿主端观测（Skeleton 无法排查）；依赖 Gateway 自带日志（缺失插件上下文）。

## 决策 5：文档与速查
- **Decision**: 更新 `docs/plan/009-consume-powerx-capability.md`、新增 quickstart 与速查表，引导 manifest `requiredCapabilities`、CLI 校验、凭证刷新、常见能力调用。
- **Rationale**: 让新插件在 30 分钟内完成首个调用并遵守流程。
- **Alternatives considered**: 仅靠代码注释（易遗漏）；把说明散落在多处文档（不利传播）。
