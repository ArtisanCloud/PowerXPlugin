# Research Notes — 插件能力注册与暴露治理闭环

## Decision 1 — 能力目录结构
- **Decision**: 使用 `plugin.yaml` + `capabilities/*.yaml` 的分文件目录，所有协议矩阵与复合任务引用在能力文件内完成，Cap Manager 统一解析。
- **Rationale**: 减少 manifest 膨胀，便于多团队维护与差异比对；符合 spec 对模块化引用的要求。
- **Alternatives Considered**:
  - 单一 `plugin.yaml` 维护全部能力（不可扩展，review 难度高）。
  - 数据库存储能力目录（与仓库脱节、影响离线构建）。

## Decision 2 — 能力同步失败策略
- **Decision**: 插件安装/升级若在向 PowerX 同步能力目录或协议资产时出现错误，立即阻断并回滚到上一版本。
- **Rationale**: 保证宿主 Workflow/Agent 始终读取一致目录，避免不完整节点导致运行时故障；spec Clarifications 已明确。
- **Alternatives**: 延迟同步或后台重试（宿主拿到旧目录但已安装新插件，状态不一致）。

## Decision 3 — 执行模式
- **Decision**: 原子能力、Workflow 节点默认同步执行；仅显式 `async` 能力可走回调/SSE，并提供状态查询 + 超时策略。
- **Rationale**: 简化 Workflow/Agent 状态管理，确保 SLA 指标可测；仅少量长任务需要异步。
- **Alternatives**: 全异步（复杂度高）；由管理员临时选择（缺乏契约，难以自动生成 SDK）。

## Decision 4 — 多协议资产生成责任划分
- **Decision**: Capabilities Manager 负责在构建/安装时生成 OpenAPI、Proto、Workflow Step、MCP manifest、Agent SSE、Webhook、SDK Bundle，并暴露 `ListCapabilities/ExportProtocols/RegisterWithHost`。
- **Rationale**: 统一出口、自动化生命周期；满足 spec 中 3 分钟同步 KPI。
- **Alternatives**: 由各通道独立脚本生成（重复劳动、难以保持一致）；宿主在运行时推断（需了解插件内部实现，违背 Host Contract First）。
