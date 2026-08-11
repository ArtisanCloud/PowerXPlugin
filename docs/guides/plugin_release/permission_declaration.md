# PowerXPlugin 权限声明落地规范

本文是 PowerXPlugin 脚手架侧的开发者实施指南，用来解释 `px-plugin init` 生成项目应如何编写、检查和发布 `plugin.yaml` 的 `permissions[]`。

权威验收规范在 PowerX Core：

- `/private/var/www/html/ArtisanCloud/X/PowerX/Core/PowerX/docs/guides/plugin_release/permission_declaration.md`

PowerX Core 文档定义宿主安装、Capability Registry、IAM Permission、Gateway page/api 预检和角色授权的最终语义。本文只描述 PowerXPlugin skeleton、px-plugin 模板和生成项目的落地方式。若两边语义不一致，以 PowerX Core 文档为准，并同步更新本文和脚手架模板。

## 1. 适用范围

本文适用于：

- `skeleton/plugin.yaml`
- `skeleton/plugin.d/rbac.yaml`
- `scaffold/templates/plugin.yaml.tmpl`
- `tools/cli/internal/templates/data/plugin.yaml.tmpl`
- 通过 `px-plugin init com.powerx.plugins.hello-world` 生成的新插件项目

相关自动化：

```bash
make plugin-permission-declaration-check
make dist
make package-pxp
npm run sync:templates -- --check
```

`make dist` 和 `make package-pxp` 已强制执行 `plugin-permission-declaration-check`。权限声明不合规时，插件包不应被构建或发布。

API 授权键统一使用 `effective_permission_code`：

```text
effective_permission_code =
  business_permission_code 非空 -> business_permission_code
  否则 independent=true -> api.permission_code
  否则声明无效，安装 / 同步 / make dist 检查必须失败
```

PowerX Gateway 预检、插件前端按钮判断和插件后端二次校验必须使用同一个 effective 权限。`*_api:*` 这类 raw API permission 默认只是接口 binding 的技术登记键，不作为业务动作授权键，除非该 API 显式 `independent: true`。

## 2. 与 PowerX Core 规范的关系

PowerXPlugin 只负责把声明写正确、检查正确、打包正确。PowerX Core 负责安装时读取声明并同步到统一权限系统。

| 关注点 | PowerX Core 文档 | PowerXPlugin 本文 |
|---|---|---|
| 权限声明语义 | 权威定义 | 按脚手架模板落地 |
| 安装/同步行为 | Capability Sync Worker、IAM、Gateway | 只做打包前检查 |
| 角色授权入口 | PowerX 统一角色权限页 | 不在插件内实现正式授权 |
| page/api binding 解释 | 宿主验收标准 | 生成项目应该怎么写路径 |
| 自动化检查 | 安装/同步时校验 | `make plugin-permission-declaration-check` |

## 3. 插件 ID、目录名和权限码

示例插件统一使用：

```text
Plugin ID: com.powerx.plugins.hello-world
Target directory: com.powerx.plugins.hello-world
```

两者可以默认相同，但含义不同：

- `Plugin ID` 写入 `plugin.yaml.id`，用于运行时注册、宿主路由、能力归属和安装识别。
- `Target directory` 是本地项目目录名，只影响文件系统位置。

权限码不能直接使用带中划线的插件 ID 作为前缀，因为 PowerX 权限码 schema 要求 `module.resource:action` 的 module/resource 片段使用小写字母、数字和下划线。脚手架使用 `PluginDBName` 风格前缀：

```yaml
permission_code: com_powerx_plugins_hello_world.console:read
```

不要写成：

```yaml
permission_code: com.powerx.plugins.hello-world.console:read
```

## 4. Manifest 分片布局

脚手架生成的插件默认使用分片 catalog：

```yaml
catalogs:
  rbac: ./plugin.d/rbac.yaml
```

因此必须按 PowerX Core `7.1.1 主 Manifest 与分片 Catalog` 的规则放置字段：

- 主 `plugin.yaml` 只登记 `catalogs.rbac` 路径。
- `permissions[]`、`rbac`、`routes` 必须统一写在 `plugin.d/rbac.yaml`。
- 主 `plugin.yaml` 不得再重复声明顶层 `permissions:`、`rbac:` 或 `routes:`。

如果违反该规则，PowerX 安装器会拒绝安装：

```text
catalog conflict on field "permissions" (catalog=rbac)
```

只有不使用 `catalogs.rbac` 的简单插件，才可以把 `permissions[]` 直接写在主 `plugin.yaml`。

## 5. 最小声明结构

生成项目必须在有效 manifest 中声明非空 `permissions[]`。脚手架分片模式下，有效位置是 `plugin.d/rbac.yaml`。

最小后台页面插件示例：

```yaml
# plugin.d/rbac.yaml
permissions:
  - type: menu
    permission_code: com_powerx_plugins_hello_world.console:view_menu
    module: com_powerx_plugins_hello_world
    title_i18n:
      zh-CN: Hello World 插件菜单
      en: Hello World plugin menu
    description_i18n:
      zh-CN: 允许查看 Hello World 插件后台菜单入口。
      en: Allows viewing the Hello World plugin admin menu entries.
    risk_level: low
    data_scope: tenant
    default_role_grants:
      - role_owner
      - role_admin

  - type: page
    permission_code: com_powerx_plugins_hello_world.console:read
    module: com_powerx_plugins_hello_world
    title_i18n:
      zh-CN: Hello World 插件后台页面读取
      en: Read Hello World plugin admin pages
    description_i18n:
      zh-CN: 允许访问 Hello World 插件后台页面。
      en: Allows access to Hello World plugin admin pages.
    risk_level: low
    data_scope: tenant
    default_role_grants:
      - role_owner
      - role_admin
    protocol_bindings:
      - channel: rest
        method: GET
        path: /intro
        actor_context: admin_user
        resource_scope: tenant
```

## 6. page binding 路径规则

`type: page` 必须声明 GET `protocol_bindings`。binding path 写插件内部业务页面路径，不写宿主挂载路径。

正确：

```yaml
path: /intro
path: /templates
path: /templates/develop
path: /admin/templates/framework-lab
path: /powerx/knowledge-lab
```

错误：

```yaml
path: /_p/com.powerx.plugins.hello-world/admin/intro
path: /api/v1/intro
path: /v1/intro
```

原因：

- `/_p/<plugin_id>/admin` 是 PowerX 宿主挂载前缀，不是插件声明的内部页面。
- `/api/v1` 和 `/v1` 是接口前缀或版本前缀，不属于 page binding。
- PowerX Gateway 会按插件 ID、HTTP method 和内部 path 解析权限。

每个 `frontend.admin.menus[].path` 指向的后台页面，都必须被某个 `type: page` 的 GET binding 覆盖。静态资源、`/_nuxt/**`、`/assets/**`、health/debug bridge 不需要声明。

## 7. api binding 规则

`type: api` 必须声明 `protocol_bindings`，并按固定规则解析 `effective_permission_code`：

- 指向业务权限：`business_permission_code: <action permission_code>`
- 声明独立授权边界：`independent: true`

解析规则：

| API 声明 | effective 权限 | 适用场景 |
|---|---|---|
| `permission_code=com_powerx_plugins_hello_world.template_api:create` + `business_permission_code=com_powerx_plugins_hello_world.template:create` | `com_powerx_plugins_hello_world.template:create` | API 只是业务动作的技术入口。 |
| `permission_code=com_powerx_plugins_hello_world.audit_api:export` + `independent: true` | `com_powerx_plugins_hello_world.audit_api:export` | API 本身是独立授权边界。 |
| `permission_code=com_powerx_plugins_hello_world.template_api:read` 且无 `business_permission_code`、无 `independent` | 无效声明 | 必须补 `business_permission_code` 或显式 `independent: true`。 |

如果 `business_permission_code` 非空，它必须引用同一 `permissions[]` 中已声明的 `type: action` 权限。即使同时写了 `independent: true`，effective 权限仍以 `business_permission_code` 为准。

推荐写法：

```yaml
permissions:
  - type: action
    permission_code: com_powerx_plugins_hello_world.template:create
    module: com_powerx_plugins_hello_world
    title_i18n:
      zh-CN: 创建模板
      en: Create template
    description_i18n:
      zh-CN: 允许创建模板。
      en: Allows creating templates.
    risk_level: medium
    data_scope: tenant

  - type: api
    permission_code: com_powerx_plugins_hello_world.template_api:create
    business_permission_code: com_powerx_plugins_hello_world.template:create
    module: com_powerx_plugins_hello_world
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

不要把 API path 写成 `/api/v1/templates` 或 `/v1/templates`。脚手架声明的是插件内部 API path，宿主前缀由 PowerX 处理。

旧 `rbac.resources` 和 `routes.permissions` 只服务插件本地 RBAC 或历史兼容，不能生成 PowerX Gateway 正式接口授权。只要 `plugin.d/rbac.yaml.routes.permissions` 中声明了某个 HTTP route，就必须有对应的 `type: api` + `protocol_bindings` 覆盖同一个 method/path。

运行时合同入口和调试入口不能靠路径推断成业务权限：

- `/admin/runtime/ws-bus/grant`
- `/admin/runtime/ws-bus/publish`
- `/admin/runtime/ws-bus/test-flow`

如需保留这些入口，必须显式声明 `type: api`，并使用 `independent: true` 作为独立基础设施授权边界；生产调试入口应使用更高 `risk_level`。

## 8. taskbus/ws-bus topic ACL 规则

页面/API RBAC 与 taskbus/ws-bus topic ACL 是两条不同边界。`effective_permission_code` 只解决 Gateway、前端和插件后端二次校验；插件服务进程使用 STS 向底座发布 topic 时，底座看到的主体是插件 principal。

凡是 `plugin.d/events.yaml` 中声明 `actions` 包含 `publish` 的 topic，`config/event_fabric.yaml` 中对应 topic 的 `acl` 必须包含插件服务态 publish 授权：

```yaml
acl:
  - principal_type: plugin
    principal_id: "plugin:com.powerx.plugins.hello-world"
    actions: [publish]
```

在 px-plugin 模板里，`principal_id` 会随插件 ID 替换为：

```yaml
principal_type: plugin
principal_id: "plugin:{{ .PluginID }}"
actions: [publish]
```

`principal_type: member` + `principal_id: "member:system"`、`principal_type: role` + `principal_id: "role:role_admin"` 只代表成员或角色主体，不代表插件服务态 principal。缺少插件 principal publish ACL 时，`make dist` 必须失败，不能等插件运行后在 PowerX 底座收到 `403 topic not allowed`。

插件通过 STS/Bearer 调用 PowerX Core 的 HTTP 接口时，还必须满足 PowerX Core 的 STS direct route policy。以 host Scheduler 为例，插件入口 `/api/v1/admin/runtime/scheduler/jobs` 的 page/api RBAC 只保护插件自身 API；插件后端再调用底座 `/api/v1/admin/scheduler/jobs` 时，PowerX Core 必须显式允许该 route 被插件服务态 STS 访问。若底座未放行，会返回 `sts token not allowed for this route`，这不是 topic ACL 问题，也不能通过给 `member:system` 或 `role:role_admin` 授权解决。

## 9. 本地检查

开发过程中可以单独运行：

```bash
make plugin-permission-declaration-check
```

它会检查：

- 有效 manifest 位置的 `permissions[]` 必须存在且非空
- 使用 `catalogs.rbac` 时，主 `plugin.yaml` 不能声明 `permissions`、`rbac`、`routes`
- 每个权限声明必须有 `permission_code`
- `title_i18n` 和 `description_i18n` 至少包含一个非空 locale
- `risk_level` 必须是 `low`、`medium`、`high`、`critical`
- `module` 必须存在且非空
- `data_scope` 必须是 `tenant`、`global`、`system`
- `default_role_grants` 如存在，必须是角色代码字符串数组
- `type: page` 必须有 GET `protocol_bindings`
- `type: api` 必须有 `protocol_bindings`
- `type: api` 必须解析出 `effective_permission_code`
- 非空 `business_permission_code` 必须引用已声明的 `type: action` 权限
- `business_permission_code` 为空时，只有 `independent: true` 才能使用 API 自身 `permission_code` 作为 effective 权限
- `routes.permissions` 中的每个 method/path 必须被某个 `type: api` binding 覆盖
- 同一个 API method/path 不能重复绑定到多个不同 effective 权限
- binding path 不能包含 `/_p/<plugin_id>`、`/api/v1`、`/api`、`/v1`
- 菜单会打开的后台页面必须被 page binding 覆盖
- `plugin.d/events.yaml` 中声明 publish 的 topic，必须在 `config/event_fabric.yaml` 中给 `plugin:<plugin_id>` 授权 publish

发布前运行：

```bash
make dist
make package-pxp
```

这两个目标都会自动执行权限声明检查。不要绕过该 gate 手动打包。

## 9. 安装后验证

PowerXPlugin 侧只能保证包内声明正确；安装后是否登记成功，需要在 PowerX Core 侧验证。权威流程见 PowerX Core 文档 `7.4 安装后验证 PowerX 登记结果`。

推荐检查顺序：

1. `make dist` 已通过。
2. 插件安装或同步完成。
3. PowerX Core 插件同步日志没有 `capability.catalog.sync_failed`。
4. PowerX IAM 插件权限目录能查到该插件权限。
5. 页面访问或 API 调用未授权时返回明确 403，而不是页面空白或无菜单。

PowerX Admin API 示例：

```bash
export POWERX_BASE_URL="http://127.0.0.1:8077/api/v1"
export ADMIN_TOKEN="<admin-jwt>"
export PLUGIN_ID="com.powerx.plugins.hello-world"

curl -sS -H "Authorization: Bearer $ADMIN_TOKEN" \
  "$POWERX_BASE_URL/admin/iam/permissions/plugin-catalog?plugin_id=$PLUGIN_ID" | jq .
```

如果目录为空，优先检查：

- `plugin.yaml` 是否打进 dist。
- `permissions[]` 是否在 dist 内的有效位置存在；脚手架分片模式下应在 `plugin.d/rbac.yaml`。
- page/api binding path 是否误写宿主前缀。
- PowerX Core 同步任务是否拒绝了该插件。

## 10. 管理员授权和默认角色

插件只声明权限颗粒度，不在插件内做正式角色授权。正式授权入口始终是 PowerX 统一角色权限页。

`default_role_grants` 只是安装/同步时的默认角色建议：

```yaml
default_role_grants:
  - role_owner
  - role_admin
```

只有插件确实面向普通成员默认开放时，才加入：

```yaml
default_role_grants:
  - role_user
```

不要通过 SQL 或插件本地设置页长期绕过 PowerX IAM 授权。

## 11. 插件运行时消费

前端和后端必须使用同一批 `effective_permission_code`：

- 前端只按 effective 权限做菜单、页面、按钮显隐，不能作为唯一安全边界。
- 后端按接口 binding 解析出的 effective 权限对敏感 API 做二次校验。
- local 模式和 delegated 模式必须输出同结构的授权快照。
- 旧粗权限只用于迁移报告，不作为运行时 alias。
- delegated 模式仍必须加载显式 route permission 表，并按 PowerX 下发的授权快照和 effective 权限二次校验。
- health、静态资源、runtime contract、debug/test-flow 入口必须显式排除或独立授权，不得靠路径推断。

授权快照结构应与 PowerX Core 文档一致：

```json
{
  "permission_codes": [
    "com_powerx_plugins_hello_world.console:read",
    "com_powerx_plugins_hello_world.template:create"
  ],
  "policy_version": "2026-08-11T10:00:00Z",
  "perms_hash": "sha256:..."
}
```

插件代码不得只依赖：

- 菜单是否可见
- 前端按钮是否隐藏
- 旧粗权限，例如 `operations.order:read/manage`
- 插件设置页内的本地正式授权

## 12. 验收标准

发布前必须满足：

- 有效 manifest 位置存在 `permissions[]`；脚手架分片模式下应在 `plugin.d/rbac.yaml`。
- 使用 `catalogs.rbac` 时，主 `plugin.yaml` 不存在顶层 `permissions:`、`rbac:`、`routes:`。
- 每个用户可访问的插件后台业务页面都有 `type: page` + GET binding。
- 每个敏感接口都有 `type: api` binding，并映射到业务 `action` 或显式 `independent: true`。
- `routes.permissions` 中的每个 method/path 都有正式 `type: api` binding。
- `permission_code` 符合 `module.resource:action` 风格，脚手架项目使用 `PluginDBName` 前缀。
- 用户可见文案来自 `title_i18n` 和 `description_i18n`。
- `risk_level`、`data_scope`、`actor_context`、`resource_scope` 明确。
- `make plugin-permission-declaration-check`、`make dist` 通过，且每个 API binding 都能解析出唯一 effective 权限。
- PowerX Core `plugin-catalog` 能查到插件权限。
- 未授权用户访问页面或接口返回明确 403。
- local 模式和 delegated 模式输出同结构的 `permission_codes/policy_version/perms_hash`。
- delegated 模式插件后端仍加载显式 route permission 表，并按 PowerX 授权快照做二次校验。
- health、静态资源、runtime contract、debug/test-flow 入口已被明确排除或独立授权。

## 13. 回滚与风险控制

权限声明变更属于授权面变更。发布前必须确认：

- 新增页面是复用已有 read 权限，还是需要新授权。
- 新增接口是业务动作入口，还是独立业务授权边界。
- 删除或重命名 `permission_code` 会影响既有角色授权，应提供迁移说明。

回滚建议：

1. 插件版本回滚到上一版。
2. PowerX Core 重新同步上一版权限目录。
3. 保留审计中的授权变更记录。
4. 对旧粗权限缺口输出迁移报告，而不是长期保留 alias。

## 14. 修改模板后的同步要求

如果修改了 `skeleton/plugin.yaml`、`skeleton/make-files/**`、`skeleton/scripts/manifest/**` 或本文档对应的 skeleton 版本，必须同步模板：

```bash
npm run sync:templates
npm run sync:templates -- --check
```

至少确认以下位置一致：

- `scaffold/templates/**`
- `tools/cli/internal/templates/data/**`
- 新生成项目中的 `plugin.yaml`
- 新生成项目中的 `scripts/manifest/check-permission-declaration.mjs`
- 新生成项目中的 `docs/guides/plugin_release/permission_declaration.md`

## 15. 常见错误

### catalog conflict on field "permissions"

如果旧 `manifestcheck` 报：

```text
manifestcheck: invalid_manifest: catalog conflict on field "permissions" (catalog=rbac)
```

说明主 `plugin.yaml` 和 `plugin.d/rbac.yaml` 同时声明了 `permissions`。当前脚手架按 PowerX Core `7.1.1` 执行：使用 `catalogs.rbac` 时，`permissions[]`、`rbac`、`routes` 必须统一放在 `plugin.d/rbac.yaml`。

### 页面安装后不显示

优先检查：

1. `plugin.d/rbac.yaml` 是否有 `permissions[]`；未使用 `catalogs.rbac` 时才检查主 `plugin.yaml`
2. 菜单 path 是否有对应 `type: page` GET binding
3. binding path 是否误写了 `/_p/<plugin_id>/admin`
4. `make plugin-permission-declaration-check` 是否通过
5. PowerX Core 插件同步日志是否有权限声明校验失败

### 页面能打开但接口返回 Gateway 403

优先检查：

1. `plugin.d/rbac.yaml.routes.permissions` 是否只保留了旧 route RBAC。
2. 同一个 method/path 是否有 `permissions[].type: api` 的 `protocol_bindings`。
3. `api.business_permission_code` 是否指向业务 action，或是否显式 `independent: true`。
4. binding path 是否误写了 `/_p/<plugin_id>` 或 `/api/v1`。

### 权限码里用了中划线

错误：

```yaml
permission_code: com.powerx.plugins.hello-world.console:read
```

正确：

```yaml
permission_code: com_powerx_plugins_hello_world.console:read
```

插件 ID 可以包含中划线；权限码前缀不直接复用插件 ID。

## 16. 变更记录

| 日期 | 版本 | 变更 |
|---|---|---|
| 2026-08-11 | v1.1 | 对齐 PowerX Core 更细权限声明指南，补充安装后验证、管理员授权、运行时消费、验收标准和回滚风险控制。 |
| 2026-08-11 | v1.0 | 新增 PowerXPlugin 脚手架权限声明落地规范。 |
