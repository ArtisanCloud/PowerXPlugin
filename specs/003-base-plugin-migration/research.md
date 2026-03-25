# Research: Base Plugin Migration

## Phase 0 Discovery Summary

- 目标：梳理 `com.powerx.plugins.base` 中与 Templates CRUD 相关的响应结构、错误码、租户上下文约定，为 Phase 1/2 的框架与 Skeleton 实现提供基准。
- 资料来源：`com.powerx.plugins.base/backend/internal/transport/http/admin/templates/**`、`internal/services/admin/templates/**`、`internal/entity/repository/template/**`、`plugin.yaml`、`web-admin/app/composables/api/useTemplate.ts`。

## Response Envelope 观测

### APIResponse 字段（源：`backend/internal/contracts/api.go`）

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `success` | bool | 请求是否成功 |
| `data` | any (可选) | 成功时返回的实体或分页对象 |
| `error` | `{ code, message, details? }` (可选) | 失败时的错误结构 |
| `message` | string (可选) | 附加的成功信息 |
| `timestamp` | RFC3339 | 服务器生成时间 | 
| `request_id` | string (可选) | 由 RequestID 中间件提供，用于日志关联 |

`APIError` 结构固定包含 `code` 与 `message`，并允许附带 `details` 说明（如字段验证信息）。

### 错误码清单（`backend/internal/contracts/api.go`）

- 通用：`INTERNAL_ERROR`、`INVALID_REQUEST`、`UNAUTHORIZED`、`FORBIDDEN`、`NOT_FOUND`、`CONFLICT`、`VALIDATION_FAILED`
- 模板 CRUD 实际使用：
  - 参数错误 / JSON 解析失败 → `INVALID_REQUEST`（HTTP 400）
  - 资源不存在 → `NOT_FOUND`（HTTP 404）
  - 缺少租户信息或权限不足 → `UNAUTHORIZED` / `FORBIDDEN`（HTTP 401/403）
  - 未处理异常 → `INTERNAL_ERROR`（HTTP 500）
- 其他领域扩展：`TENANT_NOT_FOUND`、`TENANT_MISMATCH`、`AGENT_TOOL_NOT_FOUND` 等（Skeleton 目前暂未使用，但需要保留扩展点）。

HTTP 状态码与错误码的映射在 handler 中明确调用 `contracts.Response*` 方法控制，框架响应助手需支持自定义 status + code 组合。

## 租户上下文与数据类型

- `tenant_uuid` Header 为主要租户来源，类型为 **UUID string**。
- Repository 层要求：
  - 内嵌 `repository.BaseRepository[T]`
  - 在查询/更新时追加 `tenant_uuid = ?` 条件
  - 在事务或连接初始化时调用 `SET LOCAL app.tenant_uuid`
- Skeleton 内存实现需持续保存 `tenant_uuid`（可通过 `map[uint64][]Template` 或在记录上保留原字段），并在无 Header 时回退至 Standalone 默认租户 `1`。

## Templates CRUD 请求 / 响应样例

### 路径与方法（源：`backend/internal/transport/http/admin/templates`）

| Method | Path | 说明 | 请求体 | 响应摘要 |
| --- | --- | --- | --- | --- |
| GET | `/api/v1/templates` | 查询模板列表，支持 `q`、`page`、`page_size` | n/a | `data` 为 `Page` 结构：`{"list": [...], "page_index": 1, "page_size": 20, "total": 1}` |
| GET | `/api/v1/templates/:id` | 获取单个模板 | n/a | `data` 为单个模板对象 |
| POST | `/api/v1/templates` | 创建模板 | `{ "name": "...", "description": "...", "content": "..." }` | 返回新建模板对象 |
| PUT | `/api/v1/templates/:id` | 更新模板 | 同 POST | 返回更新后的模板对象 |
| DELETE | `/api/v1/templates/:id` | 删除模板 | n/a | `data` 为 `{ "ok": true }` |

### 成功响应示例

```json
{
  "success": true,
  "data": {
    "list": [
      {
        "id": 42,
        "tenant_uuid": 100,
        "name": "Draft",
        "description": "Demo template",
        "content": "Hello",
        "created_at": "2025-10-31T12:34:56Z",
        "updated_at": "2025-10-31T12:34:56Z"
      }
    ],
    "page_index": 1,
    "page_size": 20,
    "total": 1
  },
  "timestamp": "2025-10-31T12:34:56.789Z",
  "request_id": "req-123"
}
```

### 常见错误场景

- `GET /api/v1/templates/abc` → HTTP 400 + `INVALID_REQUEST`（编号解析失败）
- `GET /api/v1/templates/1` 且非本租户 → HTTP 404 + `NOT_FOUND`
- 缺少 `tenant_uuid` → HTTP 401 + `"tenant context missing"`
- 未处理异常 → HTTP 500 + `INTERNAL_ERROR`

> 以上示例基于 handler 与 repository 行为推导；实现阶段应在测试中复现并记录实际响应。

## 组件复用清单

- 前端页面：`intro.vue`、`templates/index.vue`、`templates/crud.vue`
- 公共组件：`TemplateFormModal.vue`、`ConfirmDialog.vue`、`ToastAlert.vue`
- Composable：`useTemplateApi`（主要引用 `$fetch` 和 `tenant_uuid` 逻辑）

## 未决事项 / 待确认

- Base 插件是否存在特殊的错误码扩展（例如安全、配额相关），若需要在 Skeleton 中预留位置需继续盘点。
- CLI 模板中 starterPages 的默认开关策略（是否允许通过参数关闭）。
- BaseRepository 内存适配层：Skeleton 使用局部 map 存储即可，暂未发现需要在 framework 层抽象公共适配器的需求，后续若多项目共享可再单独拆出。

> 后续任务：T001/T002/T026/T030 将补充具体示例、执行验证步骤与输出差异记录。

## 覆盖率记录

- 2025-11-01：`go test ./framework/backend/go/... -coverprofile=framework/backend/go/coverage.out`；router 包覆盖率 93.8%，结果已留存。

## CLI 脚手架验证（T030）

- 2025-11-02：`./bin/px-plugin init com.powerx.demo` 成功生成完整骨架。
  - 新版模板已正确转义 Vue `{{ }}` 语法，未再出现 `function "templates" not defined` 报错。
  - 输出目录包含内存版 Templates CRUD（backend/internal/templates/**）与前端页面/组件（web-admin/app/**）。
  - CLI 同步写入 contracts（docs/contracts/*.json|yaml）与默认 README/插件元数据，符合 Phase 5 预期。
  - 后续步骤：针对生成项目执行 `go test ./...`、`npm run lint` 并比对 Skeleton（T034/T035）。

## 综合验证（T034）

- 2025-11-02：`GOWORK=off go test ./...`（目录：`com.powerx.demo/backend`）通过，`manifest.Menu` 新增 `children` 字段后编译正常。  
- 2025-11-02：`npm run lint`（目录：`com.powerx.demo/web-admin`）执行占位脚本，确认 CLI 模板内置占位 Lint 流程；`npm install` 安装本地 Layer 依赖成功。  
- 2025-11-02：`./bin/px-plugin init --force com.powerx.demo` 再次运行验证模板幂等性，输出与 Skeleton 保持一致。

## 延迟观测（T035）

- 2025-11-02：通过脚本启动 `go run ./skeleton/backend/go-gin/cmd/plugin` 并在 0.5s 间隔内轮询 `/api/v1/ping` 直至服务就绪；随后使用 `curl -w 'time_total:%{time_total}'` 观测多租户请求。  
  - `tenant=1` → `0.001355s`  
  - `tenant=2` → `0.001451s`  
- 结果均远低于 1s SLA，命令执行后立即终止服务并保存原始响应到 `/tmp/tenant{1,2}.json` 供复核。

## Nuxt 配置 Diff 记录（Phase 7 进行中）

- 2025-11-02：Skeleton/CLI 已引入 Base `nuxt.config.ts` 主要段落（`compatibilityDate`、`ssr=false`、`runtimeConfig`、Nitro headers、`POWERX_PROXY` 切换、`@nuxt/icon`、`@pinia/nuxt`、`@nuxtjs/color-mode`）。  
- 语言包目录统一调整为 `i18n/locales`（`langDir: '../i18n/locales'` / `i18n/locales`），Skeleton、CLI、Scaffold 模板已同步。  
- 默认布局、导航栏、侧边菜单、主题/语言切换组件已迁移至 Skeleton，并复制到 CLI/scaffold 模板；当前实现保留简化版 Bridge（仅保留主题初始化，无宿主通信）。  
- 待确认：桥接、store、Tailwind pipeline 是否需要 Stub；若不迁移，将在 Phase 7 输出差异说明。

## Phase 7 收敛结果（2026-03-25）

### T039：Nuxt 配置对齐 Diff（Base 对照项）

对照项来源：`spec.md` 中 FR-009 约束与 `docs/plan/002-plan-base-plugin-migration.md` 差异清单。  
结论：Skeleton/CLI/scaffold 三套模板均已对齐核心项，未发现结构性偏差。

| 对照项 | Skeleton | CLI 模板 | Scaffold 模板 | 结论 |
| --- | --- | --- | --- | --- |
| `compatibilityDate` | ✅ `2025-11-02` | ✅ | ✅ | 对齐 |
| `ssr` | ✅ `false` | ✅ | ✅ | 对齐 |
| `runtimeConfig.public`（`apiBaseUrl/pluginApiBase/insidePowerX/iamMode`） | ✅ | ✅ | ✅ | 对齐 |
| Nitro `routeRules` headers | ✅ | ✅ | ✅ | 对齐 |
| Dev HMR / 代理（`/api`、`/ws`、`/_p/{pluginId}/api`） | ✅ | ✅ | ✅ | 对齐 |
| `@nuxt/icon` / `@pinia/nuxt` / `@nuxtjs/color-mode` | ✅ | ✅ | ✅ | 对齐 |
| devtools 开关（宿主模式禁用） | ✅ | ✅ | ✅ | 对齐 |

### T040：语言包路径与加载策略

1. 语言包目录统一：`i18n/locales/{zh,en}.json`。
2. Nuxt i18n 配置统一：`langDir: '../i18n/locales'`（`srcDir: 'app'` 前提下兼容 Skeleton/CLI/scaffold）。
3. 两种运行模式策略：
   - Standalone：启用浏览器语言探测（cookie `px_lang`）。
   - 宿主代理（`POWERX_PROXY=1`）：关闭浏览器探测，避免宿主与插件双重重定向冲突。
4. 历史兼容：`i18n/{zh-CN,en}/menus.json` 仍保留用于菜单文案回退，不影响新 `locales` 主链路。

### T042：未迁移能力与后续 Roadmap

| 能力 | 当前状态 | 影响 | 建议 |
| --- | --- | --- | --- |
| Bridge 完整通信（`app/bridge`） | 部分迁移（仅保留最小主题初始化） | 宿主高级交互能力未示例化 | P2：补最小可运行 bridge stub |
| Pinia 业务 Stores（operations/dev-console） | 未迁移 | Starter 页面状态管理示例较少 | P2：提供 1-2 个 store 示例 |
| Tailwind v4 pipeline 深度定制 | 未迁移 | 主题扩展能力受限 | P3：单独评估升级成本 |
| Dev Console/Operations 扩展模块 | 未迁移 | 与 Base 高阶模块存在差距 | P3：以可插拔子模块形式分期 |

执行建议：

1. 下一迭代优先补 `Bridge Stub + Store 示例`（低风险高收益）。
2. Tailwind4 与 Dev Console 放入后续里程碑，不阻塞当前 CRUD Starter 交付。
