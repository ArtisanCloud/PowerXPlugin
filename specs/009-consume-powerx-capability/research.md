# Research Log - PowerX 通用能力插件消费

## 决策 1：统一 Gateway Client 形态
- **Decision**: Go backend 与 Nuxt 前端均提供框架层 `InvokeCapability(capabilityId, action, payload)` 封装，内部自动注入 Tool Token、`X-Tenant-UUID`、`X-Request-ID`，并以 `/tenant/invocations` 为唯一 HTTP 入口（gRPC 走 `IntegrationGatewayTenantService`）。
- **Rationale**: 消除插件自行拼装请求导致的安全/兼容性问题，使升级 Gateway 契约时只需更新框架层。
- **Alternatives considered**: 允许各业务直接调用 Gateway（风险高，无法统一治理）；由 CoreX 下发 SDK（会与插件框架重复，放弃）。

## 决策 2：凭证与环境变量管理
- **Decision**: 宿主模式由部署系统注入 `PX_PLUGIN_TOOL_TOKEN`/`PX_GATEWAY_BASE_URL`，Skeleton 执行 `px-plugin login --manifest ./skeleton/plugin.yaml` 后写入 `.env.local`，框架负责检测过期并提示刷新。
- **Rationale**: 贴合现有 STS/Tool Grant 体系，避免开发者硬编码凭证；同一封装可在 CI/本地/生产切换。
- **Alternatives considered**: 通过配置文件手动维护 Token（风险高、易泄露）；完全依赖宿主 Admin 注入（Skeleton 无法使用）。

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
