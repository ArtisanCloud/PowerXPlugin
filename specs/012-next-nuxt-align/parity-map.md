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
| IAM | /admin/iam/overview | /admin/iam/overview | both | pending | E2E-IAM-OVERVIEW |
| IAM | /admin/iam/members | /admin/iam/members | both | pending | E2E-IAM-MEMBERS |
| IAM | /admin/iam/roles | /admin/iam/roles | both | pending | E2E-IAM-ROLES |
| IAM | /admin/iam/settings | /admin/iam/settings | both | pending | E2E-IAM-SETTINGS |
| Capability | /capabilities/Lifecycle | /capabilities/lifecycle | both | pending | E2E-CAP-LIFECYCLE |
| Capability | /capabilities/RegisterForm | /capabilities/register | both | pending | E2E-CAP-REGISTER |
| Capability | /powerx/capability-lab | /powerx/capability-lab | both | pending | E2E-CAP-LAB |
| Integration | /_p/{pluginId}/admin/integration/* | /integration | host | pending | E2E-INTEGRATION |
| Operations | /_p/{pluginId}/admin/operations/* | /operations | host | pending | E2E-OPERATIONS |
| Security | /_p/{pluginId}/admin/security/* | /security | host | pending | E2E-SECURITY |

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
