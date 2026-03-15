# Parity Map (Nuxt -> Next)

| Domain | Nuxt Route | Next Route | Mode Scope | Status | Case ID |
|---|---|---|---|---|---|
| Auth | /users/login | /users/login | both | migrated | E2E-AUTH-LOGIN |
| Auth | /users/register | /users/register | both | migrated | E2E-AUTH-REGISTER |
| Auth | /users/forgot-password | /users/forgot-password | both | migrated | E2E-AUTH-FORGOT |
| Intro | /intro | /intro | both | migrated | E2E-INTRO |
| Templates | /templates | /templates | both | migrated | E2E-TPL-LIST |
| Templates | /templates/crud | /templates/crud | both | migrated | E2E-TPL-CRUD |
| Templates | /templates/develop | /templates/develop | both | migrated | E2E-TPL-DEV |
| IAM | /admin/iam/overview | /admin/iam/overview | both | migrated | E2E-IAM-OVERVIEW |
| IAM | /admin/iam/members | /admin/iam/members | both | migrated | E2E-IAM-MEMBERS |
| IAM | /admin/iam/roles | /admin/iam/roles | both | migrated | E2E-IAM-ROLES |
| IAM | /admin/iam/settings | /admin/iam/settings | both | migrated | E2E-IAM-SETTINGS |
| Capability | /capabilities/Lifecycle | /capabilities/lifecycle | both | migrated | E2E-CAP-LIFECYCLE |
| Capability | /capabilities/RegisterForm | /capabilities/register | both | migrated | E2E-CAP-REGISTER |
| Capability | /powerx/capability-lab | /powerx/capability-lab | both | migrated | E2E-CAP-LAB |
| Integration | /_p/{pluginId}/admin/integration/* | /integration | host | migrated | E2E-INTEGRATION |
| Operations | /_p/{pluginId}/admin/operations/* | /operations | host | migrated | E2E-OPERATIONS |
| Security | /_p/{pluginId}/admin/security/* | /security | host | migrated | E2E-SECURITY |

## US1 TestID 对照

| 场景 | Nuxt 选择器语义 | Next TestID | 说明 |
|---|---|---|---|
| 登录账号输入 | `input[name="identifier"]` | `login-username` | `auth-local.spec.ts` 复用 |
| 登录密码输入 | `input[name="password"]` | `login-password` | `auth-local.spec.ts` 复用 |
| 登录提交 | 登录按钮 | `login-submit` | 本地/委托模式共用断言 |
| 首页标题 | intro 入口标题 | `intro-title` | 登录后落地页断言 |
| 模板 CRUD 标题 | templates/crud 标题 | `templates-crud-title` | CRUD 页面就绪断言 |
| 创建模板按钮 | create 操作按钮 | `templates-create-btn` | 打开表单弹窗 |
| 模板表单-名称 | form name 输入 | `template-form-name` | 创建/编辑共用 |
| 模板表单-描述 | form description 输入 | `template-form-description` | 创建/编辑共用 |
| 模板表单-内容 | form content 输入 | `template-form-content` | 创建/编辑共用 |
| 模板表单提交 | submit 按钮 | `template-form-submit` | 触发保存请求 |
| 模板行 | 模板行定位 | `template-row-{id}` | 编辑/删除目标定位 |

## US2 关键断言映射

| 场景 | Next 路由/元素 | 用例 |
|---|---|---|
| 委托鉴权禁用本地登录 | `/users/login` + `login-submit` disabled | `auth-delegated.spec.ts` |
| IAM 概览可达 | `/admin/iam/overview` + `iam-overview-page` | `iam-local.spec.ts` |
| IAM 成员可达 | `/admin/iam/members` + `iam-members-list` | `iam-local.spec.ts` |
| IAM 角色可达 | `/admin/iam/roles` + `iam-roles-list` | `iam-local.spec.ts` |
| IAM 设置可达 | `/admin/iam/settings` + `iam-settings-page` | `iam-local.spec.ts` |
| 能力调用成功/失败语义 | `/tests/capability` + `status-indicator` / `error-indicator` | `capability-invocation.spec.ts` |
| 双模式路由一致性 | `/integration` 与 `/_p/{pluginId}/admin/integration` | `mode-parity-edge.spec.ts` |
| 逐页路由可达性 | IAM/Capabilities/Integration/Ops/Security 各路由 | `route-parity.spec.ts` |
| 错误语义矩阵回归 | IAM 403 与 capability 500 语义 | `error-semantics.spec.ts` |
