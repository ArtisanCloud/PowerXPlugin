# Data Model: 插件业务场景回归

**Branch**: `025-framework-business-regression`
**Status**: 实现契约；迁移已通过 SQLite 与独立 PostgreSQL 验证。

## 1. 定义对象

| 对象 | 必要字段 | 来源 |
| --- | --- | --- |
| SuiteDefinition | uuid、key、title_key、description_key | 启动时代码注册 |
| CaseDefinition | uuid、suite_uuid、key、version、title_key、description_key、tags、supported_profile_kinds、input_schema、execution_timeout_ms、cleanup_timeout_ms、非空步骤定义、required_assertion_keys | 启动时代码注册 |
| ProfileDefinition | uuid、key、title_key、kind(controlled/live)、dependency_declarations、fixture_namespace、binding_revision | 启动配置与 factory 注册 |
| StepDefinition | uuid、key、title_key、phase、验证边界 | 场景代码注册 |

所有对象引用使用 UUID。key、version、phase、序号及 schema 字段为机器语义，不用作业务对象的替代引用。场景目录只有当前定义，历史执行必须读取被冻结的定义。binding_revision 在启动时由非敏感依赖目标配置与 fixture_namespace 计算；清理重试必须与原快照匹配，不把凭据纳入报告。

required_assertion_keys 是场景内非空、唯一的机器断言 key 列表；每个 key 非空且不能含首尾空白，属于机器语义而非业务对象引用。注册时缺失、空列表或重复项明确失败。该列表随 definition_snapshot 冻结，运行时不能修改或降级。

## 2. 持久化记录

插件 schema 中新增三类表。表名常量在插件 `internal/entity/models/model.go` 注册，模型 `TableName()` 调用 `models.S(...)`。

### regression_runs

| 字段 | 类型/约束 | 用途 |
| --- | --- | --- |
| uuid、tenant_uuid、actor_uuid、profile_uuid、client_request_uuid | UUID，非空 | 运行和可信身份/配置引用 |
| rerun_of_run_uuid | UUID，可空，同租户 | 原始批次关联 |
| profile_snapshot、selection_snapshot、input_snapshot、source_snapshot | 结构化 JSON，脱敏，非空 | 配置、定义、输入、插件/源码版本 |
| phase | queued/running/cancelling/completed/cancelled/interrupted | 执行生命周期 |
| verdict | pending/passed/failed/blocked | 业务结论 |
| cleanup_state | pending/running/succeeded/failed/not_required | 当前资源清理状态 |
| reason_code、message_key | 机器码/i18n key | 诊断与显示 |
| revision | 正整数 | 快照与事件的一致性版本 |
| created_at、updated_at、started_at、finished_at | UTC 时间；后两项可空 | 审计与耗时 |

唯一约束 `(tenant_uuid, actor_uuid, client_request_uuid)`。同一请求 UUID 与同一输入返回原记录，不同输入冲突；同租户原始批次才允许 rerun 引用。

### regression_case_executions

| 字段 | 类型/约束 | 用途 |
| --- | --- | --- |
| uuid、tenant_uuid、run_uuid、case_uuid | UUID，非空 | 场景结果及关联 |
| ordinal | 正整数，运行内唯一 | 场景执行顺序，不是对象 ID |
| definition_snapshot、input_snapshot、dependency_evidence | 结构化 JSON，非空 | 实际定义、输入和依赖方式 |
| phase | pending/preflight/preparing/executing/verifying/cleaning/finished | 当前执行阶段 |
| verdict | pending/passed/failed/blocked/skipped | 场景结论 |
| steps、fixture_resources | 结构化 JSON 数组，非空 | 步骤、断言与资源清单 |
| missing_assertion_keys | 唯一机器 key 数组，非空值或空数组 | 最终核对未报告的必要断言；通过时必须为空；初始为空且 verdict=pending |
| cleanup_state、reason_code、message_key、revision | 与运行语义一致 | 清理及诊断 |
| created_at、updated_at、started_at、finished_at | UTC 时间 | 审计与耗时 |

唯一约束 `(tenant_uuid, run_uuid, ordinal)`。相同批次不重复选择同一 case_uuid。所有关联查询同时检查 tenant_uuid。

### regression_events

| 字段 | 类型/约束 | 用途 |
| --- | --- | --- |
| uuid、tenant_uuid、run_uuid | UUID，非空 | 事件引用与运行游标范围 |
| case_execution_uuid | UUID，可空 | 场景事件关联 |
| sequence、revision | 正整数 | 运行内顺序与快照版本 |
| kind | run.created/run.updated/case.updated/assertion.recorded/cleanup.updated/run.finished | 结构化事件类型 |
| meta | 完整 EventMeta JSON，非空 | 复用 Framework event.Meta；六项必填关联元信息 |
| payload | 结构化 JSON，非空，已脱敏 | 对应变化的完整快照或断言 |
| created_at、updated_at | UTC 时间 | 审计 |

唯一约束 `(tenant_uuid, run_uuid, sequence)`。UUID 是公开游标；sequence 仅排序。未知或非当前运行/租户的游标明确拒绝。payload 的类型由 kind 明确区分，详见 scenario-sdk.md，不以自由文本解释状态。

EventMeta 必填 tenant_uuid、request_id、trace_id、source_plugin、occurred_at、payload_version。Go 复用 `framework/backend/go/event.Meta`，为回归事件执行严格校验：tenant_uuid 与事件/批次一致，请求/追踪/来源字符串非空，发生时间为有效 UTC，payload_version 固定 v1。来源取运行 manifest，身份/请求信息取可信操作 Scope；缺失时明确失败，不从外层别名字段回填。meta 与快照/事件原子保存，SSE 补读及报告输出保留原值。

## 3. 步骤、断言与 fixture

- StepResult：uuid、step_uuid、phase、title_key、status、开始/结束时间、assertions。
- AssertionResult：uuid、key、title_key、status(passed/failed/blocked/skipped)、required、expected、actual、evidence、reason_code。
- Evidence：kind、boundary、source_ref、resource_uuid、trace_id、request_id、observed_at、sanitized_data。source_ref 是声明的业务入口或读取来源；不能以日志中的自然语言填充 actual。
- DependencyEvidence：dependency_key、declared_mode、actual_mode、验证边界、必要能力及授权结果；controlled/live 标签不能替代逐项真实依赖说明。
- FixtureResource：uuid、resource_type、resource_uuid、display_name、role_key、parent_resource_uuid、tenant_uuid、run_uuid、case_execution_uuid、cleanup_key、cleanup_parameters、cleanup_state、cleanup_evidence、cleaned_at。登记对象本身和目标业务对象均有 UUID。名称为登记时脱敏快照，角色为插件 locale key；来源是同场景已登记对象的 UUID，不解析名称。清理读回证据必须明确关联目标资源，成功时间为 UTC。字段保存在现有 JSON 快照中，不新增业务表迁移。

fixture 在创建后立即登记；登记失败必须停止并撤回已创建资源。部分 Prepare 失败仍清理已登记资源。清理前验证租户、归属和声明的清理动作，禁止接受任意浏览器传入的待删除对象。

AssertionResult.required 由被冻结的 required_assertion_keys 派生，不接受调用方降为 false。每个 key 在同一场景执行中至多产生一个终态断言；重复报告明确失败。必要断言 passed 时 evidence 必须非空且字段有效，不能用空证据支撑通过。

## 4. 状态迁移与结论

- Run：queued → running → completed；取消经 cancelling → cancelled；进程恢复将未终结记录变为 interrupted。queued 中的取消可以直接进入 cancelled，并注明未开始。
- Case：pending → preflight → preparing → executing/verifying → cleaning → finished。前置阻塞、业务失败、超时或取消均停止依赖的后续步骤，已有资源仍进入清理。
- Step：pending → running → passed/failed/blocked/skipped；预期业务拒绝可通过断言成为 passed。
- Cleanup：pending → running → succeeded/failed；无资源且无需清理时为 not_required。重试从 failed 发起新清理记录，保留原始 verdict。
- 聚合优先级：必要断言失败、正常 Execute 返回后漏报必要断言、重复报告、非预期操作错误或清理失败 → failed；否则必要场景/断言/证据阻塞、跳过、取消或中断 → blocked；只有非空必要断言集合中的每一项恰好产生一个 passed 结果、必要证据完整且清理成功/无需清理，才为 passed。尚在执行时保持 pending。

正常执行结束却漏报时使用 REGRESSION_ASSERTION_MISSING，并记录 missing_assertion_keys；重复报告使用 REGRESSION_ASSERTION_DUPLICATE。必要证据不可获得时对应断言 blocked，原因为 REGRESSION_EVIDENCE_MISSING。前置依赖阻塞、超时、取消或中断导致断言未执行时记录其未执行原因并保持非通过，不伪造成正常执行后的漏报，也不伪造 expected/actual 或证据。

取消/超时只是停止请求，须等待场景协作式退出后再写终结状态，不声称已撤销外部副作用。不能以 phase=completed 推导 verdict=passed。

## 5. 存储、迁移与恢复

- 表含明确 gorm/json 标签、UTC created_at/updated_at；UUID 作为新表主键，不新增 numeric id 对外引用。
- Store 的快照更新与事件追加必须原子提交；结果写入失败时明确终止，不退化到只在内存展示。
- Framework storegorm 通过注入租户事务函数访问，插件 Repository 组合 BaseRepository；PostgreSQL 执行 SET LOCAL app.tenant_uuid 并启用 RLS，SQLite 所有读写带显式租户过滤。
- 模型注册现有 AutoMigrate，RLS/index 迁移幂等；新增表的显式回滚仅限本功能对象，不修改业务数据表。启动未授权迁移时不得自动补表。
- 单测试实例重启后仅恢复记录状态，不重执业务；资源清理由管理动作发起。首期不支持共享库上的多实例回归调度。
- 默认保留报告和事件，不自动删除历史；凭据不持久化，输入、expected/actual、证据均按声明脱敏。
