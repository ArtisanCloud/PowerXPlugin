# Feature Specification: 插件能力注册与暴露治理闭环

**Feature Branch**: `006-plugin-capability`  
**Created**: 2025-12-04  
**Status**: Draft  
**Input**: User description: "请对docs/use_cases/_from_hub/SCN-INT-PLUGIN-CAPABILITY-001下的各种用例场景，生成相应的spec文档"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 开发者 5 分钟完成能力建模 (Priority: P1)

插件开发者打开能力注册表单或 CLI，引导式填写能力名称、场景、输入输出 Schema、敏感字段、示例与 Demo 链接；系统实时校验命名冲突与字段格式，在 5 分钟内完成所有校验并生成能力 ID。

**Why this priority**: 没有结构化建模，后续审核、暴露与通知都无法启动，是整个闭环的入口。

**Independent Test**: 以一条全新能力为例，模拟从草稿到提交的完整流程，验证校验反馈、草稿恢复和审计事件即可衡量本故事是否交付价值。

**Acceptance Scenarios**:

1. **Given** 表单或 CLI 已加载官方模板，**When** 开发者填写完所有必填字段并点击“提交”，**Then** 系统在 5 分钟内完成校验、返回唯一能力 ID 并写入审计。
2. **Given** 命名或字段与已注册能力冲突，**When** 校验引擎检测到冲突，**Then** 系统在 10 秒内提示冲突字段、给出修复建议并允许保存草稿。

---

### User Story 2 - 多角色审核与整改闭环 (Priority: P2)

安全、合规与运营审核人根据能力标签被自动指派任务，查看数据分级、权限矩阵与调用频率风险，能在流程内发表评论、退回整改，并确保高敏能力双人复核于 2 个工作日内完成。

**Why this priority**: 审核确保能力合法合规，是“可暴露”状态的唯一入口，直接影响平台安全。

**Independent Test**: 创建一条带高敏标签的能力，验证审核任务派发、SLA 计时、退回整改与双人复核链路即可独立验收。

**Acceptance Scenarios**:

1. **Given** 能力提交并标记为高敏，**When** 审核任务生成，**Then** 系统同时指派至少 2 名审核人、启动 SLA 计时并在看板中展示截至时间。
2. **Given** 审核人要求补充材料，**When** 开发者在平台上传整改说明，**Then** 工作流自动恢复到审核节点且所有历史评论与附件可追溯。

---

### User Story 3 - 宿主管理员配置暴露通道与租户授权 (Priority: P3)

宿主管理员在能力目录中选择需要暴露的通道（REST/GraphQL/gRPC/OpenAPI/Webhook/Workflow Step/Agent Tool/Agent SSE/SDK）、配置鉴权策略、限流与熔断参数，并为租户或项目授予额度，系统需在 3 分钟内同步至 API 网关、Workflow Builder、Agent Hub 与开发者门户，并生成文档、Postman、SDK 示例。

**Why this priority**: 宿主只有在暴露配置生效后才能将能力开放给订阅方，直接影响业务启用速度与安全隔离。

**Independent Test**: 以已审核通过的能力为起点，配置不同通道和租户授权，核验文档生成与未授权拦截行为即可单独验收。

**Acceptance Scenarios**:

1. **Given** 能力处于“可暴露”状态，**When** 管理员提交暴露配置，**Then** 网关策略、门户文档与通知在 3 分钟内同时更新且状态板显示“生效”。
2. **Given** 未被授权的租户调用该能力，**When** 请求到达宿主平台，**Then** 系统立即阻断、记录 `audit.capability.exposure.denied` 并向管理员推送提醒。

---

### User Story 4 - 订阅方感知能力版本变更与下线 (Priority: P4)

运营或合规负责人发起能力版本升级或下线申请，系统生成差异报告、灰度计划和通知模板；订阅方通过邮件/站内/Webhook 在灰度期开始前收到迁移指引，并可在灰度结束前确认或反馈；若调用未清零，系统暂停下线或触发回滚。

**Why this priority**: 生命周期治理避免突发变更导致订阅方业务中断，是平台可信的关键承诺。

**Independent Test**: 以已有订阅方的能力为例，模拟版本升级、灰度监控与失败回滚的全流程，即便不改动其他故事也能独立交付通知与回滚能力。

**Acceptance Scenarios**:

1. **Given** 运营提交版本升级计划，**When** 系统完成差异分析，**Then** 自动生成影响范围、灰度窗口与订阅方清单并允许多渠道通知配置。
2. **Given** 灰度结束但仍有租户调用旧版本，**When** 系统检测到残留流量，**Then** 自动暂停下线、告警负责人与提供“一键回滚”入口。

---

### User Story 5 - 插件能力目录与 PowerX Workflow/Agent 统一同步 (Priority: P2)

PowerX 底座在插件安装或版本升级时调用 Capabilities Manager，获取 `capabilities/*.yaml` 与 `contracts/exposure/*`，自动将原子能力、复合任务、Workflow Step 模板与 Agent 工具注册至 `capability_registry`、Workflow Builder 与 Agent Hub，使宿主能够在拖拽节点或智能体调度时即刻引用插件能力。

**Why this priority**: 若 PowerX 无法及时识别插件能力，Workflow/智能体入口将无法复用插件生态，整体方案失去意义。

**Independent Test**: 安装一个示例插件，验证 Capabilities Manager 输出的目录可被 PowerX 接口消费，Workflow Builder 出现插件节点，智能体工具库自动加入 MCP/SSE 工具，确保目录更新可在 3 分钟内同步。

**Acceptance Scenarios**:

1. **Given** 插件安装完成并暴露 `capabilities/*.yaml`，**When** PowerX 启动能力同步流程，**Then** `/admin/capability_registry` 中新增对应能力记录，Workflow/Agent 后台可查询到节点或工具。
2. **Given** 插件新增复合 Workflow 与 Agent SSE 模板，**When** Capabilities Manager 导出最新协议并提交，**Then** Workflow Builder 可拖拽该 Workflow 节点，Agent Hub 可在意图匹配时调度该工具并接收 SSE 事件。

### Edge Cases

- 当能力名称、别名或 URI 与现有能力冲突时，系统必须列出冲突详情并阻止提交，避免 ID 分配错误。
- 贴有高敏标签的能力若仅有一位审核人在线，系统需自动升级给备份审核人，并在 SLA 前 4 小时提醒主管。
- 租户额度被瞬时消耗完或调用速率突破设定阈值时，应触发熔断并通知租户与管理员决定是否扩容。
- 订阅通知渠道（如 Webhook）三次失败仍未送达，需要自动切换到备用渠道并记录未达名单，供运营人工跟进。
- 当插件方提交下线计划但存在未完成的整改项时，系统须阻断下线申请并提示需完成的审核任务。
- 如果 `capabilities.imports` 引用的能力文件缺失或能力 ID 重复，系统必须阻断打包并指明具体文件，防止 PowerX 载入错误目录。
- 当 Capabilities Manager 导出的 Workflow Step 模板与 PowerX `NodeKind` 不匹配时，需要记录校验错误并回退到上一版本，避免 Workflow Builder 加载失败。
- 若能力目录或多协议资产同步至 PowerX 失败，必须立即终止插件安装/升级并回滚，避免宿主载入不完整目录。
- 原子能力和 Workflow 节点默认同步执行；仅当能力明确标记为 `async` 时才允许通过回调或 SSE 完成，且需附带状态查询与超时策略。

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 能力建模界面与 CLI 必须提供官方模板、字段说明与实时校验，帮助 95% 的提交在 5 分钟内完成并生成能力 ID。
- **FR-002**: 系统必须在提交时将能力元数据、示例与敏感标签写入审计，并支持草稿恢复与多语言字段提示。
- **FR-003**: 审核工作流必须依据能力敏感度自动指派角色、启动 SLA 计时、记录评论/附件，并在超时前推送升级提醒。
- **FR-004**: 高敏能力审核必须执行双人复核，任何拒绝原因要结构化保存且可供导出报表。
- **FR-005**: 暴露配置界面需支持多通道选择、鉴权策略、限流/熔断、租户额度与生效预览，并在 3 分钟内同步到调用入口与文档。
- **FR-006**: 系统必须阻断所有未授权或超额度的调用，记录审计并支持管理员即时调整额度或停用通道。
- **FR-007**: 文档、Postman 集合与 SDK 示例需在每次配置或版本更新后自动生成，保证与最新契约保持一致。
- **FR-008**: 生命周期管理需提供版本差异分析、灰度计划、通知编排、双版本并行及失败回滚控制，通知覆盖率需达到 100%。
- **FR-009**: 所有通知（审批提醒、暴露生效、版本变更）必须支持邮件、站内消息与 Webhook，多渠道失败需重试与升级。
- **FR-010**: 系统需提供可审计指标（注册耗时、审核 SLA、暴露生效时长、通知覆盖率）以供运营看板展示。
- **FR-011**: 插件 manifest 必须支持 `capabilities.imports` 与分文件目录，CLI 能解析 `capabilities/*.yaml`、`contracts/capabilities/*.yaml` 及 `contracts/schema/**`，并在打包前阻断重复 ID 或缺失文件。
- **FR-012**: Capabilities Manager 需要在插件安装/升级时导出统一协议资产（OpenAPI、Proto、Workflow Step、MCP manifest、Agent SSE、Webhook、SDK Bundle），并调用 PowerX 注册接口在 3 分钟内同步至 `capability_registry`、Workflow Builder 与 Agent Hub。
- **FR-013**: 复合任务应以 DAG/Workflow 模板描述 `kind/use/params/io`、输入输出映射与回滚策略，完全兼容 PowerX `pkg/corex/flow/schemas.Node`，并支持作为 Workflow 节点或插件 Workflow 导入。
- **FR-014**: 智能体工具同步需覆盖 MCP 工具清单与 SSE 通道，PowerX Agent Hub 在意图触发时必须能够选择插件原子能力、复合任务或插件 Workflow，调用时自动完成租户/额度校验与审计。
- **FR-015**: 若能力目录或协议资产注册到 PowerX 失败，安装/升级流程必须立即阻断并回滚到上一版本，待问题修复后重新触发同步。
- **FR-016**: 原子能力与 Workflow 节点默认采用同步请求-响应模式；若声明 `async`，必须提供回调/SSE 说明、状态查询接口与超时/重试策略。

### Key Entities *(include if feature involves data)*

- **能力条目（Capability Record）**：存放名称、描述、输入输出 Schema、敏感标签、示例与当前状态，用于贯穿建模至下线的主键对象。
- **审核任务（Review Task）**：绑定能力 ID、角色、SLA、风险等级、评论与附件，是跟踪整改与复核的核心单元。
- **暴露配置（Exposure Package）**：描述选中的通道、鉴权策略、限流额度、租户授权与对应文档版本，关联多个租户或项目。
- **租户订阅（Tenant Subscription）**：记录租户/项目对能力的启用状态、额度、消耗与通知偏好，用于触发熔断与通知。
- **生命周期计划（Lifecycle Plan）**：储存版本差异、灰度窗口、通知渠道、回滚策略与执行状态，关联订阅方确认结果。
- **能力目录（Capability Catalog）**：由 `plugin.yaml` 与 `capabilities/*.yaml` 组成，描述原子能力、协议矩阵与复合任务，是 Capabilities Manager 与 PowerX 同步的唯一数据源。
- **能力管理器（Capabilities Manager）**：CLI/运行时组件，负责解析能力目录、生成协议资产、暴露 `ListCapabilities`/`ExportProtocols`/`RegisterWithHost` 等接口供 PowerX 调用。
- **Workflow & Agent 模板（Workflow/Agent Templates）**：位于 `contracts/exposure/workflow/*.json`、`contracts/exposure/mcp-tools.json`、`contracts/exposure/agent-streams/*.yaml`，定义 `NodeKind`、MCP manifest、SSE 协议，是 Workflow Builder 与 Agent Hub 识别插件节点/工具的依据。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: ≥95% 的能力提交在 5 分钟内完成自动校验并生成能力 ID，命名/Schema 冲突率 <2%。
- **SC-002**: ≥98% 的审核任务在 2 个工作日内完成；高敏能力 100% 完成双人复核，无未记录的拒绝理由。
- **SC-003**: 暴露配置生效耗时 ≤3 分钟，未授权调用阻断率 100%，租户额度变更即时可见。
- **SC-004**: 版本变更或下线通知覆盖率 100%，灰度期内异常触发回滚的处理时间 ≤30 分钟，且所有操作写入审计。
- **SC-005**: 插件安装或更新后 3 分钟内，PowerX Workflow Builder 与 Agent Hub 均能展示最新能力节点/工具，多协议协议资产导出成功率 100%，能力目录同步失败率 <1%。

## Assumptions

- 能力注册、审核、暴露与生命周期流程在同一门户中串联，用户无需在多个系统重复输入数据。
- IAM、API Gateway、通知中心等基础设施已可用，本规范只定义与能力治理直接相关的需求与指标。
- 插件生态仍以内存或 Mock 持久层为主，但需要预留与宿主实际 API 对接的契约字段，确保未来引入持久层无需大幅改造。
- PowerX Core 的 `capability_registry`、Workflow Builder、Agent Hub 会按 `pkg/corex/flow/schemas.Node`、MCP manifest 与 Agent SSE 协议解析插件导出的能力目录，因此插件侧必须严格遵守该结构约定与字段命名。

## Clarifications

### Session 2025-12-04

- Q: 对于插件安装或升级时能力目录/协议资产同步 PowerX 失败的处理策略是什么？ → A: 立即终止安装并回滚到上一版本，待问题修复后重新同步。
- Q: 插件能力在 Workflow/Agent 中执行时默认是同步还是异步？ → A: 默认同步执行，仅 `async` 标记的能力可通过回调或 SSE 完成。
