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


## 2026-10-07：Core 快照 v1 已接入并完成插件浏览器验收

本节替代 2026-10-05/06 的“快照合同缺失”当前状态描述；前文保留为历史记录。Core 的 `docs/contracts/knowledge-host-provisioning.md:103` 与 OpenAPI 已发布 `powerx.knowledge.document-ingestion/v1`，用户确认正式 8077 已按原方式重启。此次只修改 PowerXPlugin，没有修改或重启 Core。

- Go Framework 用 `KnowledgeDocument.HostIngestion` / `DocumentIngestionSettings` 发送正式嵌套 snake_case `ingestion`；Local 的 `IngestionConfig` 仍独立。保留 false、0、空数组、数组顺序及精确 Profile UUID/版本，避免旧顶层字段。原文 UTF-8 内容保持原样，与服务端 SHA-256 一致。
- .NET Framework 提供同一 typed settings、目录支持范围、不可变快照/任务详情和 `IKnowledgeJobChunksService`，错误保留机器码、HTTP 状态与 Trace。其 Local adapter 拒绝 Host DTO，避免跨 Provider 写入。
- Lab Handler 接收/验证正式配置，服务端生成 URI/checksum/version；新增 `GET /admin/runtime/knowledge-lab/index-jobs/:jobID/chunks`。没有 Local 文档双写、用户租户覆盖或旧协议降级。
- 共享向导根据 Core 目录 `document_ingestion` 开启配置提交，使用真实目录默认值。任务固定使用空间绑定策略/Profile，由 Core 冻结 UUID 与版本；本版本不提供任务策略覆盖。semantic、PDF、独立 processor/masking 不支持；TXT/Markdown 分页优先禁用，锚点必须匹配模式，Markdown 专用模式不允许用于 TXT。
- 共享记录页提供本次任务生效配置、分块数量、冻结 Profile 版本、技术详情及按需分块读取。显示的是本次提交，不假装完整历史；完整文档目录/历史、文档修改与向量核验仍未接通。实时进度事件未发布，继续由用户手动刷新读取真实状态，不增加自动轮询。

浏览器使用已有空间 `test-base-space` 提交两份生成的非敏感测试文件，每份 2000 rune，配置 `unit/cap/640/80`、空 separators、全部 anchors=false。两次提交 202；任务详情、分块回读均 200，两个任务达到 succeeded。每份实际四块，长度 640/640/640/320，相邻块重叠 80；请求、生效快照及 Profile 版本均可回读。检索 `snapshot20261007` 返回两份文档。整个 Lab 流程未请求 `admin/local-knowledge`。测试资料保留在 Core，未自动删除。

脱敏证据见 [插件浏览器快照验收](evidence/knowledge-lab-snapshot-20261007.json)。这是 Local+proxy API Key 浏览器验收；Go/.NET 的 STS 路径由 transport 合同测试覆盖，未声称插件新增一次真实 STS Exchange。Core 自身的真实 STS、不可变性和非法请求验收见其交付证据。

验证结果：Go Knowledge/Host client 与插件 runtime_ops 包测试通过；.NET Knowledge 19 项、Nuxt 单测 48 项通过；14 个 Knowledge SFC 编译、`sync-templates --check` 和 diff 检查通过。8078 插件后端已重启加载最终代码。最后一次 Local 页面视觉回归因浏览器实例占用及 CUA 连接错误未完成；这不替代或撤销上述已完成的 PowerX 浏览器闭环，Local 行为由共用组件定向测试覆盖。

## 2026-10-07：快照与重建消费端对齐

Core 已交付 `HostIngestionSettings`、任务快照回读及分块接口，此前“快照合同缺失”的说明为历史状态。Go SDK 使用独立 Host DTO 提交 snake_case 配置，保留显式 false、0、空数组和顺序；Lab 从 Core 目录启用受支持的分段选项，Local 参数不作为 Host Profile 引用发送。

本轮按 Core 新 `HostRebuildInput` 对齐单篇及空间重建：显式模式、幂等键和可选 typed 配置；`reuse_snapshot` 不读取新 Profile 配置，`apply_config` 重新校验当前空间绑定。页面在任务记录中提供“重建本篇”和“重建整个空间”弹窗，显示作用范围、实际任务状态、分块数量、生效配置和逐文档配置。同一次提交的网络重试沿用请求键；任务失败后重新发起操作、修改配置或开始新重建时使用新键，以创建独立任务并保留失败历史。没有轮询兜底或 Local 数据回退。

Core 路由与 Service 正在同一轮交付；新路由、迁移、能力授权与运行进程须由 Core 完成部署。当前 SDK/Handler/UI 回归不代替真实 API Key/STS 验收。最终联调需同时验证单篇、空间、跨租户、非法配置、重复/并发、失败重试、策略变化、其他文档未受影响，以及失败后旧索引仍可检索。本仓库不直接修改 Core 或重启运行进程。本次代码尚未发布新 Go 版本。


### 2026-10-07 后续复验：运行恢复与 Local 页面回归完成

Core 当前进程 PID 11316，8077 正常监听，`GET /healthz` 为 200。浏览器已重新连接，复读两份 PowerX 测试任务及 chunks 均为 200，状态 succeeded，每份仍为 640/640/640/320 四块。没有修改或重启 Core。

Local 知识空间 `test` 的两文件向导已逐步检查至第 4 步，策略、分段、锚点保持 Local 行为，提交按钮可用，请求仅访问 Local API。仅生成浏览器草稿，没有提交 Local 文档或触发 Embedding。截图发现 UProgress 使用过时 `value` 绑定导致 0% 时显示不定进度动画，已将两处绑定改为当前 Nuxt UI 的 `model-value`。复验三条进度的 aria-valuenow 均为 0，indicator 为 translateX(-100%)，显示与待提交状态一致。

此前未完成的 Local 视觉回归现已补齐；当前前端 51 项测试、15 个 Knowledge SFC 编译、模板同步和 diff 检查通过。最新结果记录在同一脱敏证据的 `recheck` 字段。


## dev 历史空间 Profile 绑定修复要求（2026-10-07）

用户在 `dev` 提交五份 Markdown，Core 返回 `KNOWLEDGE_PROFILE_UNAVAILABLE`。只读核查空间 `30a6d654-c4c5-4a33-9806-6d66c6728187`：三个 Profile key 都为 `p1_general`，三个 UUID 字段均为 NULL。空间策略仍为 `h_fusion`，场景为 `product_specs`。同租户对应的三个 published v1 Profile 实际存在：

| 空间绑定字段 | 对应已发布对象 UUID | key / version |
| --- | --- | --- |
| ingestion_profile_uuid | 4245b851-0710-4566-9f53-ecf3dd7e04cc | p1_general / 1 |
| index_profile_uuid | 81e39770-93dd-4903-918c-9847224d30a4 | p1_general / 1 |
| rag_profile_uuid | aa8c8f9c-6558-435d-a352-4727adb8163f | p1_general / 1 |

**Core 团队负责数据修复**：执行前重新校验空间租户、Profile 类型、key、UUID、published 状态与版本，在受控事务中补齐历史空间绑定；保留空间已有策略/场景，不自动更换为 p0 或其他策略。若状态或映射已变化须明确失败，不能拿首个目录项代替绑定。修复后使用 dev 真实提交、快照回读、任务终态与 chunks 验收，并给出 HTTP/Trace 证据。此前 test-base-space 的成功记录不代表 dev 已就绪。本仓库没有修改 Core 数据。

Plugin 已将 Profile 不可用及其他正式的写入前拒绝映射为 zh/en 可读提示，并保留 Trace。队列提供显式“重试失败文件”：仅重试有完整原文、未受理且错误属于已发布写入前校验/鉴权拒绝的文件。保留用户配置和之前已受理任务，不重复提交成功项；没有自动重试、Profile 替换、Local 回退。读取失败或结果不明确的请求不自动提供同一重试路径。

验证：前端 53 项单测、15 个 Knowledge SFC 编译、模板同步及 diff 检查通过。真实 dev 浏览器初次两份提交与一次显式重试均返回 422；页面保留原文/配置、可读绑定说明和 Trace。没有自动重试或 Local 请求。成功修复后的实际重试仍待 Core 完成数据修复，不能把负向 UI 验收当成 dev 可入库。证据见 [dev 拒绝与重试复验](evidence/knowledge-lab-dev-profile-rejection-20261007.json)。


## 2026-10-08：dev Profile 绑定修复已完成，插件浏览器提交验收通过

Core 交付 `docs/contracts/evidence/knowledge-dev-profile-bindings-20261008.json`，仅变更三个 Profile UUID 及 updated_at，保留 h_fusion/product_specs 和原空间配置。当前正式 8077、插件 8078、前端 3131 正常运行。本次没有修改或重启 Core，没有修改插件源码。前节 dev 不可入库的描述保留为修复前历史。

浏览器在 dev 生成并提交四份非敏感 TXT/Markdown 测试文件，每份 2000 rune：

- 默认 1024/0：两次 POST 202，两个任务均 succeeded；每份实际两块，长度 1024/976。
- 640/80：两次 POST 202，两个任务均 succeeded；每份实际四块，长度 640/640/640/320，三个相邻边界均保留 80 字符重叠。
- 两组任务/分块回读均为 200，冻结三个 p1_general v1 Profile，strategy 为 h_fusion；请求与生效配置一致。
- 检索 devbrowser20261008 为 200，命中四份测试文档；全程没有 admin/local-knowledge 请求。

证据见 [dev 修复后插件浏览器验收](evidence/knowledge-lab-dev-profile-success-20261008.json)。调试浏览器已重建会话，Chrome 插件连接无法取回用户原五文件的内存队列，因此本次用独立测试文件验证，**没有声称已重试或上传用户原五文件**。原页面若仍保留队列，可直接点击“重试失败文件”；队列已丢失则重新选取文件。无需重启。四份测试资料保留在 Core，未自动删除。

## 2026-10-08：任务过程页优化与剩余 Core 合同

Plugin 的 Local 入库向导与 PowerX 任务记录现共用 `KnowledgeTaskProcessList`。展示真实待提交/受理/排队/执行/终态、成功与失败数量、任务结束比例、分块数量和耗时。批次进度明确表示“已结束任务数 / 总任务数”，失败计入终态但不计成功；单任务执行百分比仅使用实际事件数值，没有数值时使用不定进度动画，不模拟百分比或延迟。

入库向导记录的是请求耗时；记录页使用 Core started_at/completed_at 计算执行耗时，未结束任务不显示虚构耗时。PowerX 列表收敛为紧凑行，参数、分块与 Trace 进入共用任务详情弹窗；索引模式集中展示，完整历史缺口改为中性范围说明。Core 当前只返回单任务状态，页面继续提供显式“刷新状态”，不添加自动轮询。Core 分块的用户显示序号按展示顺序计数，不改变实际 ordinal 或 metadata。

本节中的以下能力仍需 Core 正式交付；本仓库不创建临时 Core 路由，不直接读取 Core 数据库作为插件运行时接口，不修改 Core。

### Core A：空间入库就绪检查

请发布 typed、只读的空间就绪合同，最终路径/DTO 由 Core 确认并写入 OpenAPI。接受空间 UUID 和声明的索引模式，按当前 API Key/STS 租户与权限校验：三个 Profile UUID/key/精确版本/published 状态、策略/场景与索引绑定、文档入库授权、格式及模型/存储依赖。返回 ready、稳定 blocked reason code、可读名称/locale key、支持模式与能力版本，保留 HTTP/Trace。检查不修复数据、不创建 Profile、不替换默认策略；明确检查到提交之间状态可能变化，提交仍须重新校验。

Plugin 现仅能校验已经公布的格式、长度、模式/锚点等参数与目录支持范围，不能根据目录总体可用性假装空间依赖已检查。当前已发布的 semantic-index 能力属于语义/向量配置，不能拿它替代所有 lexical 空间的就绪结果。

### Core B：文档目录与任务历史

请发布受租户和对象授权约束的 typed 分页文档列表、任务列表/详情，支持空间 UUID、文档 UUID、状态与分页筛选。文档提供可读标题、source_type、当前版本/状态、当前及生效任务 UUID、分块数量；任务提供可读来源名、operation、status、started/completed/created、priority、错误码/Trace、冻结快照和 processed/total 的真实语义。查询不依赖 Plugin URL 中的 job_uuid 数组，能够刷新/重新进入后回看；未知统计使用缺失值，不能填 0。

Profile/文档/任务引用使用 UUID。合同需区分当前文档与不可变任务快照，说明排序、删除后的历史保留、分页边界及权限错误。不通过 admin 用户 JWT 路由或 raw proxy 绕过服务凭证合同。

### Core C：任务实时进度

请发布并验收 WS/SSE 或可由 Framework EventBridge 接收的正式任务事件合同。至少提供 tenant/space/document/job UUID、事件版本、seq/cursor、operation、phase/status、processed/total、可选真实 progress_percent、错误码、时间与 Trace。区分 queued、running、chunking、embedding、writing、succeeded、failed 等实际支持阶段；不存在的阶段不发布，百分比未知必须缺失。定义订阅授权、断线恢复、乱序/重复去重以及终态与持久化状态的一致性。

Local+proxy 通过服务端 API Key，Delegated 通过 STS/可信上下文订阅；浏览器不能持有 Core 服务凭证，不能将 Local JWT 转发为 Core 身份。Framework 桥接到 Plugin WS 时必须校验宿主租户与插件订阅租户的授权映射，不转发到无权会话。请给出准确 capability/scope 及是否复用既有 grant，避免无依据地重复授权。

### 接入与验收顺序

Core 分别交付 A/B/C 的 OpenAPI、DTO、能力声明、真实 API Key/STS 请求/响应及 Trace。Framework 再添加 typed readiness/page/subscription 映射；Plugin 最后启用对应 capability。不能单独翻转 `documents`/`realtime`，不能使用 SQL 或 Local 数据回退、自动轮询、计时百分比或仅会话缓存冒充历史/实时状态。当前剩余合同没有交付，完整预检、历史、实时进度不得标为已完成。

本次验证：60 项前端测试通过；16 个 Knowledge SFC 编译、模板同步和 diff 检查通过。浏览器复用用户已有五个 Core 任务验证成功 5/5、结束比例 100%、实际分块及执行耗时；详情弹窗读到第一份文档的两块内容，显示序号为 1/2。Local 两文件草稿至确认页显示待提交 2、结束 0/2，没有写入文档。queued/running 的无百分比呈现、手动状态刷新和详情授权失败通过定向测试验证。未修改/重启 Core，未重传用户原文件。证据见 [任务过程页复验](evidence/knowledge-task-process-ui-20261008.json)。


## 2026-10-08：Playground 语义接口加载与就绪状态修复

页面语义能力请求原为 Plugin 8078 的 404/plain text，旧进程 PID 82692 尚未加载新增 semantic 路由；此前前端把没有结构化错误码的响应错误显示为 KNOWLEDGE_UPSTREAM_DEPENDENCY。本次定向 Go 回归通过后，仅重启获授权的插件后端，新进程 PID 65141。当前正式能力接口 200，dev 实际 configured=false/ready=false/indexed=false，reason_code=KNOWLEDGE_SEMANTIC_INDEX_NOT_READY。Core 服务未修改或重启。

新增共用 requestFailure 提取器：读取结构化 machine code、真实 HTTP 状态和 Trace，不从 free-text body 解析、不杜撰上游错误码。Semantic Panel 与 Playground 区分传输错误和正常未就绪状态，显示模型配置/索引准备引导，检索仍由真实 ready+indexed+indexed_query_modes 阻止。页面明确返回召回片段、分数和来源，未添加生成答案；没有自动 lexical 回退。

68 项前端测试、17 个 Knowledge SFC 编译、Go Knowledge/Host 与 Plugin runtime_ops 定向回归、模板同步及 diff 检查通过。模型绑定和向量重建会改变 Core 业务配置/数据，当前等待用户明确扩展授权，尚未执行；不能把接口修复当作真实向量 QA 验收。证据见 [Playground 接口修复](evidence/knowledge-playground-interface-fix-20261008.json)。


## 2026-10-08：dev 语义/混合检索插件浏览器验收通过

Core 完成 dev 的模型配置和原五文档语义准备，交付报告为 `PowerX/docs/contracts/repairs/knowledge-dev-semantic-preparation-20261008.md`。插件页面当前已配置/通道就绪/已有发布索引，模型 ollama/bge-m3，真实 1024 维，Embedding Profile UUID 2ff1b71c-5fbe-4753-9c17-8ca3a85cdce2 v1，hybrid 文档5份/分块23个。上节未配置状态属于准备前历史，当前无需再为本轮验收重启或配置数据。

在实际 Playground 页面，围绕 Provider 模式和宿主全屏接入各提出完整自然语言问题，semantic/hybrid 各测试一次，共4次服务请求均200，目标文档排名1。保持 top_k=5、空过滤、required_generations=已读取的实际 corpus generation；语义结果显示余弦原始距离/相似度，hybrid 显示 RRF，来源/version/generation/Trace保留。同文档多个相关分块没有去重。

只读核对浏览器返回的20个片段：与Core持久化原文的codepoint区间、完整原文SHA256及source version全部一致。此SQL只用于独立验收，插件运行时仍走 typed Host合同；没有Local knowledge或旧lexical fallback请求，没有修改Core配置/数据、没有重启任何服务。本轮没有修改插件源码。

本阶段仅验收知识召回，不宣称已接入LLM生成答案或完成新的插件STS Exchange。证据见 [语义检索浏览器验收](evidence/knowledge-playground-semantic-browser-20261008.json)。


## 2026-10-08：共用知识检索页面交互收敛

Local 与 PowerX 的 Playground 继续共用 `KnowledgeSearchWorkspace`，页面标题改为“知识检索测试”。主流程为输入问题、检索、阅读原文片段及查看来源；顶部只显示空间与实际就绪状态，模型配置移入“索引设置”弹窗，检索模式、数量、版本锁定和过滤进入折叠的“高级检索设置”。结果优先展示标题与片段，分数、来源偏移、UUID、版本和 Trace 默认折叠。示例按钮只填入问题，不自动发起查询。

索引设置默认关闭时，页面仍会读取实际 capability；打开/关闭设置不会清空已有问题与结果，刷新或配置变更才使旧结果失效。读取失败明确阻止 semantic 查询并提供手动刷新，查询失败保留问题，不添加 Local/lexical 回退。默认 semantic、top_k=5、required_generations 保持，未新增 LLM 生成答案。

71 项前端测试、17 个 Knowledge SFC 编译、zh/en 文案键检查、本次知识库模板双目标同步及 diff 检查通过。浏览器 semantic/hybrid 各一条自然语言查询均 200，各返回 5 个真实片段；Local 检索 200，provider=plugin_local，返回 5 个片段。已检查弹窗往返、片段展开/收起、默认详情折叠，以及 1435/1024 宽度下工作区无横向溢出。全仓模板检查仅剩当前工作树 AGENTS.md 对应两份模板的既有差异，未扩展修改。

本次只调整前端共用交互、locale、回归测试和生成模板；没有修改或重启 Core，没有操作模型绑定、隐藏文档、重建或重新入库。浏览器保持在 PowerX 检索页，可直接刷新使用。证据见 [知识检索页面优化复验](evidence/knowledge-playground-ui-20261008.json)。


## 2026-10-09：内容导览组件与 Core 目录交付边界

共用检索页新增 `KnowledgeContentOverview`：文章目录与原文预览进入弹窗，问题选择仅填入输入框。Local 从已有空间文档 API 读取真实标题与原文前 320 个 Unicode codepoint；这是目录预览，不表示所有文章都已发布索引，不生成 AI 摘要。固定的 Framework 示例问题已移除，组件不会用验收样本或检索采样结果冒充空间目录。

PowerX Host 尚未发布空间文档目录读取合同，Lab 内容导览 accessor 没有启用。页面明确说明内容导览尚未接入，检索继续独立使用；没有添加猜测的 Core URL、Local 回退或 Core SQL 读取。Framework 的 typed wire DTO/client/adapter 需等 Core 正式合同后实施，不能把前端 view model 当作已发布合同。

Core 最小交付是当前授权、当前检索 mode/corpus generation 下的真实文章目录、原文预览、稳定分页及授权/版本冲突错误；概要和来源文章绑定的推荐问题属于可选真实产物。具体职责、字段语义与 API Key/STS 验收见 [内容导览 Core 交付要求](knowledge-content-overview-core-requirements.md)。

验证：80 项前端测试、18 个 Knowledge SFC 编译及 locale 检查通过，新增组件在运行中的 Nuxt 服务编译为 JavaScript 并返回 200；内容导览相关双目标模板同步与 diff 检查通过。目录失败/空目录、跨空间响应、语料 generation 不匹配、分页重复以及检索中的问题替换已覆盖。浏览器工具连接失败，未完成真实 UI/Local 目录数据浏览器验收；PowerX 真实导览数据待 Core 交付，不标记为完成。未启动、停止或重启服务，没有修改 Core。
