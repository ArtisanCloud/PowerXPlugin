# plugin.yaml 规范（Plugin Manifest Specification）

> 本页目标：定义 **PowerX 插件清单文件**（`plugin.yaml`）的字段、规则与加载流程。  
> 读者对象：插件开发者 / 平台集成方 / Marketplace 运营人员。

> 📁 **最新脚手架示例**：`docs/lifecycle/examples/plugin.yaml` 为唯一维护点，编辑完成后执行 `make sync-lifecycle-docs` 同步到公开文档。请勿直接修改集成目录下的副本。快速上手请参考 [`docs/lifecycle/quickstart.md`](../lifecycle/quickstart.md) 与 [`docs/lifecycle/bootstrap.md`](../lifecycle/bootstrap.md)。

---

## 一、文件位置与作用

每个插件必须在根目录包含一个 `plugin.yaml` 文件，  
这是 PowerX Plugin Manager 的唯一识别入口，用于：

- 注册插件基本信息；
- 声明后端/前端运行入口；
- 定义权限（RBAC）与菜单；
- 注册 Agent 能力；
- 描述依赖、替代或冲突插件；
- 指导打包与市场分发。

PowerX 在启动时会扫描插件目录：

```

plugins/
├── com.powerx.plugins.crm/
│   └── plugin.yaml
├── com.powerx.plugins.ecommerce/
│   └── plugin.yaml
└── com.powerx.plugins.base/
└── plugin.yaml

```

扫描到后：

1. 验证结构；
2. 注入运行环境变量；
3. 启动后端进程；
4. 挂载前端与反代路径；
5. 拉取 `/admin/manifest` 与 `/admin/rbac` 信息注册到宿主系统。

---

## 二、基本结构

`plugin.yaml` 现行 schema 与 `docs/lifecycle/examples/plugin.yaml` 保持 1:1 对齐，顶层通常包含：

1. **元信息**：`id/name/version/description/corex_version/security_baseline_version/data_usage`
2. **运行入口**：`runtime`（宿主如何启动进程）、`backend`（供反代的 HTTP 服务）、`endpoints`
3. **前端托管**：`frontend.admin`（Nuxt/Nitro 服务、静态兜底、i18n、菜单）
4. **路由与权限**：`routes`、`rbac`、`permissions`
5. **能力声明**：`events`、`agents`、`capabilities`、`tools`、`workflows`
6. **分发与合规**：`dependencies`、`migrations`、`assets`、`checksums`、`signature`、`metadata`

以下示例节选自 `docs/lifecycle/examples/plugin.yaml`，其余字段请直接参考该文件：

```yaml
id: com.powerx.plugins.base
name: Base template Plugin
version: 0.7.0
description: "PowerX 基础任务管理插件，提供任务、权限与 Agent 能力"
corex_version: ">=0.1.0"
security_baseline_version: "2025.10"

data_usage:
  - purpose: tenant_data_classification
    type: pii_inventory
    retention: 365d
    requires_consent: true

runtime:
  kind: process
  entry: backend/bin/plugin
  env:
    POWERX_BIND_ADDR: ${POWERX_BIND_ADDR:-:8086}
    POWERX_SECURITY_JWT_SECRET: "${POWERX_AUTH_JWTSECRET}"
  health:
    http: /healthz
    interval: 10s
    timeout: 3s

backend:
  entry: backend/bin/plugin
  port: 8086
  health: /healthz

endpoints:
  http_base_path: /api/v1

frontend:
  admin:
    kind: process
    process:
      entry: node
      args: ["./web-admin/.output/server/index.mjs"]
      env:
        NODE_ENV: production
        POWERX_PROXY: "1"
    static_dir: web-admin/.output/public
    i18n:
      dir: web-admin/i18n
      format: i18next
      default_namespace: menus
      locales: [zh-CN, en]
    menus:
      - id: plugins.base
        title_i18n:
          namespace: menus
          key: menu.base.template
          default: Base Plugin
        path: /intro
        required_policies: [base:template:read]

routes:
  basePath: /api/v1
  adminManifest: /api/v1/admin/manifest
  rbac: /api/v1/admin/rbac
  operations:
    admin: /api/v1/admin/operations
  marketplace:
    admin: /api/v1/admin/marketplace
    public: /api/v1/marketplace
    checklist_graphql: /api/v1/admin/marketplace/checklist/graphql

rbac:
  resources:
    - resource: base:template
      actions: [read, create, update, delete]

permissions:
  - resource: base:template
    actions: [read, create, update, delete]

agents:
  - id: base.assistant
    plugin_id: com.powerx.plugins.base
    default_tools:
      - base.template.create
      - base.template.query

capabilities:
  provides:
    - id: base.template.create
      descriptor: contracts/capabilities/base.template.create.yaml

tools:
  - id: base.template.create
    transport: http
    endpoint: ${POWERX_PLUGIN_HTTP_BASE:-/api/v1}/templates
    method: POST
    rbac_resource: base:template

events:
  topics:
    - key: _topic.marketplace.license.renewal.due
      actions: [publish]
      description: "License 续费任务触发事件"

migrations:
  driver: go
  entry: backend/bin/migrate
  args: ["setup"]
  workdir: ./backend

assets:
  public_dir: ./web-admin/public
  webAdminPath: ./web-admin/.output

checksums:
  package_sha256: ""

signature:
  enabled: false

metadata:
  author: PowerX Team
  category: productivity
  homepage: https://powerx.dev/plugins/base
```

---

## 三、字段详解

> 下表与后续小节均以示例文件为准，如需新增字段，请先更新 `docs/lifecycle/examples/plugin.yaml` 再回写本规范。

### 1️⃣ 元信息（Metadata）

| 字段                       | 类型     | 必填 | 说明 |
| ------------------------ | ------ | -- | --- |
| `id`                     | string | ✅  | 插件唯一标识，命名空间风格：`com.<org>.<category>.<name>`。 |
| `name`                   | string | ✅  | 插件显示名称，可结合前端 i18n。 |
| `version`                | string | ✅  | 语义化版本，遵循 `semver`。 |
| `description`            | string | ☐  | 简要说明。 |
| `corex_version`          | string | ✅  | 所需 PowerX Core/Plugin Manager 版本范围，例如 `>=0.1.0`。 |
| `security_baseline_version` | string | ☐  | 当前遵循的安全基线版本号，用于 Marketplace 审核。 |

> `author`、`license`、`homepage` 等信息统一收敛到文末的 `metadata` 段，避免重复。

### 2️⃣ 数据使用（`data_usage`）

- 描述插件涉及的用户 / 租户数据处理场景，供安全与隐私审计。
- 每一项包含：

| 字段               | 说明 |
| ---------------- | ---- |
| `purpose`        | 数据处理目的（如 `tenant_data_classification`）。 |
| `type`           | 数据类别，示例：`pii_inventory`、`operational_log`。 |
| `retention`      | 保留时长，单位可为 `d`、`h`。 |
| `requires_consent` | 是否需要租户额外同意。 |

### 3️⃣ runtime（运行时托管）

| 字段        | 说明 |
| ---------- | ---- |
| `kind`     | 当前仅支持 `process`，表示由宿主直接 `exec` 二进制或 Node 进程。 |
| `entry`    | 可执行文件路径，通常与 `backend.entry` 相同。 |
| `env`      | 启动所需环境变量，支持 `${VAR:-default}` 模板，敏感值由宿主注入。 |
| `health`   | 存活探针：`http`（路径）、`interval`、`timeout`。 |

PowerX 将渲染 `POWERX_*` 变量后启动该进程，并按健康检查结果做重试或告警。

### 4️⃣ backend（反代入口）

| 字段     | 说明 |
| ------ | ---- |
| `entry` | 后端二进制（或脚本）相对路径。 |
| `port`  | 插件监听端口（宿主在本地回环访问）。 |
| `health`| HTTP 健康检查路径，仅返回 200 即视为存活。 |

### 5️⃣ endpoints

- `http_base_path`：插件内业务 API 的统一前缀，例如 `/api/v1`，便于 tools / Agent 构造 URL。

### 6️⃣ frontend.admin（后台 UI 与菜单）

| 字段 | 说明 |
| --- | --- |
| `kind` | `process` 或 `static`，Nuxt 4 默认 `process`。 |
| `process.entry/args/env` | 如何启动 Nitro Server。 |
| `process.health` | 前端健康检查，推荐实现 `/healthz` 路由。 |
| `static_dir` | 静态兜底目录，当 SSR 宕机时直接喂文件。 |
| `i18n` | 包含 `dir`、`format`（如 `i18next`）、`default_namespace`、`namespaces`、`locales`。 |
| `menus` | 菜单树，与前端路由和权限绑定。子项可递归包含：`id`、`title`/`title_i18n`、`icon`、`path`、`route`、`order`、`slot`、`required_policies`、`children`。 |

> 菜单标题需配合 i18n key，`required_policies` 应与 `permissions` / `rbac.resources` 中的动作匹配。

### 7️⃣ routes（路由映射）

| 字段                       | 说明 |
| ------------------------ | ---- |
| `basePath`               | 插件业务 API 前缀（建议 `/api/v1`）。 |
| `adminManifest`          | 管理端 Manifest 接口。 |
| `rbac`                   | RBAC 清单接口。 |
| `operations.admin`       | Ops 控制面接口，用于任务编排或运行观测。 |
| `marketplace.admin`      | Marketplace 管理端接口。 |
| `marketplace.public`     | Marketplace 公共查询接口。 |
| `marketplace.checklist_graphql` | 审核清单 GraphQL 入口（如适用）。 |

宿主反代规则示例：

```
/_p/<plugin-id>/api/*   → backend:port
/_p/<plugin-id>/admin/* → frontend.admin.static_dir / 进程
```

### 8️⃣ rbac（资源与动作）

`rbac.resources` 描述宿主需要聚合的资源、动作及用途：

```yaml
rbac:
  resources:
    - resource: marketplace.listings
      actions: [read, write, review]
```

此结构会与 `/api/v1/admin/rbac` 响应进行校验，必须与运行时代码保持一致。

### 9️⃣ permissions（权限声明）

权限数组用于 Marketplace 与宿主 UI 展示，结构与 `rbac.resources` 类似，但更关注插件内部消费：

```yaml
permissions:
  - resource: base:template
    actions: [read, create, update, delete]
```

> 插件无需在 YAML 中声明角色或绑定关系，宿主会根据租户策略注入最终权限列表。

### 🔟 events（事件能力）

```yaml
events:
  topics:
    - key: _topic.marketplace.license.renewal.due
      actions: [publish]
      description: "License 续费任务触发事件"
```

- `topics[].key`：Topic 标识（建议 `_topic.*` 命名）。
- `topics[].actions`：声明行为（`publish` / `subscribe`）。
- `topics[].description`：用于审核、文档和可观测上下文。
- 过渡期执行层映射文件为 `config/event_fabric.yaml`（供底座当前实现扫描）。

#### 事件 Topic 声明规范（强制）

插件必须在 `plugin.yaml` 中声明事件主题，建议使用 `events.topics[]`：

```yaml
events:
  topics:
    - key: orders.created
      actions: [publish, subscribe]
      description: 订单创建事件
```

约束：

1. `key` 表示语义主题（`namespace.name`），不包含 tenant 前缀。
2. `actions` 仅允许：`publish` / `subscribe` / `replay`。
3. 插件代码中禁止“未声明 topic 直接调用”。
4. `plugin.yaml` 是规范声明层；执行层由插件包中的 `config/event_fabric.yaml` 提供给底座播种。

### 1️⃣1 Agents / Capabilities / Tools / Workflows

- **agents**：声明 Agent ID、描述、默认工具、所需权限等，详见 [Agent Contract](./agent_contract.md)。
- **capabilities**：`provides`/`consumes` 引用能力描述文件（YAML/JSON），并指向输入输出 schema。
- **tools**：对接 HTTP/GRPC 接口，必须包含 `transport`、`endpoint`、`rbac_resource`，并推荐提供 JSON Schema。
- **workflows**（可选）：声明多步骤编排，字段包含 `steps`、`required_permissions` 等。

这些字段共同驱动 Agent Hub、px-plugin CLI 以及 Marketplace 的能力目录。

### 1️⃣2 dependencies / migrations

| 字段          | 说明 |
| ----------- | ---- |
| `dependencies.requires` | 需要先安装/启用的其他插件或外部服务。 |
| `dependencies.conflicts` | 不可同时启用的插件列表。 |
| `dependencies.replaces` | 替换/升级关系。 |
| `migrations` | 描述迁移命令：`driver`（go/bash/tern）、`entry`、`args`、`workdir`、`once`、`timeout`。 |

### 1️⃣3 assets / checksums / signature / metadata

- `assets.public_dir`：静态资源根目录。
- `assets.webAdminPath`：Nuxt 打包目录（供宿主直接挂载静态资源）。
- `checksums.package_sha256`：发布后由 CI 写入，用于完整性校验。
- `signature`：是否启用离线包签名，后续若启用需补充算法与公钥。
- `metadata`：作者、分类、标签、图标、主页、许可证等 Marketplace 展示信息。

---

## 四、平台加载流程

```text
PowerX 启动
  ├─ 扫描 plugins/<id>/plugin.yaml
  ├─ 解析 backend/port/routes
  ├─ 启动插件进程
  ├─ 反代注册 (/_p/:id/api/*, /_p/:id/admin/*)
  ├─ 调用 /api/v1/admin/manifest
  ├─ 调用 /api/v1/admin/rbac
  ├─ 聚合菜单与权限
  └─ 注册 Agent 能力
```

---

## 五、最佳实践

✅ **语义化版本号**
版本应遵循 semver 规则（`MAJOR.MINOR.PATCH`）。

✅ **多语言菜单**
菜单标题应使用多语言 key，而非硬编码文字。

✅ **路径前缀统一**
后端所有业务接口挂载在 `/v1/...`，管理接口统一 `/api/v1/admin/...`。

✅ **独立 schema**
后端数据库 schema 建议命名为 `px_com_powerx_<plugin>_<module>`。

✅ **权限与菜单联动**
菜单项可通过 `required_permissions` 字段绑定权限。

```yaml
menus:
  - id: "plugins.base.templates"
    title: "menu.base.templates"
    path: "/plugins/base/templates"
    required_permissions: ["base:template:read"]
```

---

## 六、示例清单结构（目录模式）

```
dist/
  0.1.0/
    plugin.yaml
    backend/bin/plugin
    web-admin/.output/
    README.md
```

---

## 七、验证与调试

验证清单结构：

```bash
make check-plugin
```

手动测试：

```bash
curl http://localhost:8080/_p/com.powerx.plugins.base/api/v1/admin/manifest
```

---

## 八、常见错误与排查

| 错误               | 原因                          | 解决方案                       |
| ---------------- | --------------------------- | -------------------------- |
| Plugin 启动失败      | `backend.entry` 路径错误        | 检查二进制文件是否存在                |
| /manifest 返回 404 | 插件未正确注册 admin 路由            | 检查 `router.go` 路由注册        |
| 前端访问空白页          | `webAdminPath` 缺失或未构建       | 执行 `make frontend-build`   |
| 权限未显示            | `/admin/rbac` 无返回或结构错误      | 返回需包含 `resource + actions` |
| Agent 未注册        | 未在 manifest 声明 agents/tools | 检查 YAML 与注册 API            |

---

## 下一步阅读

- 🔐 [RBAC Manifest 规范](./rbac_manifest_spec.md)
- 🤖 [Agent Contract 规范](./agent_contract.md)
- ⚙️ [上下文签名规范（HMAC / JWT）](./ctx_signing.md)
