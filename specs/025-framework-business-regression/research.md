# Research: 插件业务场景回归工作台

**Date**: 2026-10-13
**Branch**: `025-framework-business-regression`
**Sources**: 已确认的 [业务方案](../../docs/plan/025-framework-business-regression.md)、`.specify/memory/constitution.md`、当前工作树及相关 CRUD/Nuxt 技能规则。

## R1 - 场景代码作为事实来源

**Decision**: 用统一 SDK 注册场景 Definition、Preflight、Prepare、Execute、Cleanup，Step/Assert/Evidence 通过 Recorder 输出；Definition 明确非空且唯一的 required_assertion_keys，运行逐项核对，空声明或漏报不能通过；原生测试可调用同一场景。
**Rationale**: 已确认页面直接执行，环境不应依赖源码与 Go 工具链；业务断言需要结构化过程，不能只读取 go test 文本。
**Alternatives considered**: 浏览器执行任意脚本、仅扫描 *_test.go 或只展示 CI 报告；与已确认的页面执行和场景 SDK 方向不符。
**Evidence**: `scripts/testing/regression.sh` 目前聚合测试与报告，缺少公共场景 Registry、Run 和 Assertion 契约。

## R2 - 公共机制与业务 ownership

**Decision**: Framework 拥有 SDK、执行器、Store/GORM adapter、报告和通用工作台；插件拥有场景、正式业务依赖、实体模型映射、迁移、权限与启动配置。
**Rationale**: 其他插件可复用机制，同时 Framework 不引入订单、支付、会员领域或引用 Skeleton 内部包。
**Alternatives considered**: 每个插件复制调试接口；直接将业务场景加入 Framework；另建 Core 测试平台。

## R3 - 独立 profile，不改正常 ProviderMode

**Decision**: Profile 以稳定 UUID 和 controlled/live 类型声明，启动时固定 factory，每次执行得到独立依赖组；US1 在模板场景与执行器完成后装配 controlled，US3 再扩展 live 并注册真实场景；报告记录逐项实际依赖模式。
**Rationale**: Framework `runtime/module/factory.go` 在启动模式内选择已注入 adapter。现有 metadata Framework Lab 已分别装配 Local/Delegated debug runtime，可沿用独立绑定原则。
**Alternatives considered**: 临时修改全局配置或 host 不可用时改用测试替身；均违反已确认的无降级边界。

## R4 - 插件三表持久化

**Decision**: 保存 Run、CaseExecution、Event；步骤、断言、依赖证据和 fixture 清单保存在场景结果的结构化字段中。套件和场景由代码定义，不新增可编辑用例数据库。
**Rationale**: 满足刷新、补读、历史与资源清理，避免增加复杂的用例管理和编辑器。结果快照与事件采用同一租户事务提交。
**Alternatives considered**: 仅内存记录；把 completed 任务当作通过结果；每个步骤与断言独立建表。
**Evidence**: 当前 `runtime/taskcenter` 只定义任务记录；插件 `BaseRepository.WithTenantTx` 已提供事务和 PostgreSQL 租户上下文。

## R5 - GORM adapter 与目录/版本基线

**Decision**: Framework 可选 storegorm 接受注入的租户事务回调及 schema-qualified 表名；插件 Repository 组合现有 BaseRepository。插件模型采用实际 `internal/entity/models`，仓储采用实际 `internal/entity/repository`。
**Rationale**: 宪章通用段的 domain 路径与当前活跃代码、crud-model 规则不同；不为本功能引入第二套实体层或 Framework→Skeleton 依赖。当前 package.json 已为 TypeScript 5.x/Nuxt 4.2；按实际栈设计，不修改依赖版本。
**Alternatives considered**: 整仓目录迁移、Framework 直接导入插件仓储或反向降级 TypeScript；均扩大已确认范围。

## R6 - 串行、协作式取消与明确中断

**Decision**: 单测试实例一次仅一个批次或清理作业；重复提交去重；取消与超时使用 context 并等待操作停止，清理使用独立时限；重启标记 interrupted，不自动重放。
**Rationale**: 外部副作用不能用 goroutine 强杀或状态标记撤销；串行降低 fixture 并发污染，并符合首期复杂度。
**Alternatives considered**: TaskBus 调度回归、多实例队列、超时后直接宣告清理完毕；首期不采用。

## R7 - SSE 与一致性

**Decision**: 复用现有 `runtime/realtime` 和 Framework Client 的会话绑定；RegressionEvent.meta 复用 `event.Meta` 的六项必填关联字段，严格校验、持久化并在回放保持原值；事件事务提交后广播；事件 UUID 游标、运行内序号排序；连接先建立订阅再补读并去重。
**Rationale**: 同一页可展示进度，刷新后以持久化快照恢复；避免查询与订阅竞态。身份/租户/会话 revision 变化必须关闭旧连接。
**Alternatives considered**: 文本日志解析、静默轮询、客户端假造完成状态；不满足证据与无 fallback 要求。

## R8 - CLI、权限与报告

**Decision**: CLI 调用现有受保护插件接口，显式读取测试身份凭据，使用相同 SDK 契约与 SSE；JSON 报告是完整事实源，JUnit 为自动化展示映射。
**Rationale**: 页面与 CLI 不维护第二套测试实现；集成业务调用沿用现有 STS/授权，工作台不自动 grant。
**Alternatives considered**: CLI 使用 Dev API 凭据绕过管理端权限；自动开权限；命令行重新执行独立业务脚本。

## R9 - 验收与实施默认值

**Decision**: 先 Go/Nuxt，显式启用开发/专用测试实例；源码、迁移、页面、受控场景、真实授权和业务读回分别验收。报告默认保留，测试资源默认清理，无自动历史删除策略。
**Rationale**: 已确认两类回归和统一接入，但未要求生产开放、其他栈实现或失败 fixture 自动保留。当前仅生成设计文档，不声称任何业务验收通过。
**Alternatives considered**: 发布时自动执行 live、单靠构建/单测宣布完成或默认把测试接口开放到普通生产实例。

所有影响首期范围的选择均来自已确认方案或上述明确默认值；无未解决澄清项。实际环境权限、外部测试服务及临时进程归属在实施验收时检查。
