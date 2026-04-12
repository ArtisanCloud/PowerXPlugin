# Next API 与鉴权说明

## 目标
- 与 Nuxt/Gin 既有契约保持一致。
- 不新增仅 Next 可用的私有后端接口。

## 目录
- `client.ts`: 通用请求入口，解析 envelope（`code/message/data`）。
- `normalizeApiError.ts`: 错误标准化，统一抛出 `ApiError`。
- `baseUrl.ts`: 运行模式（standalone/host）与 API 基址解析。
- `auth.ts`: 登录/注册/重置密码接口封装。
- `template.ts`: 模板列表与 CRUD 接口封装。
- `iam.ts`: IAM 概览/成员/角色/设置接口封装。
- `capabilities.ts`: 能力生命周期/注册/调用接口封装。
- `operations.ts`: Integration/Operations/Security 接口封装。

## 鉴权约定
- token 主存储：`localStorage`
  - `access_token`
  - `refresh_token`
  - `expires_at`
- 兼容 `middleware` 守卫：写入 `access_token` cookie。
- 请求默认携带：
  - `Authorization: Bearer <access_token>`
  - `credentials: include`

## 请求/响应约定
- 所有请求经过 `apiRequest()`，默认 `Content-Type: application/json`。
- 统一 envelope：

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

- 非 2xx 响应统一转 `ApiError` 并在页面层展示。

## 联调与门禁
- 每次新增 API 调用后执行：

```bash
cd /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin
./specs/012-next-nuxt-align/scripts/check-contract-drift.sh
```

- 若检测到 drift，必须登记 `parity-gap-log.md` 并阻断发布。
