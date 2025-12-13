# Quickstart: Standalone 模式 IAM & RBAC

## 1. 初始化本地 IAM
```bash
cd skeleton/backend
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
cd skeleton/backend
go run ./cmd/plugin

# Frontend admin
cd ../web-admin
npm install
npm run dev
```
- Standalone 模式访问 `http://localhost:3031/admin/iam/overview`。
- Delegated 模式访问 `_p/<plugin-id>/admin`，IAM 菜单不可见。

## 3. Playwright 验证
```bash
# Delegated 登录（需宿主）
npm --prefix skeleton/web-admin run test:e2e -- auth-delegated

# Local IAM 流程
PLAYWRIGHT_LOCAL_IAM=1 \
PLAYWRIGHT_LOCAL_EMAIL=admin@local.test \
PLAYWRIGHT_LOCAL_PASSWORD='S3cret!!' \
npm --prefix skeleton/web-admin run test:e2e -- auth-local

# IAM 管理新增用例（完成后）
PLAYWRIGHT_LOCAL_IAM=1 npm --prefix skeleton/web-admin run test:e2e -- iam-local

# Delegated 提示（验证本地入口隐藏）
PLAYWRIGHT_LOCAL_IAM=0 npm --prefix skeleton/web-admin run test:e2e -- auth-local
```

## 4. CLI 工具
```bash
# 导出租户/角色/权限
px-plugin iam export --entry skeleton --output /tmp/iam-export.json

# 重置管理员凭据
px-plugin iam seed --entry skeleton --admin-email new@tenant.test --admin-password 'NewP@ssw0rd'
```

## 5. 指标与审计
- Prometheus 端点：`GET /api/v1/admin/runtime/metrics`
  - 关注 `plugin_iam_member_total`, `plugin_rbac_denied_total`, `plugin_iam_seed_duration_seconds`
- 审计日志：`/api/v1/admin/iam/audit/logs` 支持按租户过滤；系统管理员查看全局，租户管理员仅限自身租户。

## 6. 文档/Manifest 更新
- `docs/guides/develop/standalone-mode.md`：新增 IAM/RBAC 小节、菜单显隐说明。
- `docs/operations/runbooks/iam-rbac.md`：记录常见故障、解锁管理员流程。
- `docs/contracts/rbac.schema.json` & `capabilities/catalog.json`：声明新的资源/动作。
