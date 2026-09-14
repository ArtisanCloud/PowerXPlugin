# Cache / TaskCenter：Framework 合同与 Core 交付要求

基线：2026-09-09。Core 已交付 `docs/contracts/cache-taskcenter-host-api.md` 与 `specs/contracts/runtime-host.openapi.yaml` v1.0.0。Framework 已实现六项 delegated HostProvider、STS 租户绑定、信封与错误映射，并同步 local 校验；状态 ready_for_integration。Core 目标环境迁移/授权/部署和真实安装态验收仍后置，不猜 `/internal/tasks`。

## 已交付的 Framework 边界

| 模块 | 工厂 / accessor | 插件 local 实现 |
|---|---|---|
| runtime/cache | NewRuntime(mode, local, delegated).Cache() | Service.Get/Set/Delete |
| runtime/taskcenter | NewRuntime(mode, local, delegated).Tasks() | Service.Create/Get/Update |

工厂沿用 runtime/module，拒绝未知 mode；nil、typed-nil、零值或未提供当前模式 adapter，accessor 明确报 FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE。必须在启动时取得必需 accessor，不得注入空实现假装启动成功。

Scope.TenantUUID 来自受信任身份，不从任意请求 body 取值。这里是 Go 内部参数，不能原样当成 Core 请求 JSON。`hostapi.TokenProvider.Credential(ctx)` 必须由后端可信 STS 层返回同一凭证的 Token 与 TenantUUID；不能从未验证的请求/token 内容补造归属。Framework 先匹配 Scope，Core 再验签并实时鉴权。本版不支持 API Key、普通用户 JWT 或 on-behalf-of。

## Cache 语义

- 键身份为 tenant + namespace + key；namespace/key 非空。Core 还必须隔离插件 namespace，不能跨插件猜键。
- Value 是字节，不限定为许可证。Get 返回 Found/Value/ExpiresAt，空值命中与 miss 分开；存储错误不能返回 miss。
- Set 显式正 TTL，禁止隐含永久缓存。local 原子写值和 TTL；过期后 Get 必须 miss。Delete 不存在键成功，依赖故障仍报错。
- Framework 校验 canonical 非零 tenant UUID；namespace/key 分别 1–128/512 UTF-8 字节、无首尾空白/NUL/CR/LF；Value 最多 1 MiB；TTL 为整毫秒且 1ms–24h。delegated 使用 canonical Base64 和 ttl_ms；不静默截断 Duration。local 同样执行这些限制。
- 此模块不定义许可证是否有效。LicenseCache 将自身 DTO 编码为 bytes；授权真相不能取决于客户端缓存的 Core grant。

## TaskCenter 语义

- 创建 Type + IdempotencyKey + JSON Payload；返回持久化 TaskUUID、TenantUUID、状态、Revision 与时间。TaskUUID 必须 canonical 非零 UUID，不使用 task-123/numeric ID。
- 默认 queued，执行进入 running，结束 succeeded/failed/cancelled；succeeded 的 Progress=100，所有终态必须有 CompletedAt。创建命中幂等记录可以返回原状态，不重复创建或执行。
- Update 必须 ExpectedRevision，以存储事务执行 CAS；版本范围 1..9223372036854775807。成功返回原版本+1、请求状态和进度；达到上限后更新 conflict。type、message_key 使用机器标识，payload/result 最多 256 KiB、64 层；拒绝重复 JSON 字段、非法 UTF-8 和 NUL。省略 Payload/Result 按 JSON null 发送。local 必须同步这些约束。
- adapter 负责合法状态转换、进度不倒退、终态不可覆盖；未提供取消执行协议时 cancelled 仅是任务记录状态，不宣称停止底层业务。TaskCenter 记录与 Queue/Scheduler 执行分开。
- Framework 提供 `taskcenter.ValidateTransition(scope, current, update)`：local adapter 在自己的事务/锁内、对实际当前行调用，再执行 CAS 更新。它拒绝无归属/跨租户任务、版本冲突与溢出、进度倒退、终态覆盖、running 回 queued、queued 直接 succeeded；允许 queued 失败/取消。校验器不代替原子写入，不在 Runtime 中先 Get 再 Update。
- i18n 使用 MessageKey；JSON Result 不得泄露他人资源/凭证。数据库保存独立 tenant 与创建主体，不能把可覆盖的 metadata 当作唯一归属。
- Get 必须验证 tenant + task UUID；无归属历史任务不能自动公开。Core 还需校验调用插件/服务主体；更新不得修改其他主体任务。

## Core 已交付的验收要求（安装态仍须复核）

1. 正式 tenant Host OpenAPI：Cache 三项、TaskCenter 三项的 method/path、DTO、字节编码、TTL/大小限制、错误 envelope；未知字段与租户/主体覆盖拒绝。
2. STS service actor 身份、能力发布、tenant registration 与实时 grant；API Key 若支持则显式 scope 与凭证主体隔离。不能把“插件已安装”视为所有任务/缓存均可读写。
3. Cache 的 tenant+plugin namespace 隔离、TTL 原子操作；Task 的 UUID、tenant、caller subject、CAS revision、状态、时间、幂等关联等持久化与迁移。真实迁移不得由 Framework 代跑。
4. 稳定模块错误：invalid_argument、unauthorized、forbidden、not_found（任务）、conflict（任务）、upstream_dependency。缓存 miss 是成功结果，不等于 404/依赖故障。
5. 成功、非法输入、无认证/无 grant、跨租户/跨插件、撤权、过期缓存、空值缓存、幂等创建冲突、并发更新冲突、终态更新和依赖故障合同测试。
6. capability/grant 映射和部署说明；路由冻结后 Framework 才实现 typed STS HostProvider。不能绕过正式授权借用管理端 Redis/Task API。

## 电商现在可以做什么

实现 Cache Service 的 Memory/Redis adapter 和 TaskCenter Service 的 local store；启动单选，业务不再根据错误回退。将 LicenseCache、TaskSubmitter/TaskReporter/StatusProvider 接到已选 Service。local 任务需要 UUID、独立租户归属和 revision，不是仅改接口签名。

电商可替换“Framework 永远缺失”的硬编码门禁：按配置调用 `cache.NewHostProvider` / `taskcenter.NewHostProvider` 后注入各自 NewRuntime。必须仍检查配置、可信凭证、必需 accessor 与实际 grant；不能无条件解除启动检查。许可证缓存是否为整个插件的必需依赖由业务明确决定。

正式映射：Cache GET/PUT/DELETE `/api/v1/tenant/runtime/cache/entries`，分别使用 `com.corex.runtime.cache.read/manage/manage`；Task POST `/api/v1/tenant/runtime/tasks` 与 GET/PATCH `/api/v1/tenant/runtime/tasks/{task_uuid}`，分别使用 `com.corex.runtime.taskcenter.manage/read/manage`。不新增独立 admin 或 internal 路径。

部署使用 Core 目标配置：迁移、capability-seed、插件授权流程、部署重启；capability-seed 不自动授予插件 grant，不执行全量 seed 扩权。四项能力按实际读写需要申请。HTTP 错误先声明 `var upstream *hostapi.HTTPError`，再使用 `errors.As(err, &upstream)` 获取 StatusCode/ReasonCode/RequestID；不解析 message，不记录凭证或原始 body。

## 验证与状态

```bash
go test -race ./framework/backend/go/runtime/hostapi ./framework/backend/go/runtime/cache ./framework/backend/go/runtime/taskcenter -count=1
```

测试覆盖六项精确 method/path/body、STS 租户不匹配、二进制/空值/miss、错误状态及 request_id、禁止重定向、取消、非法输入和 Runtime 单选；HTTP 使用 httptest，不证明真实 TTL/CAS 或 Core STS 验签。2026-09-09 定向竞态与 Framework 全量测试通过。正式 delegated transport 已实现，真实安装态验收尚未执行。
