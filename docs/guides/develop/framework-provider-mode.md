# Framework Provider 模式

PowerXPlugin 的正式业务页面统一走插件后端 API，前端不感知运行模式。

## 运行模式

- `local`：插件后端调用本地 service / DB。
- `delegated`：插件后端调用 framework client，再通过 gateway/capability 访问 PowerX Core。

页面路径与 API 路径在两种模式下保持一致：

- `/api/v1/admin/metadata/*`
- `/api/v1/admin/iam/*`
- `/api/v1/admin/customers/*`
- `/api/v1/admin/ai-settings/*`

后端按模块内部 provider 自动分流。缺少 provider 时必须返回明确错误，不能返回空数据伪装成功。

## Mode / Diagnostics

正式业务模块需要提供 `/mode` 或等价 diagnostics，至少包含：

- `mode`
- `provider`
- `delegated_available`
- `local_available`
- `read_only`（适用于 delegated 只读模块）

provider 缺失时返回 `503`，并提供稳定错误码，例如：

- `METADATA_PROVIDER_NOT_CONFIGURED`
- `IAM_PROVIDER_NOT_CONFIGURED`
- `CUSTOMER_PROVIDER_NOT_CONFIGURED`
- `AI_SETTINGS_PROVIDER_NOT_CONFIGURED`

## 菜单边界

正式菜单保留业务入口，不因为 delegated 模式隐藏页面：

- 业务运营：客户基础管理
- 组织与权限：租户、成员、角色、权限、部门/组织树、渠道配置
- 设置：元数据治理、AI 设置

`framework-lab` 只做链路自检和健康诊断，例如 gateway ping、WS bus、scheduler、provider health，不承载 metadata/customer/IAM/AI 的 CRUD 调试入口。

## 验证

本地验证建议：

```bash
go test ./framework/backend/go/runtime/metadata ./framework/backend/go/runtime/customerfw ./framework/backend/go/runtime/aisettings
go test ./skeleton/backend/go-gin/internal/transport/http/admin/metadata ./skeleton/backend/go-gin/internal/transport/http/admin/iam ./skeleton/backend/go-gin/internal/transport/http/admin/customer ./skeleton/backend/go-gin/internal/transport/http/admin/ai_settings
npm run sync:templates -- --verbose
go work sync
git diff --check
```
