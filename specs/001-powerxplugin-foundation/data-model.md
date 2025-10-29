# 数据模型 - PowerXPlugin 仓库基线

## 1. 插件 Manifest

- **标识**: `pluginId`（字符串，必须与目录/路由前缀保持一致，例如 `com.powerx.note`）
- **版本信息**: `version`（语义化版本，默认 `0.1.0`，CLI 初始化时写入）
- **菜单定义**: `menus[]`
  - `path`（字符串，需以 `/_p/<plugin-id>/admin/` 为前缀）
  - `title`（字符串，多语言预留）
  - `icon`（可选，字符串）
  - `permission`（字符串，对应 RBAC 权限键）
- **权限声明**: `permissions[]`
  - `key`（字符串，命名约定 `<plugin-id>.<domain>.<action>`）
  - `description`（字符串，说明用途）
- **健康检查依赖**: `healthChecks[]`（可选，引用框架标准探针）

> 关系：Manifest 与 RBAC 权限键一一对应，菜单中引用的权限必须存在于 `permissions[]`。

## 2. RBAC 权限报告

- **标识**: `pluginId`
- **角色定义**: `roles[]`
  - `name`（字符串）
  - `description`（字符串）
  - `permissions[]`（引用 Manifest 中的权限键）
- **默认绑定**: `defaultAssignments[]`（可选，指明宿主角色默认映射）

> 关系：`roles[].permissions[]` 只能引用 Manifest 中声明的权限；框架在启动时通过 `rbac.Report(app)` 上报。

## 3. 健康检查 & 运行态

- **服务端点**:
  - `GET /_p/<plugin-id>/api/v1/ping` → 结构 `{ status: "ok" }`
  - `GET /_p/<plugin-id>/healthz` → 复用框架健康探针
- **状态**:
  - `Standalone`：直接监听 `Config.Listen`，主要用于本地开发/CI。
  - `Hosted`：由宿主代理，路由仍需保持 `/_p/<plugin-id>/api/**` 前缀。

> 关系：`bootstrap.Config.Standalone` 决定路由公开方式；CLI README 需告知切换方法。

## 4. 前端 Layer 配置

- **Nuxt 配置**: `definePowerXAdminConfig()`
  - `pluginId`
  - `starterPages`（布尔，控制是否启用默认页面）
  - `baseURL`（由框架强制设置，开发者无需修改）
- **组件覆盖**:
  - 文件路径遵循 Nuxt Layer 规则，同路径同名文件直接覆盖框架实现。
- **API 客户端**: `usePluginApi({ pluginId })`
  - 自动拼接 `/_p/<plugin-id>/api/v1/` 前缀
  - 返回 `fetch` 包装器，支持标准 REST 方法

> 关系：Manifest 菜单路径必须与 Nuxt 页面路径一致，保证路由/权限对齐。
