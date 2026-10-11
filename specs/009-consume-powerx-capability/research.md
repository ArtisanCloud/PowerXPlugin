# Research Log - PowerX 通用能力插件消费

## 决策 1：通用 Gateway 与强类型 Host Client 分层
- **Decision**: 通用 capability 继续使用 `InvokeCapability(capabilityId, action, payload)`；已有正式 Core Host Contract 的 IAM、Knowledge、Media、Agent、AI 等模块通过各自的 Framework typed client 调用。tenant 永远由已认证凭证推导，前端与请求 payload 不传 `tenant_uuid`。
- **Rationale**: 消除插件自行拼装请求导致的安全/兼容性问题，使升级 Gateway 契约时只需更新框架层。
- **Alternatives considered**: 允许各业务直接调用 Gateway（风险高，无法统一治理）；由 CoreX 下发 SDK（会与插件框架重复，放弃）。

## 决策 2：凭证与环境变量管理
- **Decision**: 安装态 delegated 只使用 Framework 注入的 STS service token provider；Skeleton 开发调试可显式使用 Gateway API Key。两种凭证均由 Core 校验 capability 发布记录、tenant registration 和 credential grant；缺失或无权即失败。
- **Rationale**: 消除多变量 fallback 导致的歧义，快速定位问题归属（宿主注入 vs 插件实现），保证安装/启用后行为一致。
- **Alternatives considered**: 保留兼容期（排障复杂、长期双轨）；插件本地兜底推断 delegated 凭证（与宿主契约冲突）。

## 决策 2.1：宿主安装/启用注入与探活
- **Decision**: PowerX 在插件启用链路必须完成变量注入，并在 PostEnable 执行一次插件进程内凭证探活；失败标记 `enable_failed_missing_gateway_credential`。
- **Rationale**: 防止“表面启用成功，但首次调用才失败”的体验割裂。
- **Alternatives considered**: 仅依赖插件启动日志人工排查（反馈慢）；仅靠配置中心推送成功标记（无法保证进程内可见）。

## 决策 3：失败策略
- **Decision**: Host Contract Lab 不支持 Mock、local provider、替代 credential 或替代协议的自动降级。认证、授权和依赖故障必须以稳定 `reason_code` 返回。
- **Rationale**: 调试台的目的是真实验证 Host Contract 与 grant；假成功会掩盖安装态问题。
- **Alternatives considered**: 自动切换 Mock 或 local provider（均会破坏验收真实性）。

## 决策 4：观测与限流
- **Decision**: 在调用封装中记录 capabilityId、credential-derived tenant、traceId、耗时、稳定错误码与审计事件；提供受控后端 probe 供 QA/开发按模块调试。
- **Rationale**: 满足宪章的可观测与零信任要求，快速定位问题并验证 quotas。
- **Alternatives considered**: 仅在宿主端观测（Skeleton 无法排查）；依赖 Gateway 自带日志（缺失插件上下文）。

## 决策 5：文档与速查
- **Decision**: 更新 `docs/plan/009-consume-powerx-capability.md`、quickstart 与速查表，引导 manifest `capabilities.required`、安装升级后的 grant 同步、强类型模块 probe 与常见错误处理。
- **Rationale**: 让新插件在 30 分钟内完成首个调用并遵守流程。
- **Alternatives considered**: 仅靠代码注释（易遗漏）；把说明散落在多处文档（不利传播）。
