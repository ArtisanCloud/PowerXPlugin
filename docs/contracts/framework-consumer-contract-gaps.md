# 消费者合同缺口与跨仓库交付任务单

日期：2026-09-08。依据当前 Framework/Core 源码，只读核对不等于安装态验收。对外入口仍为 [接入指南](../guides/features/009-consume-powerx-capability/guide.md)。最新进度：Core 已交付 Variant 与八项 Registry/Gateway 实时授权规则（见 Core docs/contracts/framework-consumer-core-delivery.md）；Framework 已接入 Variant 三项传输方法。Core 尚未 migrate/seed/重启；组织写入仍未交付。以下原始清单保留作为验收要求，不代表缺口仍全部存在。

## 1. Media preview Variant：Core 与 Framework 均有功能缺口

原缺口已在代码层补齐：runtime/media.Service 新增 PresignVariantUpload、CompleteVariantUpload、PresignVariantDownload，使用 POST /api/v1/tenant/media/assets/{asset_uuid}/variants/{variant_uuid}/{presign-upload|complete-upload|presign-download}，三项复用 com.corex.media.assets.transfer。不得以 Asset ticket 操作 Variant，不恢复旧按 variant 名称定位的接口。

### Core 交付要求

1. 在现有 tenant Media Host 中公布 Variant 上传预签名、完成上传、下载预签名三项操作。所有定位使用 asset_uuid/variant_uuid；具体 method/path/request/response/error/capability/scope 由 Core 写入正式 OpenAPI 后冻结，不能让插件猜路径。
2. 明确父子归属：variant 必须属于指定 asset 和凭证租户；跨租户、父子不匹配统一不可见。tenant 不接受 body/query/header 覆盖。
3. ticket 绑定资源 UUID、操作、有效期和吊销版本。原始文件 ticket 不能操作 Variant，反之亦然；删除资源或吊销后旧 ticket 失效。不得泄露内部 object key 或接受任意存储路径。
4. 明确 variant 上传状态与持久化字段：期望 size/mime/checksum、完成时间；complete 根据实际对象校验后才能变为可下载状态。重复 complete 的幂等与冲突行为写入合同；不能仅信任请求 checksum。
5. STS 服务身份及支持的 API Key 均检查发布、tenant registration、实际 grant；独立说明读取、创建变体、传输所需 capability。不能用 sts_direct 代替授权。
6. 交付成功、无凭证、无 grant、跨租户、父子不匹配、ticket 过期/吊销/错资源、size/mime/checksum 不一致、重复完成和依赖失败测试。提供迁移、seed、重启与真实请求证据；只在模型变化时增加迁移。

### Framework 与插件后续

- Core 冻结合同后，Framework 新增对应 DTO 与 Variant transfer contract、typed delegated 方法及工厂装配；决定扩展 Service 还是新增独立 accessor 时同步版本影响，不能让已有插件靠空方法满足编译。
- Framework 覆盖精确 method/path/body/headers、错误传播、请求取消、模式隔离测试；更新所有受影响的 Skeleton、模板和 test doubles。
- AI Craft 随后实现 local adapter 并迁移 preview 写入和读取。旧链路不宣称合规；受影响业务暂停迁移或明确不可用，不新增静默回退。

## 2. 组织写入：不是 IAM Directory 的已实现范围

`iam/contracts.DirectoryService` 是只读目录。Core 现有 members/roles provisioning 不等于完整的部门/成员组织同步合同；插件后台登录、refresh/logout 也不是 Registry 的职责。

### Core 交付要求

- 明确部门和成员 UUID-only 创建、更新、停用/删除、成员部门关联操作的输入输出；冻结 parent_department_uuid、leader_member_uuid 等引用，禁止 numeric ID 和显示名定位。
- 单独发布组织写能力与精确权限；STS 只授予明确获批插件。租户/服务主体从凭证推导，目录读 grant 不得获得写权限。
- 冻结来源 namespace、external subject 与 member/department UUID 的绑定和冲突规则；不能靠同名、邮箱或手机号自动关联既有成员。
- 写清 push/pull 的数据权威、幂等键、重试与并发冲突、部分失败、删除/停用及关联完整性。历史人工成员不能自动归属某个同步插件。
- 提供 OpenAPI/capability/grant 映射及成功、越权、跨租户、重复同步、并发冲突、失效引用、依赖失败测试。

### Framework 与 SCRM 后续

Framework 在 Core 合同冻结后新增独立组织写 contract/Factory/delegated adapter，不向只读 Directory 塞入写方法；local 由 SCRM 注入。SCRM 目前返回 ORGANIZATION_DIRECTORY_WRITE_UNAVAILABLE 是安全阻断，不是组织同步验收完成。目录读取和线索负责人 UUID 校验不受此阻断影响。

## 3. Capability / Integration：已有实现，补逐操作授权证据

以下是已存在的 Framework client 与 Core tenant 路由，不是待新建 API。路径相对 `/api/v1`；具体代码分别为 `runtime/powerx/{capability,integration}/client.go` 及 Core `openapi/{capability_registry,integration_gateway}/routes.go`。

| Framework 操作 | method/path | 授权核对要求 |
|---|---|---|
| Registry.List | GET /tenant/capabilities | 目录可见范围及当前凭证过滤 |
| Registry.GrantStatus | POST /tenant/capabilities:grant-status | 已知自身能力 com.corex.capabilities.grant_status.read；查询项不能被当成获授 |
| Registry.Resolve | GET /tenant/capabilities/resolve | 目标 capability 可见性与解析不等于执行授权 |
| Registry.Invoke | POST /tenant/invocations | 当前凭证对目标 capability 的即时 grant、输入约束 |
| Registry.GetInvocation | GET /tenant/invocations/{traceId} | trace 格式按现行合同；结果所属 tenant 与调用主体隔离 |
| Gateway.ListRoutes | GET /tenant/integration/routes | route 目录可见范围 |
| Gateway.GetRoute | GET /tenant/integration/routes/{route_slug} | route_slug 为现行协议字段，不臆造 route_uuid |
| Gateway.InvokeRoute | POST /tenant/integration/routes/{route_slug}/invoke | route/tool grant 与路由解析后的目标 capability 授权 |

Core 已交付以上八项操作的实时校验与主体隔离规则：目录分页前过滤，执行目标再次鉴权，trace 要求 caller_subject 且保留目标 grant 校验；Gateway 同时执行 tool grant。Framework 不缓存或伪造授权；这些规则不改变现有 method/path。安装态仍需逐项验证：身份类型、入口权限、目标权限、发布/registration/grant 校验执行位置、API Key scope（若支持）、稳定错误和撤权测试。若某入口不是独立 capability，明确记录其网关访问策略，不能编造 capability ID 填空。`hostcontract/catalog.go` 是诊断操作白名单，不是 Core 授权执行器，空 CapabilityID 既不能证明无授权，也不能作为已验收证明。

Framework 根据映射修正实际 DTO/transport 差异和文档，保留远端错误，不增加本地模拟 grant。SCRM 先列出现有实际调用操作和目标能力，再以 Runtime 替换重复分流；local 注入本地实现，不能把旧远程 Gateway 包装成 local。非目标能力的全部模块不必一起等待。

## 4. 顺序与完成条件

1. Framework 立即纠正指南、台账与 specs，回归已有实现的单选与错误传播。
2. Core 并行补 Media Variant、组织写合同和上述 P2 映射；不要重建已经存在的 Registry/Gateway 路由。
3. Framework 按冻结合同实现、更新模板与测试，才能将具体新操作标为 ready_for_integration。
4. 插件实现 local 并迁移已支持操作；安装态记录 STS 成功、403、跨租户、撤权和实际业务结果后才验收。

禁止将“明确不可用”“文档已更正”“httptest 通过”“make capability-check 通过”写成缺失功能已实现。不得为赶版本擅自发布协议、提交、打 tag 或更改 Core/插件数据库。

## 5. 本轮 Framework 回归证据

仓库根目录执行：

```bash
go test ./framework/backend/go/runtime/media ./framework/backend/go/runtime/powerx/media ./framework/backend/go/runtime/capability ./framework/backend/go/runtime/powerx/capability ./framework/backend/go/runtime/integration ./framework/backend/go/runtime/powerx/integration -count=1
git diff --check
```

2026-09-08 六个包通过。新增两个 Runtime → typed HTTP client 的 403 测试，断言 method/path/STS header、reason_code 传播与 local 调用次数为零；Integration 另补 local adapter 缺失时不使用 delegated 的测试。HTTP 服务为 httptest，不证明 Core grant 或真实套件消费已验收。缺失操作仍由第 1、2 节跟踪。
