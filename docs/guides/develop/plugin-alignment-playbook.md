# PowerX 插件对齐作战手册（鉴权 / RBAC / 安装 / CI）

适用对象：所有基于 PowerXPlugin skeleton 的插件项目。  
目标：让“功能迭代后即可安装可用、CI 稳定通过、线上链路不跑偏”。

---

## 1. 今日已验证的关键规范（必须统一）

### 1.1 身份鉴权路径规范

- 身份接口统一走：`/api/v1/admin/{identity}/auth/*`
- 当前默认 identity：`user`，即：`/api/v1/admin/user/auth/*`
- 不再使用旧路径：`/admin/auth/*`

### 1.2 插件网关路径边界

- 插件业务 API 走：`/_p/{plugin_id}/api/v1/*`
- 身份认证接口必须走宿主主路由：`/api/v1/admin/{identity}/auth/*`
- 不要把身份接口放到 `/_p/...` 链路里做最终鉴权。

### 1.3 RBAC 资源命名与路由推导一致

- `GET/POST /templates` 这类路由，网关按资源 `template` 推导。
- capability 与 catalog 中的 `rbac.resource` 必须与之一致（`template`），避免 `403 no permission rule`。

---

## 2. plugin.yaml / catalog 必备配置

### 2.1 `plugin.yaml` 必须包含 `migrations`

示例（已在 skeleton 对齐）：

```yaml
migrations:
  driver: go
  entry: backend/bin/migrate
  args: ["setup"]
  workdir: ./backend
  once: true
  timeout: 60s
```

目的：保证安装后可自动执行迁移，避免业务接口因表/字段缺失报 `500`。

### 2.2 `catalogs` 由能力源自动同步

- `contracts/capabilities/*.yaml` 是能力单一事实源。
- `skeleton/plugin.d/{capabilities,exposure,rbac}.yaml` 由同步脚本产出，不建议手写漂移。

---

## 3. 本地开发标准流程（推荐）

在仓库根目录执行：

```bash
make manifest-align-fix
make skeleton-reinstall VERSION=<new_version> API_BASE=http://127.0.0.1:8077/api/v1 TOKEN=$ADMIN_BEARER_TOKEN
```

说明：

- `manifest-align-fix`：自动同步 plugin.d 并校验 capability→exposure/rbac 映射。
- `skeleton-reinstall`：disable -> force install -> switch version(enable=true)。
- 每次涉及清单、权限、路由契约变更时，请递增版本号再重装。

---

## 4. CI 建议接入（必须）

### 4.1 严格门禁

```bash
make manifest-align-check
```

失败即阻断（说明 catalog 漂移或映射不一致）。

### 4.2 devwatch 测试稳定性

`tools/cli/internal/devwatch` 相关测试应显式设置：

- `PX_RESOURCE_CPU_THRESHOLD=101`（测试内设置）

避免 CI 环境 CPU guard 误触发导致超时。

---

## 5. 故障快速判定

- `403 no permission rule`：先查 capability `rbac.resource/actions` 与 `plugin.d/exposure.yaml` 映射。
- `401 Unauthorized`：先查 token 受众/链路（宿主用户 token vs 插件 token）。
- `500 Internal Server Error`：先查插件后端日志、schema、迁移执行状态。

数据库核对（PostgreSQL）：

```sql
select to_regclass('px_com_powerx_plugins_base.template');
select column_name from information_schema.columns
 where table_schema='px_com_powerx_plugins_base' and table_name='template';
```

---

## 6. 相关命令清单

- 自动对齐（本地）：`make manifest-align-fix`
- 严格校验（CI）：`make manifest-align-check`
- 清单校验：`make plugin-yaml-check`
- 重装插件：`make skeleton-reinstall VERSION=<v> API_BASE=... TOKEN=...`

