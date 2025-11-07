# Research Findings — Publish Hub

## Decision 1: `.pxp` Artefact + Integrity Chain
- **Rationale**: 离线客户与 Marketplace 审核依赖统一包体；`.pxp` 必须打包 manifest、后端/前端 artefact、migrations、assets，并附带 `integrity.txt`、`manifest.signature`、`dist/audit.log` 以实现可追溯链路。
- **Alternatives considered**: 仅上传原始 dist 目录（缺乏签名/大小控制）、将 artefact 拆分为多包（让审核与回滚复杂）。统一 `.pxp` + integrity 更符合离线 SLA 并便于 CLI/Marketplace 自动校验。

## Decision 2: Dev API register/reload 契约
- **Rationale**: 热加载是发布链起点，register 提交 manifest + bundle 元数据，reload 提交差异与诊断，DELETE 清理 session。接口必须支持 mTLS、幂等 `x-reload-id`、返回 Admin 预览链接与 session TTL；Telemetry 事件 `dev.hotload.*` 在 CLI 与 Backend 同步。
- **Alternatives considered**: 通过 WebSocket 双向推送（复杂度高且需额外隧道）、直接访问数据库（违背契约优先原则）。RESTful Dev API + SSE 日志最贴合现有 Gin 路径。

## Decision 3: 在线/离线审核 SLA + 通知策略
- **Rationale**: 市场运营需要区分在线（≤4h）与离线（≤1 工作日）队列，并在 SLA 超时自动通知发布者/运营；批准后 30 分钟内向订阅租户广播 `plugin.publish.approved`，并允许灰度或白名单策略。
- **Alternatives considered**: 统一 SLA（无法覆盖离线人工流程）；仅人工提醒（无法保证透明度）。SLA + 自动告警提供可追踪 KPI 并满足宪章的透明交付原则。

## Decision 4: CLI 命令族参数化
- **Rationale**: `px-plugin dev --watch`、`px-plugin publish --channel`、`px-plugin dist --target offline --sign` 需要标准化参数、示例与校验。CLI 负责预检、签名、Telemetry 上报与错误分级，脚手架通过 `px-plugin init` 验证一致性。
- **Alternatives considered**: 将 dist/publish 合并为单命令（导致在线/离线策略耦合）；热加载依赖 IDE 扩展（影响跨平台体验）。保留三条命令线最贴近现行工作流。
