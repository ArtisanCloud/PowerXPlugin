# Research

## Decision 1: 行为裁决基线
- Decision: 以 Nuxt 当前线上行为作为最终裁决基线。
- Rationale: 迁移目标是行为等价，先锁定用户可见行为可降低回归风险与争议成本。
- Alternatives considered: 以 Gin 返回语义为唯一裁决；逐案由评审会裁决。

## Decision 2: 后端契约策略
- Decision: 以既有 Go-Gin 管理端契约为联调权威，不新增 Next 私有接口。
- Rationale: 保持双端共享同一后端能力，避免形成只服务于 Next 的分叉接口。
- Alternatives considered: 为 Next 新增临时接口；通过 BFF 层封装专用接口。

## Decision 3: Gin 变更准入
- Decision: Gin 仅允许“确认缺陷后”的最小化修复。
- Rationale: 双栈并行阶段直接改 Gin 适配 Next 会破坏 Nuxt 稳定行为，且增加不可控回归面。
- Alternatives considered: 允许常规优化与重构并行进入；先改后端再调前端。

## Decision 4: 第一阶段覆盖范围
- Decision: 第一阶段直接覆盖全部页面域（Auth/Templates/IAM/Capabilities/Integration/Operations/Security）。
- Rationale: 已明确交付目标为全域迁移，先做整体验收口径可避免阶段目标反复变更。
- Alternatives considered: 先做最小闭环再逐域扩展。

## Decision 5: 双模式验收深度
- Decision: 验收覆盖全部页面域主链路 + 关键异常场景（会话过期、权限拒绝、路由切换）。
- Rationale: 该深度能有效覆盖生产风险，同时保持执行成本可控。
- Alternatives considered: 仅主链路；全分支穷举验证。

## Decision 6: 发布门禁口径
- Decision: 阻断级与高优先级缺陷必须清零；中低优先级可带风险评估发布。
- Rationale: 在质量与节奏之间保持平衡，确保高风险问题不进入发布。
- Alternatives considered: 全量缺陷清零；仅阻断级清零。

## Decision 7: 差异归因时限
- Decision: 联调差异从发现起 2 个工作日内完成归因并记录处理路径。
- Rationale: 设置明确 SLA 能防止差异长期悬置并拖慢迁移节奏。
- Alternatives considered: 1 个工作日（过紧）；3 个工作日或无时限（风险积压）。

## Decision 8: 回归执行策略
- Decision: 以 Nuxt 关键 E2E 场景为迁移回归硬约束，并在 Next 侧保持 locator 与测试意图可复用。
- Rationale: 复用既有基线能快速发现语义偏差并降低测试重建成本。
- Alternatives considered: 全新设计 Next 专属回归套件。
