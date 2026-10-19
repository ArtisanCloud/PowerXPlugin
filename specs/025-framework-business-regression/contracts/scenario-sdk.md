# 场景 SDK、实时更新与报告契约

**Version**: v1
**Feature**: [spec.md](../spec.md)
**HTTP Contract**: [regression.openapi.yaml](./regression.openapi.yaml)

## 1. Go 场景与公共接口

```go
type Scenario interface {
    Definition() CaseDefinition
    Preflight(context.Context, *Scope) error
    Prepare(context.Context, *Scope) error
    Execute(context.Context, *Scope) error
    Cleanup(context.Context, *Scope) error
}
```

- Registry 注册 SuiteDefinition、Scenario factory 与 ProfileDefinition，按 UUID 查找；启动时校验重复定义、输入 schema、i18n key、profile 支持范围及正数时限。CaseDefinition.steps 和 required_assertion_keys 均为非空数组；必要断言 key 非空、无首尾空白、场景内唯一，缺失或重复时拒绝注册。
- Scope 包含 run_uuid、case_execution_uuid、可信 tenant_uuid/actor_uuid、不可变定义/输入/profile 快照、独立业务依赖、Recorder 和 FixtureRegistry。不能把 gin.Context 保存给后台 worker。
- Profile factory 在启动时绑定，按执行构造独立依赖；只接受服务端注册的 profile_uuid，不接受客户端提交任意 provider/endpoint。binding_revision 由非敏感依赖目标配置与 fixture_namespace 计算，随快照保存，用于拒绝跨环境清理重试。
- Recorder 提供 Step 与 Assert；断言由机器字段、业务读取及 expected/actual 形成，显示文案用 title_key/message_key。required 由被冻结的 required_assertion_keys 派生，调用方不能降级；每个 key 至多一个终态结果，必要断言 passed 必须有非空、有效的 Evidence。
- FixtureRegistry 提供 Register、ListOwned 和清理记录。注册资源携带自身 uuid、真实业务 resource_uuid 和本次归属；Cleanup 只能处理登记资源。
- Register 同时保存业务名称 `display_name`、插件角色 locale key `role_key` 和显式来源 `parent_resource_uuid`。来源必须已登记在同一场景执行内，不能自指或跨场景；不从名称推断关系。业务名称应用相同敏感值脱敏规则。
- `Cleaned(ctx, resourceUUID, evidence)` 必须传入非空的正式读回证据：每项 `kind=readback`、`resource_uuid` 与目标一致，含有效 `boundary`、`source_ref`、`observed_at` 和 JSON `sanitized_data`。成功后保存 `cleanup_evidence` 与 `cleaned_at`。删除响应不替代删除后读回；恢复型清理需正式读取并确认原状态。没有有效证据不能将资源标为已清理。
- Scene/SDK 不依赖 testing.T；原生 feature test 可使用同一 Registry/Runner 和显式测试 Store，但内存 Store 仅用于测试，不能作为工作台持久化失败时的 fallback。

## 2. 正式业务链路与隔离

- Service 场景使用真实业务 Service/Repository；HTTP、鉴权或回调场景必须经过已声明入口和相应中间件，不能注入伪造的宿主身份替代验证。
- controlled 只替换声明的外部边界，不替换被测业务；live 使用实际 Core/测试服务及已有授权。逐项保存 declared_mode、actual_mode 和证据。
- 前置检查失败产生 blocked 场景结果，仍保存声明和实际诊断。SDK 不自动 grant，不切换 ProviderMode，不用本地业务库替代宿主权威数据。
- 重复事件、重试和异步等待属于场景业务操作，须有明确超时及最终读回；不提供无期限 sleep 或自动重试所有失败的通用机制。

## 3. 执行、取消与异常

- 运行体独立于发起 HTTP 请求的 context；存储 queued 后启动 worker，实例内一次一个活动批次或清理作业。忙碌返回 REGRESSION_BUSY。
- 执行前验证场景输入和 profile；同一次提交用 client_request_uuid 去重。新重跑使用新请求 UUID，并记录 rerun_of_run_uuid。
- 执行时限覆盖 Preflight/Prepare/Execute；Cleanup 使用独立 context 和 cleanup_timeout_ms。所有业务 client 和异步等待必须支持 context。
- Step 依赖失败时跳过后续步骤，已完成证据保留；独立场景可继续。预期拒绝以机器错误码断言。
- 正常 Execute 返回后逐项核对 required_assertion_keys，缺失项以 REGRESSION_ASSERTION_MISSING 判 failed 并保存 missing_assertion_keys；重复报告以 REGRESSION_ASSERTION_DUPLICATE 判 failed。只有全部必要 key 各有一个 passed 结果、证据完整且清理成功/无需清理才可通过，零断言或仅可选断言不能通过。
- 前置依赖阻塞、超时、取消、中断造成未执行时保留其原因；必要断言 blocked/skipped 或证据不可获得均为非通过。证据不可获得使用 REGRESSION_EVIDENCE_MISSING；不能把未获得事实的断言写为 passed。
- 超时、取消、panic 和 Prepare 部分失败都进入清理；不强杀 goroutine。迟迟未协作退出时保持停止中并显示原因，不提前宣布完成或资源已清理。
- 重启遗留执行为 interrupted；显式重试清理需当前场景注册仍支持该 cleanup_key，并匹配原始 profile 的环境配置标识。定义不再可用或环境已变化时返回稳定错误并保留资源信息，不猜测清理方式，也不在另一环境执行删除。

## 4. HTTP、身份与错误

- 路由相对已配置 API base：`/admin/regression`。普通响应沿用 success/data/error/timestamp/request_id，SSE 和报告下载使用其声明的 content type。
- `regression:read`：目录、profile、运行、SSE 与报告；`regression:invoke`：发起运行；`regression:manage`：取消和清理。
- tenant_uuid/actor_uuid 从入站验证身份解析，不接受请求 body 覆盖。浏览器使用现有验证宿主会话；CLI 使用显式测试身份凭据；Core 出站复用现有 STS/授权。
- 无效参数 400；未认证 401；权限不足 403；未知或其他租户对象 404；忙碌/重复提交冲突 409；缺失配置/存储 503。
- 稳定业务机器码：REGRESSION_INVALID_REQUEST、REGRESSION_NOT_FOUND、REGRESSION_BUSY、REGRESSION_REQUEST_CONFLICT、REGRESSION_DEPENDENCY_BLOCKED、REGRESSION_TIMEOUT、REGRESSION_CLEANUP_FAILED、REGRESSION_INTERRUPTED、REGRESSION_STORE_UNAVAILABLE、REGRESSION_CLEANUP_UNAVAILABLE、REGRESSION_EVENT_META_INVALID、REGRESSION_ASSERTION_MISSING、REGRESSION_ASSERTION_DUPLICATE、REGRESSION_EVIDENCE_MISSING。
- error.message 和展示原因使用 locale key；日志记录 request_id、trace_id、tenant_uuid、run_uuid、case_execution_uuid，不输出凭据。

## 5. SSE 契约

```text
id: <event_uuid>
event: powerx.regression.run.updated.v1
data: <Framework realtime JSON envelope>
```

占位表示格式语法；实际输出必须为结构化 JSON 和标准 UUID。

- 复用 realtime Envelope：protocol=sse、channel、event_type、payload、plugin_id、tenant_uuid、member_uuid、trace_id、request_id、timestamp；payload 为 data-model.md 的 RegressionEvent。
- RegressionEvent.meta 复用 `framework/backend/go/event.Meta`，必填 tenant_uuid、request_id、trace_id、source_plugin、occurred_at、payload_version；payload_version 固定 v1。构造时从可信操作 Scope/manifest 取得身份、请求和来源，记录事件发生的 UTC 时间。写入前严格验证非空、一致性和版本，错误为 REGRESSION_EVENT_META_INVALID，不从 plugin_id/timestamp 等外层别名补齐。
- meta.tenant_uuid 与事件/Run/外层 envelope 一致，meta.source_plugin 与外层 plugin_id 一致；外层 trace_id/request_id 使用原事件 meta 对应值。member_uuid 来自当前订阅者的已验证身份。外层 timestamp 为发送时间，meta.occurred_at 为事实发生时间；补读不得改写已保存 meta。
- RegressionEvent.payload 按 kind 定型：run.created/run.updated/run.finished 为 `{run}`；case.updated/cleanup.updated 为 `{run, case_execution}`；assertion.recorded 为 `{run, case_execution, step_result_uuid, assertion}`。全部携带已提交的 Run，关联 UUID、tenant_uuid 和 revision 必须一致；客户端按类型更新，不解析任意 JSON 文本推断业务状态。
- channel 由服务端生成并绑定插件、租户及 run_uuid；任意游标必须属于同租户、同运行。客户端使用 `after_event_uuid` 补读，UUID 只作游标，sequence 负责排序。
- 每个已提交结果变化对应一个已持久化事件。先登记订阅，再读取游标之后的事件与快照，缓存期间的新事件按序发送、去重；慢消费者/缓冲溢出明确断开并提示重连，不能丢弃事件后继续伪装正常。
- 终结事件输出后关闭流；已终结运行补读完成后同样结束。fetch SSE 由 Framework Realtime 管理，不提供轮询替代。
- 首次/恢复视图从 GET run 取得快照和 last_event_uuid，再订阅后续变化；客户端拒绝落后的 revision。租户、身份或宿主 session revision 变化关闭原连接。
- 复用或补充现有 SSE parser 的 id 支持和类型化校验，不能在业务页面自行解析文本日志。

## 6. 报告与 CLI

JSON 报告的根字段为 schema_version=v1、run、case_executions、events、exported_at。Run 和每个 CaseExecution 按 data-model.md 输出；快照保留 source version、定义 version、输入、profile 实际依赖、清理状态和 Trace。

JUnit 导出按场景映射 testcase：passed 正常，failed 对应 failure，blocked/skipped/cancelled/interrupted 对应 skipped 并附机器原因。JSON 保存完整证据，JUnit 仅作为自动化展示；CLI 不通过 JUnit skipped 数量推导成功。

CLI 命令约定：

```text
px-plugin regression list --api-base <plugin-api-base> --token-env REGRESSION_TEST_TOKEN --json
px-plugin regression run --api-base <plugin-api-base> --token-env REGRESSION_TEST_TOKEN --profile <profile_uuid> --case <case_uuid> --plugin-id <plugin_id> --inputs <inputs.json> --request-uuid <client_request_uuid> --wait --report <output.json>
px-plugin regression report --api-base <plugin-api-base> --token-env REGRESSION_TEST_TOKEN --run <run_uuid> --format json|junit --output <output-file>
```

- --case 可重复；每次用户 run 操作生成新的 client_request_uuid，传输重试复用该 UUID。
- --wait 使用 SSE；连接错误明确退出并输出可恢复的 run_uuid，不取消服务端运行，不增加轮询。
- `run --wait` 退出码：0 表示 passed 且清理成功/无需清理；1 表示参数、鉴权、HTTP 或本地文件操作错误；2 表示已结束但非通过；3 表示 SSE 中断。中断时保留可恢复的 run_uuid。未加 `--wait` 的退出 0 只表示提交被接受，不能证明业务通过。普通 CLI 用户文案走 locale，--json 保持机器契约。
- 对比不同发起方式时 run_uuid、事件时间等执行标识不同；业务字段、断言含义、判定和结构必须一致。

## 当前实现补充

- 工作台目录加载遇到无标准响应的 HTTP 404，使用客户端错误 `REGRESSION_API_UNAVAILABLE` 显示接口／配置检查提示；未分类请求失败使用 `REGRESSION_REQUEST_FAILED`，不误报为用户输入或身份错误。后端返回的标准 `REGRESSION_*` 原因优先保留。

- live 配置绑定 `tenant_uuid`、`metadata_resource_type` 与 `metadata_resource_uuid`；浏览器不能改变依赖目标或 fixture 归属。此资源须为专用测试对象；场景结束恢复已捕获的原标签集合。
- `writeOnly: true` 属性不进入输入快照；凭据身份及声明的敏感值从断言／证据中屏蔽，常见凭据字段被移除。
- 本地 CLI 通过退出 0、非通过退出 2、SSE 中断退出 3；网络重试由调用方复用 `--request-uuid`，不自动轮询或切换协议。
- 单实例启动没有跨租户扫描权限；恢复在首次可信租户请求时执行，仅修正该租户旧记录状态，不重放业务。
- RegressionRepository 组合 BaseRepository，使用 `set_config('app.tenant_uuid', ?, true)` 实现参数化事务租户作用域；Framework adapter 不持有裸 DB。

## 工作台业务展示契约

插件薄页面向 `PXRegressionWorkbench` 提供 `describeCase(definition)` 与 `defaultInputs(definition)`。前者返回 `RegressionCasePresentation`（导出位置为 Framework Admin Layer 的 `app/types/regression.ts`）：

```ts
interface RegressionCasePresentation {
  expectationKeys: string[];
  impactKey: string;
  fields: Array<{
    key: string;
    labelKey: string;
    control: 'text' | 'textarea';
    helpKey?: string;
  }>;
  results?: Record<string, RegressionResultPresentation>;
}

interface RegressionResultPresentation {
  shape: 'scalar' | 'record' | 'list' | 'references';
  fields: RegressionResultField[];
  itemLabelKey?: string;
}
interface RegressionResultField {
  key: string;
  labelKey: string;
  format: 'text' | 'number' | 'boolean' | 'enum' | 'numbers' | 'items' | 'reference' | 'references';
  values?: Record<string, string>;
  fields?: RegressionResultField[];
}
```

- 文案字段均为插件已注册的 i18n key；`fields.key` 与场景输入 Schema 的 string properties 一一对应。`defaultInputs` 返回业务字段默认值，字段为空时通过业务表单填写。
- 当前展示契约支持字符串文本／多行文本；复杂类型应明确扩展展示契约，不以 JSON 编辑作为隐式兜底。缺失映射明确提示并禁止开始执行。
- 工作台使用 Valibot 校验必填、长度及 UUID 格式；后端 Schema 与可信身份校验仍为权威。Framework Admin 声明 Valibot peer 依赖，Skeleton／生成模板已经提供该依赖。
- 未提供默认值的表单字段明确初始化为空值，按字段显示本地化必填提示；历史重跑带回保存的输入，未保存的字段仍需填写，不会以默认业务数据代替原输入。
- 默认用例列表与执行记录分标签页；查看说明使用抽屉，运行准备使用表单弹窗。点击开始才创建批次，返回准备不执行；接受后进入该批次结果详情。原 SDK、HTTP、SSE、报告及 UUID 契约不变。
- `results` 按机器断言 key 声明 expected/actual 的业务显示，不能修改断言判定。标量使用首个字段，record 按字段 key，list 按顺序展开对象与字段并使用 `itemLabelKey` 的 index 参数。`items` 显式声明嵌套字段，enum 的值映射到插件 locale；引用默认只显示匹配关系／数量，不直接输出 UUID。缺失映射、未知枚举或未记录字段明确提示，不伪造实际值或回退原始 JSON。
- Framework 通用渲染四个结果分区：概览、业务校验、测试数据与清理、执行与诊断。通过的断言也显示预期和实际；较大结果按需打开完整对比抽屉。原始证据、UUID 和报告在明确技术入口查看。Template 的字段、枚举和文案只在插件 `regressionPresentation.ts` 中声明。
- 桌面用左侧纵向分区导航，窄屏用可横向滚动导航；默认只展示概览。当前用例在分区切换与同批次 SSE 更新后保留，切换新批次才重置。失败位置、清理失败和连接异常独立于分区持续可见。
