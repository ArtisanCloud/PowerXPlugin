# Knowledge Lab 对 PowerX Core 的补齐要求

本次修复页面：`http://127.0.0.1:3131/powerx/knowledge-lab`。
界面参照：`http://127.0.0.1:3131/_p/com.powerx.plugins.base/admin/knowledge`。
仅修改 PowerXPlugin；本文是向 Core 提出的要求，不代表 Core 已完成或已授权这些能力。

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
