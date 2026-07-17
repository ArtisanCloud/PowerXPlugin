# PowerXPlugin 安装到 PowerX 指南

本文是 PowerXPlugin 安装到 PowerX Core 的统一入口。目标是验证真实安装链路，不是本地开发热更新。

## 1. 先区分两种模式

| 模式 | 用途 | 入口 | 端口 |
| --- | --- | --- | --- |
| Standalone / dev | 本地开发、调试插件后端和前端 | `make dev`、`npm run dev`、`px-plugin dev --watch` | 开发者指定，例如 backend `8078`、web-admin `3131` |
| PowerX 安装态 | 模拟生产安装、启用、切换版本、验证 `/_p/<plugin_id>` 代理 | `make dist`、`make local-install`、`POST /admin/plugins/install/local` | 由 PowerX Core 动态分配 |

安装态不使用固定 backend 端口。`8078`、`8086` 这类端口只能用于本地 standalone/dev，不允许写进可安装包的 `plugin.yaml`。

## 2. 安装态端口规范

可安装包的 `plugin.yaml` 必须使用动态端口占位符：

```yaml
runtime:
  kind: process
  entry: backend/bin/plugin
  env:
    POWERX_BIND_ADDR: ":__POWERX_DYNAMIC_PORT__"
    POWERX_PLUGIN_ID: com.powerx.plugins.base
    POWERX_PLUGIN_REGISTRATION_MODE: installed
    POWERX_PROXY: "1"
    POWERX_PROVIDER_MODE: delegated
backend:
  entry: backend/bin/plugin
  port: 0
  health: /healthz
```

PowerX Core 启用插件时会：

1. 分配一个空闲端口。
2. 注入 `POWERX_HTTP_ADDR=127.0.0.1:<dynamic_port>`。
3. 启动插件进程。
4. 对 `/healthz` 做健康检查。
5. 将 `/_p/<plugin_id>/api/*` 反代到插件进程。

`POWERX_PLUGIN_REGISTRATION_MODE=installed` 是安装态必填项。否则插件 Agent/Skill/Capability 可能被注册成 `.local` 身份，导致 Core 调用 `/_p/<plugin_id>.local/...` 时被租户插件启用校验拒绝。

## 3. 构建可安装包

在插件仓库根目录执行：

```bash
cd /absolute/path/to/<plugin-repo>

make plugin-yaml-check
make dist
```

输出目录：

```text
dist/<version>/
  plugin.yaml
  plugin.d/
  skills/              # 可选：仅当插件声明运行态 Skill 时必须包含
  backend/bin/plugin
  backend/bin/migrate
  web-admin/.output/
  web-admin/i18n/
```

`make dist` 会校验：

- `plugin.yaml` 必须包含 `POWERX_BIND_ADDR: ":__POWERX_DYNAMIC_PORT__"`。
- `plugin.yaml` 不得固化旧 backend port，例如 `8078`、`8086`。
- 如插件声明运行态 Skill，`skills/` 里必须包含标准 `SKILL.md` 包；未声明 Skill 的插件不得为了通过构建放空目录兜底。
- `plugin.d/rbac.yaml` 必须包含必要的 runtime routes。

PowerXPlugin 框架仓库的 `make dist` 当前代理到 `skeleton/dist`，`make skeleton-dist` 仅作为兼容旧命令保留。独立插件仓库必须直接使用自身根目录的 `make dist`，安装目录指向该插件的 `dist/<version>`。

## 4. 安装到 PowerX Core

准备 PowerX Core 地址和 root/admin token：

```bash
export API_BASE=http://127.0.0.1:8077/api/v1
export TOKEN=<ADMIN_BEARER_TOKEN>
```

安装并启用：

```bash
make local-install API_BASE=$API_BASE TOKEN=$TOKEN ENABLE=true FORCE=false
```

等价 API：

```bash
curl -X POST "$API_BASE/admin/plugins/install/local" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "src_dir": "/absolute/path/to/<plugin-repo>/dist/<version>",
    "enable": true,
    "force": false
  }'
```

重装当前插件并切换版本：

```bash
make local-reinstall VERSION=<version> API_BASE=$API_BASE TOKEN=$TOKEN
```

该命令会执行：

1. `POST /admin/plugins/<plugin_id>/disable`
2. `POST /admin/plugins/install/local`，`enable=false`，`force=true`
3. `POST /admin/plugins/<plugin_id>/switch_version`，`enable=true`

生产或准生产环境建议优先使用“安装不启用 -> 验证 -> switch_version”的节奏，不要直接覆盖旧版本。

## 5. 安装后验证

检查插件列表和状态：

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "$API_BASE/admin/plugins"

curl -H "Authorization: Bearer $TOKEN" \
  "$API_BASE/admin/plugins/com.powerx.plugins.base/status"
```

检查 Core 动态代理是否挂载：

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "http://127.0.0.1:8077/__debug/plugins"
```

期望能看到：

```json
{
  "apis": {
    "com.powerx.plugins.base": {
      "target": "http://127.0.0.1:<dynamic_port>",
      "basePath": "/api/v1",
      "healthPath": "/healthz"
    }
  }
}
```

检查租户是否启用插件：

```sql
select id, tenant_uuid, plugin_id, key, enabled, status, updated_at
from plugin_instance_configs
where plugin_id = 'com.powerx.plugins.base'
order by updated_at desc;
```

Agent/Skill 测试前，进入插件后台重新执行：

```text
初始化/同步模板智能体
```

同步后的 ID 应该是安装态 ID：

```text
powerxplugin.template.basic
com.powerx.plugins.base.template.prepare
com.powerx.plugins.base.template.create
```

不应该出现：

```text
powerxplugin.template.basic.local
com.powerx.plugins.base.local.template.create
```

## 6. 常见问题

### 6.1 health check timeout

现象：

```text
healthcheck_failed ... /healthz
```

检查：

- `plugin.yaml` 是否使用 `POWERX_BIND_ADDR: ":__POWERX_DYNAMIC_PORT__"`。
- 插件启动日志里是否拿到 `POWERX_HTTP_ADDR=127.0.0.1:<dynamic_port>`。
- 插件后端是否实际监听该地址。
- `/healthz` 是否注册并返回 2xx。

### 6.2 TENANT_PLUGIN_DISABLED

现象：

```text
access denied at gateway
TENANT_PLUGIN_DISABLED
当前租户未启用该插件
```

常见原因：

- Agent/Skill/Capability 被同步成 `.local`，但本地插件 backend 没有向 Core 登记 `.local` 运行时。
- 当前租户没有 `plugin_instance_configs` 启用记录。
- 请求路径是 `/_p/com.powerx.plugins.base.local/...`，但实际安装插件是 `com.powerx.plugins.base`。

处理：

1. 确认安装包 `plugin.yaml` 设置 `POWERX_PLUGIN_REGISTRATION_MODE: installed`。
2. 如果是本地联调，确认插件 backend 以 `POWERX_PLUGIN_REGISTRATION_MODE=local`、`POWERX_PROXY=1`、`POWERX_PROVIDER_MODE=local` 启动，并且 backend 端口不是 Nuxt 端口。
3. 用 `curl -s http://127.0.0.1:8077/__debug/plugins | jq '.apis | keys'` 确认包含 `com.powerx.plugins.base.local`。
4. 重新执行“初始化/同步模板智能体”。
5. 新开 Agent session 测试。

### 6.3 registration mode invalid

现象：

```text
PLUGIN_REGISTRATION_MODE_INVALID
POWERX_PLUGIN_REGISTRATION_MODE must be one of installed or local
```

处理：

- 安装态设置 `POWERX_PLUGIN_REGISTRATION_MODE=installed`。
- 真正本地 `.local` 联调才设置 `POWERX_PLUGIN_REGISTRATION_MODE=local`；插件 backend 启动时会向 Core `/api/v1/internal/plugins/debug-hosts` 自动登记 `.local` 运行时并启用当前租户实例。

### 6.4 安装后菜单还是旧的

检查：

- 是否重新构建了 `web-admin/.output`。
- 是否安装的是最新 `skeleton/dist/<version>`。
- Core 是否仍运行旧进程。
- 浏览器是否缓存旧前端资源。

## 7. 相关文档

- `README.md`：仓库根入口与 `make skeleton-*` 命令。
- `skeleton/make-files/build.mk`：dist/release 构建与动态端口校验。
- `skeleton/make-files/release.mk`：`local-install`、`local-reinstall` 实现。
- `docs/guides/develop/agent-skill-bridge/README.md`：Agent/Skill/Capability 同步规范。
- `PowerX/docs/plan/deploy/plugin-upgrade-sop.md`：Core 侧插件安装和平滑升级 SOP。
- `PowerX/docs/standards/_shared/service-port-matrix.md`：本地开发端口矩阵。
