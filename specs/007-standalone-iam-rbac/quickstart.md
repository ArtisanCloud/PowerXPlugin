# Quickstart: Standalone 模式 IAM & RBAC

## 1. 初始化本地 IAM
```bash
cd skeleton/backend/go-gin
export POWERX_PROXY=0
export POWERX_RBAC_DELEGATE=false
export PLUGIN_IAM_TENANT_KEY=00000000-0000-0000-0000-000000000001
export PLUGIN_IAM_TENANT_NAME="Local Tenant"
export PLUGIN_IAM_ADMIN_EMAIL=admin@local.test
export PLUGIN_IAM_ADMIN_PASSWORD='S3cret!!'
# 可选：覆盖 token 元数据，方便与宿主比对
export POWERX_PLUGIN_ID="com.powerx.plugins.base"
export PLUGIN_IAM_POLICY_VERSION="local.v1"
go run ./cmd/database/main.go setup
```
- `setup`=迁移+种子，日志出现 `iam tables migrated` 与 `local admin seeded` 即成功。
- Delegated 模式切换：设置 `POWERX_PROXY=1` 或 `POWERX_RBAC_DELEGATE=true`，本地 IAM 菜单自动隐藏。
- 登录/刷新接口会返回 `plugin_id` 与 `policy_version` 字段，可用于排查 Manifest 版本与 Token 来源。

## 2. 启动后端与前端
```bash
# Backend
cd skeleton/backend/go-gin
go run ./cmd/plugin

# Frontend admin
cd ../web-admin
npm install
npm run dev
```
- Standalone 模式访问 `http://localhost:3031/admin/iam/overview`。
- Delegated 模式访问 `_p/<plugin-id>/admin`，IAM 菜单不可见。

### 2.1 UI 对齐自检（与 PowerX settings* 页面一致）
对照宿主仓库 `Core/PowerX/web-admin/app/pages/settings/{index,users,roles,config}`，确认以下要点：

- **入口导航**：`/admin/iam/overview` 需要出现与 `settings/index.vue` 相同的卡片导航+概览副标题；quick links 里的“组织结构”“角色管理”“租户配置”指向 Standalone 页面。
- **组织管理 Tab**：`/admin/iam/members` 的 Tab 组合/显隐与 `settings/users/index.vue` 保持一致——部门/用户/权限三栏，并根据 root/tenant admin 权限动态出现 “权限” Tab。
- **角色面板**：角色列表、筛选、分页、远程租户选择以及“克隆/编辑”抽屉的布局与 `components/settings/users/RoleManager.vue` 对齐；scope、描述、权限树勾选行为一致。
- **租户/配置表单**：`/admin/iam/overview` 中的 Plan Drawer + 创建租户弹窗，以及 `/admin/iam/tenants`（若拆页）上的基础属性/功能开关段落，与 `settings/config` 的“快速设置”风格保持一致（含表头、描述、按钮排布）。

> 若 UI 差异较大，记录截图并在 PR 描述中说明原因；文档默认参考 PowerX settings 页面作为 UX baseline。

## 3. Playwright 验证
```bash
# Delegated 登录（需宿主）
npm --prefix skeleton/web-admin/nuxt run test:e2e -- auth-delegated

# Local IAM 流程
PLAYWRIGHT_LOCAL_IAM=1 \
PLAYWRIGHT_LOCAL_EMAIL=admin@local.test \
PLAYWRIGHT_LOCAL_PASSWORD='S3cret!!' \
npm --prefix skeleton/web-admin/nuxt run test:e2e -- auth-local

# IAM 管理新增用例（完成后）
PLAYWRIGHT_LOCAL_IAM=1 npm --prefix skeleton/web-admin/nuxt run test:e2e -- iam-local

# Delegated 提示（验证本地入口隐藏）
PLAYWRIGHT_LOCAL_IAM=0 npm --prefix skeleton/web-admin/nuxt run test:e2e -- auth-local
```

## 4. CLI 工具
`px-plugin iam` 命令用于离线备份与管理员重置，默认读取 `--entry`（插件根目录）的 `backend/etc/config.yaml` 连接数据库。

```bash
# 备份所有租户/角色/成员到 JSON，默认 10 秒内完成
px-plugin iam export \
  --entry skeleton \
  --output tmp/iam-backup.json \
  --pretty

# 仅导出指定租户（根据 key/UUID 匹配）
px-plugin iam export --entry skeleton --tenant 00000000-0000-0000-0000-000000000001 --output /tmp/local-tenant.json

# 在 Delegated 模式下强制重置管理员（谨慎使用）
px-plugin iam seed \
  --entry skeleton \
  --tenant-key 00000000-0000-0000-0000-000000000001 \
  --tenant-name "Local Tenant" \
  --admin-email admin@local.test \
  --admin-password S3cret!! \
  --force
```

导出的 JSON 包含 `tenants`、`accounts`、`members`、`roles`、`role_permissions` 等完整结构，可直接用于灾备/迁移；`iam seed` 会在事务里重建管理员密码、默认角色与部门，并对 Delegated 模式给出提示。

## 5. 指标与审计
- Prometheus 端点：`GET /api/v1/admin/runtime/metrics`
  - 关注 `plugin_iam_member_total`, `plugin_rbac_denied_total`, `plugin_iam_role_change_total`, `plugin_iam_seed_duration_seconds`
- 审计日志：`/api/v1/admin/iam/audit/logs` 支持按租户过滤；系统管理员查看全局，租户管理员仅限自身租户。

## 6. 文档/Manifest 更新
- `docs/guides/develop/standalone-mode.md`：新增 IAM/RBAC 小节、菜单显隐说明。
- `docs/operations/runbooks/iam-rbac.md`：记录常见故障、解锁管理员流程。
- `docs/contracts/rbac.schema.json` & `capabilities/catalog.json`：声明新的资源/动作。
