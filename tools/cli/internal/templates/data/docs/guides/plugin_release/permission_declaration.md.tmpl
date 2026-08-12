# 插件权限声明规范

本文是 `px-plugin init` 生成项目内的权限声明实施指南。PowerX Core 的权威验收规范位于：

- `/private/var/www/html/ArtisanCloud/X/PowerX/Core/PowerX/docs/guides/plugin_release/permission_declaration.md`

如果生成项目不在同一台机器或没有 PowerX Core checkout，请以团队 PowerX Core 仓库中的同名文档为准。本文只说明当前插件项目如何按该规范修改 `plugin.yaml` 并通过本地检查。

运行时 `manifestx.Plugin()` 只用于向 framework/router 层登记插件 ID、名称、版本和运行态菜单等基础信息，不是正式授权来源。正式权限只能来自有效 manifest 的 `permissions[]`；脚手架分片模式下就是 `plugin.d/rbac.yaml`。不得再在 `manifestx.Plugin().Permissions` 里维护旧三段式权限或另一套 `resource:action` 清单，framework 的 `manifest.Register()` 也不得用旧 runtime manifest permissions schema 阻断插件启动。

## 1. Manifest 分片布局

当前生成项目默认使用分片 catalog：

```yaml
catalogs:
  rbac: ./plugin.d/rbac.yaml
```

因此 `permissions[]`、`rbac`、`routes` 必须统一写在 `plugin.d/rbac.yaml`。主 `plugin.yaml` 不得再重复声明顶层 `permissions:`、`rbac:` 或 `routes:`。

如果安装时报：

```text
catalog conflict on field "permissions" (catalog=rbac)
```

说明主 `plugin.yaml` 和 `plugin.d/rbac.yaml` 同时声明了 `permissions`，需要把权限声明移动到 `plugin.d/rbac.yaml`。

只有不使用 `catalogs.rbac` 的简单插件，才可以把 `permissions[]` 直接写在主 `plugin.yaml`。

## 2. 必须声明 permissions[]

有效 manifest 位置必须有非空 `permissions[]`。脚手架分片模式下，有效位置是 `plugin.d/rbac.yaml`。后台插件至少应有：

- `type: menu`：插件菜单入口
- `type: page`：插件后台页面访问权限

示例使用 `com.powerx.plugins.hello-world` 作为插件 ID。插件 ID 只能作为 `plugin.yaml.id` 或 PowerX 侧 `source=plugin:<plugin_id>`，不能拼进 `permission_code/module/resource/action/menu_path`。

所有权限类型都必须提供面向管理员的展示文案：

- `type: menu` 必须有 `title_i18n` / `description_i18n`
- `type: page` / `type: action` / `type: data` 必须有 `title_i18n` / `description_i18n`
- `type: api` 必须有 `title_i18n` / `description_i18n`

API 的 `permission_code`、HTTP `method/path`、`/_p/<plugin_id>`、`/api/v1` 只能作为技术辅助信息，不能作为 PowerX 角色权限页主标题或说明。插件 API 权限的正式来源是插件包有效 manifest 内的 `permissions[]`；使用 `catalogs.rbac` 时必须写在 `plugin.d/rbac.yaml`。

权限码必须使用业务域 `module.resource:action`：

```yaml
# plugin.d/rbac.yaml
permission_code: template.template:manage
module: template
resource: template
action: manage
```

不要把带中划线的插件 ID 或插件 ID 派生命名写进权限码：

```yaml
permission_code: com.powerx.plugins.hello-world.console:read
permission_code: com_powerx_plugins_hello_world.console:read
```

菜单声明还必须和 `frontend.admin.menus` 做双向审计：

- 菜单项只要声明 `required_policies`，每个值都必须命中已声明的 `type=menu.permission_code`。
- 每个 `type=menu` 权限都必须被至少一个菜单项的 `required_policies` 引用；否则视为 invalid/orphan。
- `type=menu.menu_path` 必须等于对应菜单项在 `frontend.admin.menus` 中的真实 `id` 祖先链。
- `menu_path` 和菜单 `id` 都不能包含插件 ID、`/_p`、`/api/v1`、URL 或 API path 片段。

API 授权键统一使用 `effective_permission_code`：

```text
effective_permission_code =
  business_permission_code 非空 -> business_permission_code
  否则 independent=true -> api.permission_code
  否则声明无效，安装 / 同步 / make dist 检查必须失败
```

PowerX Gateway 预检、插件前端按钮判断和插件后端二次校验必须使用同一个 effective 权限。`*_api:*` 这类 raw API permission 默认只是接口 binding 的技术登记键，不作为业务动作授权键，除非该 API 显式 `independent: true`。

PowerX 角色权限页固定分两类视图：

- 菜单权限：只展示 `type=menu`，按 `menu_path` 渲染菜单树。
- 能力/API 权限：按“来源 -> 模块 -> 操作权限/API 权限”展示。第一层来源只能是 `PowerX 底座` 或具体插件，插件来源由 `plugin_id` / `iam_permission.source=plugin:<plugin_id>` 派生；`admin`、`settings`、`production` 等只能作为第二层 `module`，不能作为来源。`page/action/data` 归入操作权限，`api` 归入 API 权限。来源筛选必须是 `全部来源 / PowerX 底座 / 插件`，不能用“全部插件”同时显示底座权限。

## 3. menu/page binding

`type: menu` 必须声明菜单层级和联动页面权限：

```yaml
permissions:
  - type: menu
    permission_code: menu.console:view
    module: console
    menu_path:
      - console
    page_permission_codes:
      - console.page:read
    title_i18n:
      zh-CN: 后台菜单
      en: Admin menu
    description_i18n:
      zh-CN: 允许查看后台菜单入口。
      en: Allows viewing admin menu entries.
    risk_level: low
    data_scope: tenant
```

`type: page` 必须有 GET `protocol_bindings`：

```yaml
# plugin.d/rbac.yaml
permissions:
  - type: page
    permission_code: console.page:read
    module: console
    resource: page
    action: read
    title_i18n:
      zh-CN: Hello World 插件后台页面读取
      en: Read Hello World plugin admin pages
    description_i18n:
      zh-CN: 允许访问 Hello World 插件后台页面。
      en: Allows access to Hello World plugin admin pages.
    risk_level: low
    data_scope: tenant
    protocol_bindings:
      - channel: rest
        method: GET
        path: /intro
        actor_context: admin_user
        resource_scope: tenant
```

path 写插件内部页面路径：

```yaml
path: /intro
path: /templates
path: /templates/develop
path: /templates/crud
path: /admin/templates/framework-lab
path: /powerx/knowledge-lab
```

不要写宿主挂载路径或 API 前缀：

```yaml
path: /_p/com.powerx.plugins.hello-world/admin/intro
path: /api/v1/intro
path: /v1/intro
```

每个 `frontend.admin.menus[].path` 指向的后台页面，都必须被某个 `type: page` GET binding 覆盖。

## 4. api binding

`type: api` 必须有 `protocol_bindings`，并且必须解析出 `effective_permission_code`：

```yaml
business_permission_code: template.template:create
```

或：

```yaml
independent: true
```

解析规则：

| API 声明 | effective 权限 | 适用场景 |
|---|---|---|
| `permission_code=template.template_api:create` + `business_permission_code=template.template:create` | `template.template:create` | API 只是业务动作的技术入口。 |
| `permission_code=audit.audit_api:export` + `independent: true` | `audit.audit_api:export` | API 本身是独立授权边界。 |
| `permission_code=template.template_api:read` 且无 `business_permission_code`、无 `independent` | 无效声明 | 必须补 `business_permission_code` 或显式 `independent: true`。 |

如果 `business_permission_code` 非空，它必须引用同一 `permissions[]` 中已声明的 `type: page/action/data` 操作权限。即使同时写了 `independent: true`，effective 权限仍以 `business_permission_code` 为准。

API binding 的 path 同样写插件内部路径，不写 `/api/v1` 或 `/v1`。

推荐写法：

```yaml
permissions:
  - type: api
    permission_code: template.template_api:create
    business_permission_code: template.template:create
    module: template
    resource: template_api
    action: create
    title_i18n:
      zh-CN: 创建模板接口
      en: Create template API
    description_i18n:
      zh-CN: 允许调用创建模板接口。
      en: Allows calling the create template API.
    risk_level: medium
    data_scope: tenant
    protocol_bindings:
      - channel: rest
        method: POST
        path: /templates
        actor_context: admin_user
        resource_scope: tenant
```

错误写法：

```yaml
title_i18n:
  zh-CN: POST /templates
description_i18n:
  zh-CN: template.template_api:create
```

动态路径统一写 `*`，不得写 `{uuid}` 或 `:id`：

```yaml
path: /templates/*
```

旧 `rbac.resources` 和 `routes.permissions` 只服务插件本地 RBAC 或历史兼容，不能生成 PowerX Gateway 正式接口授权。只要 `plugin.d/rbac.yaml.routes.permissions` 中声明了某个 HTTP route，就必须有对应的 `type: api` + `protocol_bindings` 覆盖同一个 method/path。

宿主安装期 discovery 接口不是用户业务 API，不能写入 `permissions[].protocol_bindings`：

- `/plugin/skills`
- `/plugin/skills/*`
- `/plugin/skills/*/schema`

PowerX 安装/启用插件时会先调用这些接口读取插件元数据和 Skill 描述，此时插件能力尚未完成登记，也不存在某个用户请求携带的 `permission_codes/policy_version/perms_hash` 授权快照。插件实现必须把这些 host discovery 路由和业务/管理 API 分组隔离：它们不走用户态 JWT/RBAC，也不能返回业务数据或执行业务动作。需要管理员操作的插件智能体管理接口，例如 `/plugin/agent-registry/*`，仍然是普通管理 API，必须声明 `type: api` 并按 `effective_permission_code` 校验。

运行时合同入口和调试入口不能靠路径推断成业务权限：

- `/admin/runtime/ws-bus/grant`
- `/admin/runtime/ws-bus/publish`
- `/admin/runtime/ws-bus/test-flow`

如需保留这些入口，必须显式声明 `type: api`，并使用 `independent: true` 作为独立基础设施授权边界；生产调试入口应使用更高 `risk_level`。

## 5. taskbus/ws-bus topic ACL 规则

页面/API RBAC 与 taskbus/ws-bus topic ACL 是两条不同边界。`effective_permission_code` 只解决 Gateway、前端和插件后端二次校验；插件服务进程使用 STS 向底座发布 topic 时，底座看到的主体是插件 principal。

凡是 `plugin.d/events.yaml` 中声明 `actions` 包含 `publish` 的 topic，`config/event_fabric.yaml` 中对应 topic 的 `acl` 必须包含插件服务态 publish 授权：

```yaml
acl:
  - principal_type: plugin
    principal_id: "plugin:com.powerx.plugins.hello-world"
    actions: [publish]
```

px-plugin 模板会把 `principal_id` 替换为当前插件：

```yaml
principal_type: plugin
principal_id: "plugin:{{ .PluginID }}"
actions: [publish]
```

`member:system` 和 `role:role_admin` 只代表成员或角色主体，不代表插件服务态 principal。缺少插件 principal publish ACL 时，`make dist` 必须失败，不能等插件运行后在 PowerX 底座收到 `403 topic not allowed`。

插件通过 STS/Bearer 调用 PowerX Core 的 HTTP 接口时，还必须满足 PowerX Core 的 STS direct route policy。以 host Scheduler 为例，插件入口 `/api/v1/admin/runtime/scheduler/jobs` 的 page/api RBAC 只保护插件自身 API；插件后端再调用底座 `/api/v1/admin/scheduler/jobs` 时，PowerX Core 必须显式允许该 route 被插件服务态 STS 访问。若底座未放行，会返回 `sts token not allowed for this route`，这不是 topic ACL 问题，也不能通过给 `member:system` 或 `role:role_admin` 授权解决。

## 6. 本地检查

单独检查权限声明：

```bash
make plugin-permission-declaration-check
```

发布前检查：

```bash
make dist
make package-pxp
```

`make dist` 和 `make package-pxp` 会自动执行 `plugin-permission-declaration-check`，不合规时不会继续打包。

检查内容包括：

- 有效 manifest 位置的 `permissions[]` 必须存在
- 使用 `catalogs.rbac` 时，主 `plugin.yaml` 不能声明 `permissions`、`rbac`、`routes`
- 每个权限声明必须有 `permission_code`、`module`、`title_i18n`、`description_i18n`、`risk_level`、`data_scope`
- API 的 `title_i18n` / `description_i18n` 必须是用户可读文案，不能直接等于 `permission_code`、HTTP method/path、`/_p/<plugin_id>` 或 `/api/v1`
- `permission_code/module/resource/action/menu_path` 不能包含插件 ID 或插件 ID 派生命名
- `type: menu` 必须有非空 `menu_path` 和 `page_permission_codes`
- `frontend.admin.menus[].required_policies` 必须引用已声明的 `type=menu.permission_code`
- 每个 `type=menu` 必须被菜单引用，且 `menu_path` 必须等于菜单真实 `id` 祖先链
- `type: page/action/api` 必须有 `module/resource/action`
- `permission_code` 必须等于 `module.resource:action`
- `type: page` 必须有 GET `protocol_bindings`
- `type: api` 必须有 `protocol_bindings`
- `type: api` 必须解析出 `effective_permission_code`
- 非空 `business_permission_code` 必须引用已声明的 `type: page/action/data` 操作权限
- `business_permission_code` 为空时，只有 `independent: true` 才能使用 API 自身 `permission_code` 作为 effective 权限
- `routes.permissions` 中的每个 method/path 必须被某个 `type: api` binding 覆盖
- 同一个 API method/path 不能重复绑定到多个不同 effective 权限
- binding path 不能包含 `/_p/<plugin_id>`、`/api/v1`、`/api`、`/v1`
- binding path 动态段必须使用 `*`，不能使用 `{param}` 或 `:param`
- binding path 不能声明宿主安装期 discovery 路由 `/plugin/skills/**`
- 菜单页面必须被 page binding 覆盖
- `plugin.d/events.yaml` 中声明 publish 的 topic，必须在 `config/event_fabric.yaml` 中给 `plugin:<plugin_id>` 授权 publish

## 7. 安装后验证

PowerX Core 安装或同步插件后，应能在统一权限目录中看到插件权限。权威流程见 PowerX Core：

```text
/private/var/www/html/ArtisanCloud/X/PowerX/Core/PowerX/docs/guides/plugin_release/permission_declaration.md
```

推荐验证：

```bash
export POWERX_BASE_URL="http://127.0.0.1:8077/api/v1"
export ADMIN_TOKEN="<admin-jwt>"
export PLUGIN_ID="com.powerx.plugins.hello-world"

curl -sS -H "Authorization: Bearer $ADMIN_TOKEN" \
  "$POWERX_BASE_URL/admin/iam/permissions/plugin-catalog?plugin_id=$PLUGIN_ID" | jq .
```

预期结果：

- 能看到 `menu/page/action/api` 权限项。
- page/api 项包含 `protocol_bindings`。
- PowerX 角色权限页能按插件分组授权。
- 未授权访问页面或接口返回明确 403。

## 7. 运行时消费

插件前端和后端必须消费同一批 `effective_permission_code`：

- 前端用于菜单、页面、按钮显隐。
- 后端按接口 binding 解析出的 effective 权限用于敏感 API 二次校验。
- local 和 delegated 模式输出同结构授权快照，字段必须是 `permission_codes/policy_version/perms_hash/source`。
- 不使用旧粗权限作为长期 alias。
- delegated 模式仍必须加载显式 route permission 表，并按 PowerX 下发的授权快照和 effective 权限二次校验。
- health、静态资源、runtime contract、debug/test-flow 入口必须显式排除或独立授权，不得靠路径推断。

授权快照结构：

```json
{
  "permission_codes": [
    "console.page:read",
    "template.template:manage"
  ],
  "policy_version": "2026-08-11T10:00:00Z",
  "perms_hash": "sha256:...",
  "source": "signed_claims"
}
```

local 模式只能模拟 delegated 授权快照，`source` 必须是 `local_mock`，不得另维护一套正式授权定义。

## 8. 默认角色和回滚

`default_role_grants` 是安装/同步时的默认角色建议，不是插件内正式授权：

```yaml
default_role_grants:
  - role_owner
  - role_admin
```

删除或重命名 `permission_code` 会影响既有角色授权。发布前应确认是否需要迁移说明；回滚时应回滚插件版本并让 PowerX Core 重新同步上一版权限目录。

## 9. 排障

如果页面安装后不显示，先检查：

```bash
make plugin-permission-declaration-check
```

然后确认：

- `plugin.d/rbac.yaml` 有 `permissions[]`，或未使用 `catalogs.rbac` 时主 `plugin.yaml` 有 `permissions[]`
- 菜单 path 有对应 page binding
- page binding 没有写 `/_p/<plugin_id>/admin`
- PowerX Core 插件同步日志没有权限声明校验错误

如果看到：

```text
catalog conflict on field "permissions" (catalog=rbac)
```

说明主 `plugin.yaml` 和 `plugin.d/rbac.yaml` 同时声明了 `permissions`。当前脚手架要求 `catalogs.rbac` 分片模式下只在 `plugin.d/rbac.yaml` 声明 `permissions[]`，重新安装新版 `px-plugin` 后再生成项目。
