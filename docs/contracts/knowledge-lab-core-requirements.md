# Knowledge Lab 对 PowerX Core 的补齐要求

本次修复页面：`http://127.0.0.1:3131/powerx/knowledge-lab`。
界面参照：`http://127.0.0.1:3131/_p/com.powerx.plugins.base/admin/knowledge`。
仅修改 PowerXPlugin；本文是向 Core 提出的要求，不代表 Core 已完成或已授权这些能力。

## 页面与操作入口

Lab 空间列表与本地知识库共用 `KnowledgeSpaceOverviewPanel`，保持空间名称、部门、状态、更新时间和操作列一致。两者的数据适配独立：本地仍调用 Local API，Lab 只调用 PowerX 代理 API。

- `/powerx/knowledge-lab`：空间列表，不在列表底部堆叠检索表单与结果。
- `/powerx/knowledge-lab/create`：与本地共用创建表单；目录缺失时保留字段、显示对应错误并禁用提交。
- `/powerx/knowledge-lab/{space_uuid}/ingestion`：入库表单；只有明确提交才上传或创建任务。
- `/powerx/knowledge-lab/{space_uuid}/sources`：数据源连接；当前明确展示委托接口缺口。
- `/powerx/knowledge-lab/{space_uuid}/records`：入库任务记录；文档/分块管理仍待 Core 合同。
- `/powerx/knowledge-lab/{space_uuid}/strategy`：策略信息；修改/保存仍待 Core 合同。
- `/powerx/knowledge-lab/{space_uuid}/playground`：检索表单与内联结果。

以上入口也提供 `/_p/com.powerx.plugins.base/admin` 前缀版本，操作后保持当前入口前缀。技术诊断默认折叠；缺少部门、更新时间等字段显示 `—`，不伪造元数据。指定空间不可访问时明确报错，不改选其他空间。

Core 已有 `POST /api/v1/admin/knowledge-spaces`。创建受阻是当前委托客户端缺少场景与策略目录合同，不代表 Core 没有创建能力。这里只提出 Core 补齐要求，不直接修改 Core。

## 已使用的边界

- Skeleton 保持 Local，Lab 使用独立 `/api/v1/admin/runtime/knowledge-lab/*` 入口。
- Local + proxy 使用服务端 Gateway API Key；实际 Delegated 使用已注入的 STS Knowledge 客户端。浏览器本地登录 token 不传给 Core。
- 空间目录与检索使用已有 `/api/v1/tenant/knowledge/spaces`、`/api/v1/tenant/knowledge/search`。
- 已有入库、策略、归档等动作继续使用原有固定 Gateway capability 适配。不得接受浏览器传入的任意 endpoint 或 capability ID。
- 无凭证、无授权、接口未开放时明确失败，不返回本地 fixture 或伪造成功。

## 需要 Core 提供的合同

| 功能 | 当前证据与缺口 | 要求 |
| --- | --- | --- |
| 场景与策略目录 | Framework 的现有 tenant Knowledge 客户端未声明 catalog；空间创建表单需要该目录 | 发布 typed catalog 合同，返回场景、策略、Profile 映射、依赖及可用性；配套 capability、scope、API-key 和 STS 授权。未交付前 Lab 禁用创建，并独立展示原因 |
| 空间管理元数据 | 当前 tenant 空间列表仅返回 `space_uuid/name/status` | 返回部门业务名称与引用、Profile、更新时间及允许的管理操作；明确创建、归档、删除的 tenant 合同与权限。名称作展示，UUID 作引用 |
| 数据源连接 | Core admin 已存在 sources/connectors/sync-jobs 路由；这不等于统一 Knowledge Host 合同已提供相同能力 | 明确并发布插件可调用的数据源、连接器、同步任务合同及授权；返回可读名称、任务 UUID、状态和错误码 |
| 入库记录与分块 | 原 Lab 使用 admin Gateway 入库任务列表；管理页还支持文档记录、详情与分块编辑 | 明确文档 UUID、job UUID、chunk UUID 的关系，提供分页/详情/分块读取与编辑合同；不得用 numeric id 作跨边界引用 |
| 实时进度 | 本地知识库 WS topic 属于插件本地任务，不能冒充 Core 任务进度 | 提供受租户授权的 WS/SSE 订阅合同、事件 cursor、状态/进度/错误码及断线恢复语义；不以轮询静默兜底 |
| 错误与追踪 | 必须区分无权、接口缺失和业务失败 | 稳定错误码、request/trace ID；401/403/404/409/5xx 保持可区分。列表错误不得呈现为空列表或成功 |

## Core 交付验收

1. Local + proxy API Key 与真实 Delegated STS 分别验证，两者的租户都由凭证确定。
2. 已授权请求成功；缺少显式 grant 返回 403，跨租户对象不可读写。
3. 创建 → 入库 → 真实任务终态 → 文档/分块 → 检索引用形成可重复闭环；接受任务不等于完成。
4. 目录不可用不影响已存在空间的列表和检索；实时事件与持久化任务状态一致。
5. 提供最终 OpenAPI/DTO、能力标识、scope、API-key 授权方式、事件契约和可复现请求；Framework 按最终合同接入，不猜测或新增旧格式兼容。

## 2026-09-26 本地实测证据

- 原业务 `/admin/runtime/knowledge/provider`：200，`mode=local`。
- Lab `/admin/runtime/knowledge-lab/provider`：200，`mode=delegated`。
- Lab 空间列表：200，返回 3 个 PowerX 空间；检索：200，本次测试结果 0 条，不代表已有命中文档验收。
- 场景/策略目录：`KNOWLEDGE_UNSUPPORTED_CAPABILITY`。页面据能力声明禁用创建，并单独提示。
- 入库记录：403，`KNOWLEDGE_FORBIDDEN`，Core Trace `bdcce86b-06bc-4102-b1b1-28663e55c3e7`。
- 融合策略：403，`KNOWLEDGE_FORBIDDEN`，Core Trace `e744d19d-319c-49fd-a9f9-cdd820e99cd9`。
- Core 的 HTTP 与审计日志对上述两个 Trace 均记录 `/api/v1/tenant/invocations`、403、`DENIED`。请 Core 核查对应凭证的发布范围及显式授权，返回拒绝原因；不能仅凭此推断需要改业务实现。
- 需核查的能力：`com.corex.rest.admin.gin.get_api_v1_admin_knowledge_spaces_spaceid_ingestion_jobs`、`com.corex.rest.admin.gin.get_api_v1_admin_knowledge_spaces_spaceid_fusion_strategies`。
- 插件侧已移除上述请求的旧顶层 `action` 字段，并保留真实 HTTP 状态与 Trace ID。
- 未执行真实新建、上传、入库、归档或删除；这些写入链路仍需授权到位后的业务验收。浏览器工具连接不可用，未完成视觉验收。

## 2026-10-03 浏览器实测补充

- CUA `getState()` 在浏览器清单阶段报 `nodeRepl.fetch request failed`；重置后仍失败。此错误发生在页面访问前，工具未提供更底层的连接拒绝或超时原因。3131 页面返回 HTTP 200，8078 后端正常监听。
- 改用可用的 Chrome DevTools 工具，完成本地管理员登录、Lab 与本地知识库列表比对、页面截图及真实请求检查。
- Lab 空间列表返回 3 个 Core 空间；本地参考列表返回插件本地空间，两条数据链路独立。
- 在“插件联调知识空间”执行查询“知识库”：代理检索 HTTP 200、0 条命中。只证明此次请求可用，不代表已有知识内容命中验收。
- 入库记录请求 HTTP 403、`KNOWLEDGE_FORBIDDEN`，Trace `2ea66c9f-29f0-4b5a-b035-727f635072a6`。
- 策略请求 HTTP 403、`KNOWLEDGE_FORBIDDEN`，Trace `ffffad4a-e08a-4666-9029-bc5187fcbe7f`。需 Core 核查对应能力的显式授权；未修改 Core。
- 创建入口正确显示目录合同缺口并禁用提交；入库入口展示可编辑表单，未提交上传或入库任务。
- 浏览器发现并修正操作列横向挤压、创建标题错误显示已选空间、请求失败时仍显示“0 条记录”等展示问题。修正后列表截图确认操作换行、名称可读、诊断默认折叠；策略失败不再显示记录数量。
- 本轮未新建、上传、入库、归档或删除业务对象。完整写入闭环仍待 Core 目录及授权合同到位后验收。


## 创建表单对齐补充

Local 与 Lab 创建页共用 `KnowledgeSpaceCreatePanel`，均展示名称、部门、策略包和 Profile 自动映射。Lab 不再把目录缺口作为整张表单的显隐条件，也不再展示单独的场景选择器；目录内的场景/Profile 映射仍属于 Core 合同。目录缺失时名称可编辑，部门与策略选择禁用，字段旁解释原因；提交事件也验证依赖，取消不产生创建请求。Lab 不读取本地部门/策略目录，不从已有空间的部门字段推算选项。

此前把部门不可用归因为 Core 没有目录接口，表述不准确。Core 已发布 `/api/v1/tenant/iam/departments`，Framework 已有 typed `ListDepartments`；实际问题是 Lab 未接入。此次新增插件 `/api/v1/admin/runtime/knowledge-lab/departments`，Local+proxy 使用服务端 API Key，实际 Delegated 使用委托 IAM Directory；请求不使用 Local 租户覆盖 Core 租户。新跨服务部门引用须用 UUID；不得从插件 Local 组织取值，也不得以部门名称、numeric id 或空间列表字段代替目录。策略目录须同时提供策略名称、Profile 映射、依赖与可用性。策略目录与创建部门 UUID 合同尚未交付前，当前 Lab 创建保持明确不可提交；本次未修改 Core 或伪造目录响应。

创建表单浏览器复验：Local 与 Lab 均显示基本信息、策略包与自动映射；Local 部门下拉能展开真实组织选项。Lab 缺少目录时名称仍可输入，部门/策略选择及提交禁用，取消正确返回 Lab 列表；本轮浏览器网络记录无创建 POST。刷新后文案正确显示，两个创建页控制台均无错误。35 项前端测试、组件编译和两套模板同步检查通过；未执行真实创建或向量索引激活。

## 2026-10-04 部门目录接入修复

- 删除部门“缺少接口”的固定提示，创建页实际请求代理部门路由，显示上游状态码及 Trace ID；提供手动刷新。名称作展示，`department_uuid` 作选项值。空目录、请求失败和格式错误分别处理，不读 Local 目录。
- 使用当前配置的服务端凭证，直接只读验证 Core `/api/v1/tenant/iam/departments`：HTTP 200，26 个部门，每项包含部门 UUID 与租户 UUID；Trace `ae705a34-28aa-4cc6-b855-b5e5435cb5a7`。无需 Core 新增部门读取接口。
- 策略目录仍缺少公开可调用的 Knowledge catalog 合同，当前适配器返回 `KNOWLEDGE_UNSUPPORTED_CAPABILITY`。页面展示实际请求结果，不复制 Core 前端静态目录或 Local 策略。
- Core 当前 admin 创建 DTO 接收 `departmentCode`，尚无部门 UUID 字段。要求 Core 发布租户授权的 typed 创建合同，以 `department_uuid` 引用部门，并明确策略/Profile 字段及能力授权。Lab 创建端点暂返回 501；不会把部门 UUID 塞入 code 字段，也不创建伪造空间。
- 8078 原运行进程尚未加载新路由时，浏览器部门请求返回 404。经用户许可重启插件后端后，浏览器 `/knowledge-lab/departments` 返回 200；下拉展示 26 个 PowerX 部门，选中“研发部”后正常显示名称，原部门错误消失。Core 未重启或修改。
- 浏览器目录请求返回 400、`KNOWLEDGE_UNSUPPORTED_CAPABILITY`，对应明确显示目录合同缺口；创建按钮禁用，无创建 POST。控制台仅记录该目录请求的 HTTP 400，无组件运行异常。
- Go IAM 与 runtime_ops 测试、35 项 Nuxt 单元测试、5 个共享组件编译及两套模板同步检查通过。

## Core 下一步最小交付边界（待 Core 确认合同）

以下是需求，不是已发布的路由或 DTO；插件不会按猜测的 endpoint 发请求。

| 交付项 | 必须明确的字段与语义 | 验收条件 |
| --- | --- | --- |
| 策略目录 | 策略稳定引用及显示名称、对应场景、推荐 Profile、依赖、可用状态与不可用原因；若策略或 Profile 是业务对象，引用使用 UUID | API Key 与 STS 均有可调用的 typed 入口；未授权 403；空目录与未开放可区分；插件不复制前端静态目录 |
| 创建合同 | 空间名称、`department_uuid`、策略/Profile 引用、配额与策略模板 UUID；租户由凭证确定，禁止客户端覆盖 | 拒绝无权/跨租户部门与无效策略；返回 `space_uuid` 和状态；创建空间能从同一凭证的空间目录读取 |
| 权限交付 | 最终路由、OpenAPI、能力标识、scope、API Key 显式 grant 与 STS 的授权方式 | 提供真实已授权成功请求及无权请求；不把 admin 接口存在视作插件委托已授权 |

Core 现有 admin 创建 DTO 还要求 `policyTemplateVersionId` 和 `quotas`；必须明确插件通过目录如何取得策略模板 UUID、配额约束及 Profile 映射。不能只增加一个部门字段后就认为整个创建闭环可用了。

插件侧继续修复：创建页不再预填 `H_fusion`、`product_specs` 或 `p1_general`；缺少目录时策略、Profile 与 feature flags 保持空值。刷新保留仍在 Core 目录中的已选部门，部门移除或请求失败则清空选择并明确显示错误。策略/部门错误通过响应式 i18n 渲染，切换语言不产生额外目录请求。创建 UUID 合同限制独立显示，不被策略错误遮住。39 项前端测试、组件编译及模板同步检查通过；浏览器确认刷新保留“研发部”、切换语言立即更新错误。未执行真实创建，未修改 Core。

## Core 合同交付与 Framework 接入更新（2026-10-04）

Core 已交付 `docs/contracts/knowledge-host-provisioning.md` 与 Host provisioning OpenAPI。上文“Core 未发布目录/部门 UUID 创建合同”是修复时的历史状态，不再作为当前代码合同结论。正式 tenant 接口为 `GET /api/v1/tenant/knowledge/catalog` 和 `POST /api/v1/tenant/knowledge/spaces`，能力分别为 `com.corex.knowledge.catalog.read`、`com.corex.knowledge.space.create`。

Framework Go 和 .NET 已接入这两项 typed REST 合同，声明 catalog/create；保留 Profile、模板 UUID、配额与策略可用性，使用服务凭证且不接受 tenant/actor 覆盖，保留原上游错误和 Trace ID。说明见 Framework Knowledge README。

本次范围为 Framework，不修改 Core、不授予运行凭证，也未解除 Skeleton Lab 当前创建 guard。后续插件适配必须读取 available、unavailable_reasons、Profile 映射和模板/配额合同，调用 Framework typed create，并使用原 HTTP 状态和 Trace ID。不得继续调用旧 admin camelCase 创建 DTO。

Core 文档的运行状态仍需区分：能力 seed 不等于 grant；其记录的真实 Key 两项为 not_granted，迁移/重启/真实 STS 创建尚待验收。此项由运行环境负责人完成，Framework 合同测试不能替代。

## Skeleton Lab typed 创建接入（2026-10-04）

Lab 已移除固定 501/创建禁用逻辑，后端严格解析 Framework `CreateSpaceInput`，拒绝 tenant/actor、旧 `departmentCode` 和任意代理字段；调用 `SpaceProvisioningProvider.CreateSpace`，成功返回 `data.item`。目录/创建错误使用 Framework 原 HTTP 状态、machine code 和 Trace ID。

创建页按 Core `available`、Profile UUID 映射、真实部门和模板选项决定是否允许提交；策略不可用显示 Core 原因；模板显示名称/版本，UUID 仅作为 value。未编辑配额则整体省略，使用 Core 默认配额；创建成功返回 Lab 列表。新增 required 能力声明 catalog.read/space.create，不代表运行凭证已授权。

当前浏览器实测：部门 200；目录代理 404，Trace `1a91a413-7dae-4dc1-9385-245495a158fb`。直接只读请求 `http://127.0.0.1:8077/api/v1/tenant/knowledge/catalog` 同样 404，Trace `14bf892c-5e14-465a-b278-9e361f6755ae`；源码已注册该路由，运行 Core PID 54505 启动时间为 2026-10-03 22:56:56。证据指向当前运行实例尚未加载新路由，不能据此归因于 Framework 不支持或缺少 grant。

需要 Core 运行负责人按 provisioning 文档完成标准迁移、加载最新代码，并核查 Key 两项精确 grant/租户能力版本；本轮未操作 Core 进程、数据库或授权。之后复验 catalog 200、available 策略、真实 create 和空间目录复读；此前不宣称真实创建已通过。

## 2026-10-04：共享 Local 组件与 PowerX adapter 的实际边界

页面入口仍为 `http://127.0.0.1:3131/powerx/knowledge-lab`。创建和总览共用 Panel；入库、记录、策略、来源、检索直接复用 Local 业务组件，通过 `knowledgeWorkspaceKey` 注入 PowerX adapter。Local 页面默认使用 Local adapter，不读取 Core；Lab 不调用 `admin/local-knowledge`，不使用 Local Profile/文档/进度兜底。

已对接正式服务接口：

- 多文件选取、逐文件内容读取、去重/移除、逐文件提交和错误队列共用 `KnowledgeIngestionWorkspace`。TXT/Markdown 由插件 `POST /admin/runtime/knowledge-lab/spaces/:spaceID/documents` 调用 Framework `UpsertDocument` → Core `POST /api/v1/tenant/knowledge/spaces/:space_uuid/documents`。插件只接受标题、内容、content_type、tags；服务端生成 SHA-256、文档 URI、版本；拒绝租户覆盖和入库快照字段。
- 提交后以 Core 任务 UUID 定位 `GET /admin/runtime/knowledge-lab/index-jobs/:jobID` → Framework `GetIndexJob` → Core `GET /api/v1/tenant/knowledge/index-jobs/:job_uuid`。共享记录页展示本次提交任务，用户点击刷新复读状态；这不是完整历史。无轮询、无 Local WS 进度兜底，202/queued 不等于完成。
- 检索共用 `KnowledgeSearchWorkspace`。Local 和 Lab 的路由分别注入对应 adapter。Local 总览 Playground 链接已指向 Local 自己的空间检索页，避免把 Local 空间 UUID 发给 Core。

### Core 下一步需要提供的合同（本仓库不修改 Core）

1. **空间详情／目录字段**：返回空间及部门 UUID、可读部门名称、实际策略 key、场景 key、Profile UUID/key/version 与索引就绪状态。当前正式空间列表仅返回 UUID/name/status，插件不能推断空间使用了哪一个策略/Profile，也不能推断向量已激活。
2. **入库配置快照**：正式文档 DTO 应明确接收并校验完整的任务级 ingestion snapshot（priority、分段方式/顺序/大小/重叠、separators、anchors、Profile UUID 与版本），并持久化、执行和回读。当前 `documentRequest` 及 `HostDocumentInput` 只接收文本基础字段，不能把 Framework 发出的额外字段被忽略当成支持。完善之前，Lab 共享组件不提交这些设置，显示不可用。
3. **文档和历史读取**：租户隔离的文档列表、原文/向量核验、任务列表/分页/来源筛选、任务快照，以及 chunk 分页/详情/编辑/重建。字段需区分 current document 与 immutable job snapshot，引用使用 UUID，缺失统计不得用 0 冒充。
4. **空间策略更新**：获取实际配置并通过 typed DTO 校验更新，返回生效配置及可用性；当前目录 available 只表示创建能力，不能替代已有空间执行状态。
5. **连接器合同**：提供部门/租户授权下的连接器目录、凭证引用、数据源列表/创建/同步状态；插件不保存或伪造 Core 凭证目录。
6. **进度事件**：明确 Core 任务进度 WS/SSE 事件、租户授权、任务/空间/文档 UUID、阶段、进度和错误字段，并提供 Framework 可桥接合同；在此之前只有用户触发的状态复读。
7. **格式能力**：当前服务仅接受 text/plain 和 text/markdown。若要与 Local HTML/CSV 对齐，Core 应声明支持的 content_type、转换与处理合同；不能把 HTML/CSV 偷换成 Markdown。

### 本次浏览器验收

使用已有 Core 空间“插件联调知识空间”，共享多文件组件提交 `knowledge-ui-one.txt` 和 `knowledge-ui-two.md`。任务 UUID：`2f8703a5-e0f0-46b7-ab23-71a24b74f5d7`、`0a4929ad-a746-4cd0-8ed6-418f0ed92c7d`。自动返回 Lab 记录页，并通过 Core 任务读取接口复读为 `succeeded`；没有 Local knowledge API 请求。测试文档保留在 Core，未自动删除。

## 2026-10-05：修正配置步骤为空与重复提示

共享入库向导恢复第 2 步策略/Profile 草稿、第 3 步分段与 anchors 表单，以及实时配置预览。PowerX 草稿的 Profile key 仅由 Core 实际策略目录映射，不读取 Local Profile 数据或推断空间实际配置。

移除页面顶部及第 2、3 步的重复告警。确认页只显示一次快照合同缺口，`canSubmit` 必须要求 adapter 支持 `ingestionSettings`：当前 Core `documentRequest` 尚未包含快照，所以此完整配置向导保持草稿可编辑、提交不可用。禁止忽略用户配置后走仅文本接口并显示成功。现有基础文档 API 未被撤销；2026-10-04 的两份仅文本联调记录仍是历史验收，不代表完整向导已支持任务快照。

Core 需按上节第 2 项提供快照接收、校验、执行、持久化与回读合同；Framework 再按真实 DTO 映射策略/Profile UUID 与版本及分段快照，并验证错误语义；Plugin 最后启用 adapter 的快照能力。不能仅将前端 `ingestionSettings` 改为 true 绕过缺口。

## 2026-10-06：入库配置快照交付清单

### 当前阻塞及本仓库已完成的修正

再次核对 Core `backend/internal/transport/http/openapi/knowledge_space/routes.go` 的 `documentRequest` 与 `backend/internal/service/knowledge_space/host_contract_service.go` 的 `HostDocumentInput`：仍只有文本基础字段，没有任务配置快照。此前“策略目录与 typed 空间创建已补齐”不包含文档入库配置。

Framework Go Host 客户端已移除未经正式 DTO 支持的顶层 camelCase 配置序列化，任何非 nil `Ingestion`（包括空对象）在获取凭证或发送 HTTP 前返回 `KNOWLEDGE_UNSUPPORTED_CAPABILITY`。测试同时覆盖 API Key、STS 和无快照的正式文本请求字段。Local 配置处理不变；.NET 当前文档 DTO 没有快照参数，等待正式合同后一起接入。

### Core 需要交付什么

以下是消费端要求，**不是已发布的请求 DTO**。请 Core 在 `docs/contracts/knowledge-host-provisioning.md` 及对应 OpenAPI 中确认最终字段名、结构、默认值、约束、错误码与版本；Framework 不自行猜测新协议。

| 配置类别 | 共享向导现有配置 | Core 交付要求 |
| --- | --- | --- |
| 策略与 Profile | `rag_bundle_key`、`rag_scene_key`、`rag_primary`、`ingestion_profile`、`processor_profile`、可选 `masking_profile` | 明确哪些配置支持任务覆盖；策略 key 与 Profile key 分开；所有 Profile 对象引用使用 UUID 与版本。目录提供实际可选值和类型映射，不能从 Local 默认值推断 |
| 优先级 | `priority`: `normal` / `high` | 明确授权、队列执行语义及失败条件；不允许只保存但不执行 |
| 分段 | `segment_mode`: `unit` / `heading` / `clause` / `semantic` / `table_row` / `code_block` / `conversation`；`segment_size_policy`: `cap` / `target` | 声明实际支持的枚举；不支持的模式明确拒绝，不能静默替换 |
| 长度与顺序 | `chunk_size`、`chunk_overlap`、`segment_order` | 明确范围、长度单位、零长度语义、重叠约束、顺序允许项与执行规则 |
| 边界与定位 | `separators`、`page_priority`、`anchor_heading_path`、`anchor_clause_id`、`anchor_row_number`、`anchor_speaker`、`anchor_sentence_index` | 明确分隔符转义、TXT/Markdown 分页行为和 chunk metadata 定位字段；不能忽略不支持的配置 |
| 持久化与回读 | 本次任务完整配置 | 提交时锁定不可变的生效快照；任务回读返回快照及 Profile UUID/版本，文档后续修改不能改变旧任务快照 |

请明确正式 `POST /api/v1/tenant/knowledge/spaces/:space_uuid/documents` 的配置接收结构，并在 `GET /api/v1/tenant/knowledge/index-jobs/:job_uuid` 或单独发布的 typed 任务详情接口中提供快照回读。目录或正式版本信息必须明确声明支持范围；插件不能通过一次 HTTP 202 推断配置已生效。

鉴权沿用 Core 服务凭证决定租户的边界。Core 须声明配置所需的精确能力及 scope；如现有文档授权已经足够，应明确说明，不能要求使用者无依据地重复授权。非法快照、Profile 版本失效、跨租户 Profile/空间、策略不可用和执行失败须提供稳定机器码、HTTP 状态与 trace/request ID。

### Core 最小验收证据

1. 同一租户分别使用真实 API Key 与 STS 提交非默认配置：例如 `chunk_size=640`、`chunk_overlap=80`，并回读完全一致的生效快照。
2. 任务到达真实成功终态后，检查持久化分块与定位 metadata，证明实际执行了该配置；只验证请求 JSON 或返回 202 不算完成。
3. 改变空间/Profile 配置后复读旧任务，原快照和版本保持不变。
4. 缺失授权、跨租户引用、失效 Profile 版本、非法长度/顺序/枚举均明确失败；非法请求不遗留文档或入库任务。
5. 提供正式 OpenAPI/DTO、能力声明、请求/响应样例和脱敏 trace，供 Framework 接入；保持既有无快照文本接口的行为边界明确。

### Framework 与 Plugin 后续实施顺序

1. Framework Go 与 .NET 按正式合同增加 typed snapshot 与任务快照回读；保留 boolean false、数组顺序、UUID、版本和错误语义，不复用旧 camelCase 顶层字段。
2. Plugin Lab Handler 接受并校验 typed 配置，仍拒绝浏览器提供的租户覆盖；共享向导通过 Core 目录构造 Profile UUID/版本，不把 Local Profile key 当作 Core 对象引用。
3. 在真实接收、执行、回读合同验收后启用 `ingestionSettings`，解除当前确认页阻断；不单独翻转前端开关。
4. 浏览器使用多文件队列完成提交、逐文件任务复读和检索引用验证，并确认没有 `admin/local-knowledge` 请求。保留逐文件失败状态，不把 queued 视作入库完成。

当前没有修改 Core、数据库、授权或运行进程；上述后续实施项尚未完成。
