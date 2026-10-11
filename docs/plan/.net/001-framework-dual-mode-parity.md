# 001 - PowerXPlugin .NET Framework 双模式业务模块对齐计划

状态：进行中；N001–N404 已按各任务范围完成源码与定向测试，模块整体覆盖及 Core 安装验收仍需逐项核对。  
实施根目录：framework/backend/dotnet/src/PowerXPlugin.Framework  
Go 对照基线：framework/backend/go/runtime  
权威消费规范：docs/guides/features/009-consume-powerx-capability/guide.md

## 1. 目标与完成定义

目标是让 .NET Framework 对每个公开业务模块提供与 Go Framework 一致的双模式边界：

1. 稳定 UUID DTO、业务接口与机器错误码；
2. Local adapter 与 Delegated typed client；
3. 启动期按可信 ProviderMode 单选的 Runtime 或 Factory；
4. 业务代码只消费 Runtime 返回的接口，不能在请求中切换模式；
5. local 和 delegated 均不能静默回退到另一模式；
6. 每个模块有 local、delegated、缺 adapter、403、跨租户和依赖失败测试。

完成状态分为：

| 状态 | 完成条件 |
|---|---|
| not_started | 没有 .NET contract |
| contract_only | 仅接口与 DTO |
| partial | local、delegated、Factory 或测试至少缺一项 |
| ready_for_integration | 两端实现与定向测试齐全，等待真实 Core 安装验收 |
| installed_verified | 已验证安装、grant、成功、403 或撤权、跨租户行为 |

不得以包存在、接口存在或内存 mock 存在作为完成依据。

## 2. 职责边界

### Framework 负责

- DTO、接口、输入校验、稳定错误码和 delegated 错误映射；
- LocalXxxAdapter：由 Framework 编写，依赖注入的 ILocalXxxStore 承担真实存储；
- PowerXXxxClient：只使用可信服务凭证调用正式 Core Host Contract；
- AddPowerXXxxRuntime：启动期单选 local 或 delegated；
- 测试 doubles、reference store、Factory 与合同测试。

### 插件负责

- ILocalXxxStore 的数据库实现、事务、审计和领域规则；
- tenant、主体和业务对象 UUID 的可信校验；
- Bootstrap 中唯一的 ProviderMode 和 local store 注入；
- 自身的本地持久化及端到端测试。

Framework 不得猜测 CRM、SCRM、E-Commerce 等插件的数据表。对于插件特定的持久化，
Framework 提供 local adapter 代码与 store contract，而不是提供假生产数据库或以
内存实现冒充本地业务。

### Core 前置条件

delegated adapter 只对接冻结的 Core Host Contract：UUID DTO、操作 grant、租户与
服务主体边界、稳定错误和真实 transport。没有正式合同的模块只可实施 local
contract；delegated 必须返回明确的 FRAMEWORK_MODULE_UNAVAILABLE。

## 3. 当前盘点和目标范围

| 模块 | Go 对照 | .NET 当前状态 | 目标 | 优先级 |
|---|---|---|---|---|
| IAM | iam/contracts 与 iam/adapters | Registry、delegated；消费者 local adapter | 统一基座和 test kit | P0 |
| Media | runtime/media | 同一 Runtime 包含文件 local 与 delegated | 拆为接口、两个 adapter 和 Factory | P0 |
| Scheduler | runtime/scheduler | local 与 delegated 已有 | 迁移统一基座并补对偶测试 | P0 |
| Event Fabric | EventBridge 与 Host Event Fabric | local bridge 与 delegated subscriber 分离 | 统一事件 Runtime 与 local subscriber | P0 |
| TaskQueue | runtime/taskqueue | in-memory local 与 delegated | 持久化 store contract 和合同测试 | P1 |
| Capability | runtime/capability | 仅 grant-status client | 完整 Registry Runtime | P1 |
| Integration | runtime/integration | interface、Factory、delegated | local adapter 与合同测试 | P1 |
| Cache | runtime/cache | 缺失 | 双模式 Runtime | P1 |
| TaskCenter | runtime/taskcenter | 缺失 | 双模式 Runtime | P1 |
| Knowledge | runtime/knowledge | 缺失 | 双模式 Runtime | P2 |
| Metadata | runtime/metadata | 缺失 | 双模式 Runtime | P2 |
| Notifications | runtime/notifications | 缺失 | 双模式 Runtime | P2 |
| Skills | runtime/skills | 缺失 | 双模式 Runtime | P2 |
| Customer | runtime/customerfw | 缺失 | Auth、ExternalIdentity、Membership Runtime | P3 |
| Plugin Runtime | runtime/pluginruntime | 缺失 | 双模式 Runtime | P3 |
| Plugin Release | runtime/pluginrelease | 缺失 | 双模式 Runtime | P3 |
| Agent and Session | runtime/agent | 缺失 | 双 accessor Runtime | P4 |
| AI | runtime/ai | 缺失 | Generative Runtime | P4 |

不在本计划范围：Realtime、TaskBus、WSBus、SSEBus、Core Admin clients 和旧 generic
Gateway compatibility API。它们不可作为任何业务模块的 fallback。

## 4. P0 前置：统一双模式 Runtime 基座

新增 Runtime/Common，作为所有模块唯一的模式装配方式：

| 组件 | 职责 |
|---|---|
| ProviderMode | local 或 delegated，只在启动期解析一次 |
| DualModeRuntime of TService | 保存唯一已选择的业务接口 |
| RuntimeRegistration | DI 注册、选中 adapter 校验和 accessor 获取 |
| FrameworkAdapterException | 模块、操作、错误码与 trace 的稳定异常 |
| IServiceCredentialProvider | delegated 唯一服务凭证来源 |
| ILocalXxxStore | 插件注入的 local 持久化边界 |

所有 Runtime 必须满足：

- 只构造选中的 adapter；未选中 adapter 的调用次数必须为零；
- 必需 adapter 缺失时启动失败；可选 adapter 调用时明确失败；
- delegated 永不使用入站 user bearer；
- tenant UUID 从可信上下文取得，方法参数只做一致性验证；
- 生产 Runtime 不在业务调用时读取环境变量决定模式。

Media、IAM、Scheduler、TaskQueue、Integration 和 Event Fabric 必须优先收敛到该基座。

## 5. 分阶段任务

### Phase 0 - 基座和已有模块收敛

- [x] N001 建立 Runtime/Common：ProviderMode、DualModeRuntime、统一 DI、服务凭证
  provider、错误模型和测试 helpers。
- [x] N002 重构 Media 为 IMediaService、LocalMediaAdapter、PowerXMediaClient、
  MediaRuntime；移除 Runtime 内部环境模式判断。
- [x] N003 统一 EventBridge 与 Event Fabric 为 IEventRuntime；local 使用
  LocalEventBridge，delegated 使用 Core gRPC；两端均提供 delivery、订阅、Ack 和
  Nack 语义。
- [x] N004 将 Scheduler、IAM、TaskQueue、Integration 接入统一 RuntimeRegistration。
- [x] N005 为上述模块补选中 adapter、未选中 adapter 零调用、无配置、401、403、
  tenant mismatch、依赖失败测试。

完成门槛：CRM local 模式不需要 Core URL 或服务凭证；delegated 模式不启动 local
runner，也不访问 local store。

### Phase 1 - 基础运行时

- [x] N101 完整实现 Capability Registry：List、GrantStatus、Resolve、Invoke、
  GetInvocation 的 DTO、local adapter、delegated client、Runtime 与测试。
- [x] N102 完整实现 Integration：route ownership、local gateway adapter、现有
  delegated client、Runtime 与授权错误测试。
- [x] N103 实现 Cache：ICacheService、ILocalCacheStore、LocalCacheAdapter、Core
  client、Runtime；覆盖 TTL、1 MiB 限制和 tenant 隔离。
- [x] N104 实现 TaskCenter：revision CAS、单调进度、状态机、terminal 保护、
  ILocalTaskCenterStore、Core client、Runtime。
- [x] N105 将 TaskQueue 内存实现降为 reference store，增加
  ILocalTaskQueueStore；不得用 Queue 替代 TaskCenter。

完成门槛：Capability、Integration、Cache、TaskCenter、TaskQueue 都有两端 adapter、
Factory 和定向测试，不存在 raw HTTP consumer。

### Phase 2 - 内容、治理、通知

- [x] N201 实现 Knowledge：Space、Search、Document、IndexJob DTO；local store、
  delegated client、Capabilities 和 Runtime。
- [x] N202 实现 Metadata：Dictionary、Taxonomy、Tag、TagBinding、ResourceType
  UUID DTO；分页与 Resolve；local store、delegated client、Runtime。
- [x] N203 实现 Notifications：Publisher、local outbox store、delegated create
  client、Runtime、幂等和 member UUID tenant 边界测试。
- [x] N204 实现 Skills：Invoker、local authorization registry、delegated client、
  Runtime；禁止自由 skill 执行。

完成门槛：两个模式返回同一 DTO 和错误分类；local 写操作有真实 tenant 隔离与审计。

### Phase 3 - 身份和插件平台

- [x] N301 实现 Customer Runtime：Auth、ExternalIdentity、Membership 三组接口、
  ILocalCustomerStore、delegated 双凭证 client 和 Runtime。
- [x] N302 实现 Plugin Runtime：知识空间、Agent 实例化、Agent 列表的 UUID DTO、
  local service、delegated client 和 Runtime。
- [x] N303 实现 Plugin Release：install session、import job DTO、local release
  store、delegated client 和 Runtime。
- [x] N304 补 customer JWT、STS、membership tenant、release trust 的安全测试。

完成门槛：delegated 操作不得接受请求覆盖 tenant/customer UUID，也不得读取插件本地
权威数据作为 fallback。

### Phase 4 - 高状态复杂度

- [x] N401 实现 Agent Runtime：Lifecycle、Session、会话、消息、invocation UUID DTO、
  ILocalAgentStore、协作式取消和每会话单活跃执行。
- [x] N402 实现 Agent delegated client：十二项 session 操作、SSE state/final/error/end、
  重连和幂等键测试。
  - 已实现固定 Host Contract 的十二项操作、服务凭证、幂等键和 `state/final/error/end`
    校验；EOF 会以中断失败，绝不伪报成功。
  - Core、Go 与 .NET 均已采用 `id: 1/2/3` 和 `Last-Event-ID`；重连只重放未确认的
    state/terminal/end 帧，超过有限次数或 EOF 无 `end` 明确失败。
- [x] N403 实现 AI Runtime：GenerativeService、模型、LLM、stream/session、
  embedding、VLM、image、video、TTS DTO、local provider 与 delegated client。
- [x] N404 统一 Agent/AI 的取消、超时、异常清理、tenant isolation 和 test doubles。
  - transport 与 SSE 均透传调用方取消/`HttpClient.Timeout` 产生的取消异常；请求、响应和
    stream 使用确定性释放。测试覆盖取消不被映射为 upstream、Agent/AI 跨租户在发包前拒绝。

完成门槛：local 不用空响应伪造生产实现；delegated stream 不以 EOF 伪报成功，也不
回退 local provider。

## 6. 每个模块的强制交付物

每个任务完成前必须同时具有：

1. Runtime/Module/Contracts：接口、DTO、枚举和错误码；
2. Runtime/Module/Local：Framework local adapter 和显式 ILocalModuleStore；
3. Runtime/Module/Delegated：typed Core transport；
4. Runtime/Module/ModuleRuntime：启动期单选 Factory；
5. Framework Tests：local、delegated、adapter 缺失、禁止 fallback、tenant mismatch、
   403；
6. 与对应 Go API 的方法、DTO、错误码、测试名称对照表；
7. 更新 docs/contracts/powerx-core-framework-coverage.md 的状态和证据。

## 7. CRM 消费者迁移顺序

1. Phase 0 完成后迁移 CRM 已使用的 Media、IAM、Scheduler 和 Event Fabric；
2. 先用 local 模式验证附件、主体目录、公海回收和事件消费；
3. 再用 delegated 模式验证不会读取 CRM local 表；
4. 安装 Core Event Fabric ACL 后验证 Subscribe、Ack、Nack、重连、撤权和跨租户；
5. E-Commerce、合同签署不能猜测 Host Contract，继续 fail-closed，待 Core 发布
   UUID DTO、operation grant 和 immutable contract 后独立立项。

## 8. 实施闸门

| 闸门 | 不满足时的处理 |
|---|---|
| Go API 与 .NET 设计不一致 | 停止，实现前先更新对照表 |
| Core 没有正式 Host Contract | 只实现 local contract，delegated 明确不可用 |
| local 没有持久化 store contract | 不允许以 in-memory mock 当生产实现 |
| delegated 使用入站 bearer 或 tenant 可覆盖 | 阻断合并 |
| 未选中 adapter 零调用测试缺失 | 不得标 ready_for_integration |
| 安装态验收缺失 | 保持 ready_for_integration |

## 9. 建议下一步

2026-10-04 Go Skeleton 三个调试入口对齐：新增 .NET `HostContractProbe`，仅解析
启动时已选择的 typed runtime，返回真实装配状态并保持 `connectivity_verified=false`。
Capability Registry 客户端按 Core 正式 snake_case 编解码，新增安全目录发现接口，
与有效 grant、正式协议合同分开；拒绝非法/缺失响应，不返回部分成功。新增 typed
`PowerXWSBusClient`，服务凭证调用 Core grant → publish 和通知测试合同，无本地回退，
不伪造宿主 echo。CRM 消费入口、WS 验证和 local Scheduler → Event Fabric → WS
诊断事件桥接已落地；未绑定的九个业务模块及生产通知 outbox 仍未完成。
真实 Core 安全目录/正式合同只读检查成功；三个页面运行态验收仍待用户重启 CRM。

逐项核对第 6 节的强制交付物，并更新 Core/Framework 覆盖台账。Agent Session 的
N402 transport 与 SSE 恢复、Agent Lifecycle 六项 .NET delegated typed client 已完成
定向测试；生产 local store、实际插件消费和 Core 安装态 grant/撤权及跨租户联调仍缺，
18 项 Agent 操作整体保持 `partial`。CRM 按第 7 节
验收已消费的 Media、IAM、Scheduler 和 Event Fabric；Core 安装、grant、撤权和
跨租户的真实结果单独记录，未执行时保持 `ready_for_integration` 或 `partial`。

2026-10-02：Event Fabric .NET delegated subscriber 已增加定向 gRPC 客户端合同测试，
覆盖 Subscribe、Ack/Nack、回调失败、403/401/依赖错误和发包前 tenant 检查；CRM 的
正常 EOF 重连加入固定间隔。Core 已安装环境中的 grant、撤权和跨租户传输仍须单独验收。

同日补齐 local 租户边界：`LocalEventFabricAdapter` 在序列化前按元数据 tenant
筛选订阅；CRM local 消费改用 `IEventRuntime`，并核对 payload tenant 与启动时绑定的
可信租户，delegated 消费者也不再随配置变化重绑 tenant。`LocalScheduler` 按 tenant
与 job_id 寻址，单任务读取/更新/暂停/恢复/触发要求有效 tenant UUID；不同租户的
同名任务相互独立。对应测试覆盖跨租户零业务写入、订阅隔离、同名任务和非法租户。

2026-10-02 IAM 消费者收尾：CRM 负责人、审计及合同成员消费已使用 Registry，
保留内部数字键并显式关联 UUID；delegated 映射由部署文件提供，测试以已释放的
DbContext 证明不读 local IAM。`.NET PowerXIamClient.ListMembers` 已按 Core 正式
`items/pagination` 遍历全部分页，校验页号、总数、条数和重复成员；后续页 403 或
非法分页明确失败，不返回部分目录。CRM 本机历史 UUID 经备份后独立修复 9 行，
修复后六类缺失/非法/重复均为 0。完整测试 Framework 92/92、CRM 95/95。
操作记录见 CRM `docs/plan/08-system/mechanisms/iam-framework-consumers.md`；
真实 Core 安装态 Scheduler/Event Fabric、IAM grant/撤权和页面验收仍未完成。

2026-10-03 当前 CRM 交付目标确认为 local + POWERX_PROXY=1；Scheduler/IAM/Event
Fabric 按 provider mode 保持 local，安装态验收后置。Local Scheduler runner 增加
每任务失败隔离与稳定 due-time 幂等重试；once 成功后 completed，once 时间表达式
规范为 UTC，trigger_source 使用实际 schedule_type。Framework runner 测试覆盖
一次性执行、时区、失败后其他租户继续、稳定重试键和取消；CRM 联接测试覆盖
local runner → Event Fabric → 回收服务，两个 delegated factory 零调用。模拟 Core
RPC 的 CRM/Framework 联接测试还覆盖 Ack 丢失重连与重复事件；不算真实安装态证据。

2026-10-03 Local Scheduler 持久化：新增 `ILocalSchedulerStore`，以 detached Job 快照
和 revision CAS 保存任务；runner 派发后持久化完成状态，失败保留到期时间和重试键。
`AddPowerXScheduler` 可显式注入 local store factory，CRM 使用 PostgreSQL store；
默认无 factory 的内存 store 保留开发/测试用途，不计生产持久化。CRM 重建 DI 和
真实 PostgreSQL 临时 schema 测试覆盖恢复、暂停/完成、payload、重复创建和 CAS。
未运行用户服务重启或业务 schema 迁移，多副本调度租约后置。
