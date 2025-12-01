# Delegated Mode 调试与运行指南

本指南梳理插件在 **宿主委派模式（Delegated）** 下的端到端流程，结合 `docs/guides/develop/auth.md` 的说明，帮助定位“宿主已登录但插件仍提示登录/404”等问题。

## 1. 角色与进程
- **宿主 PowerX Core**：监听 `8077`（默认），提供 `/api/v1/**`，在浏览器 `localStorage` 中存储宿主登录 token（插件与宿主共用）。
- **插件后端（process 模式）**：由宿主拉起，暴露插件自带接口 `/_p/<pluginId>/api/v1/**`（无宿主登录态接口）。
- **插件前端（process 模式）**：由宿主以 iframe 方式渲染 `/_p/<pluginId>/admin/**`，静态文件来自插件包 `web-admin/.output`。
- **CLI 构建/打包**：`px-plugin package` 默认注入宿主前缀与宿主模式环境（POWERX_PROXY=1，baseURL=`/_p/<pluginId>/admin/`，宿主 API=`/api/v1`）。

## 2. 关键配置（宿主 Delegated/process 模式）
- **前端 baseURL / assets 路径**
  - `NUXT_APP_BASE_URL=/ _p/<pluginId>/admin/`（已由 CLI 默认注入；模板 `nuxt.config.ts` 也覆盖）。
  - `buildAssetsDir='assets/'`（相对路径，避免落到宿主根 `/assets`）。
- **前端 API 基址（宿主 vs 插件）**
  - INSIDE_POWERX=true 时：`apiBaseUrl=/api/v1`，并从宿主浏览器 `localStorage` 读取宿主 token 复用。
  - 独立/本地：`apiBaseUrl=/_p/<pluginId>/api/v1` 或 `NUXT_PUBLIC_API_BASE`，token 由插件自己管理。
- **宿主模式标记**：
  - `POWERX_PROXY=1`（后端/构建时环境，生成 baseURL、判断委派模式）。
  - `NUXT_PUBLIC_POWERX_PROXY=1`（前端 runtimeConfig.public 可见，浏览器侧判定 insidePowerX）。
  - 两者语义一致但作用域不同，`px-plugin package` 会同时注入。
- **安装路径**：`backend/plugins/installed/<pluginId>/<version>/`，前端产物在 `web-admin/.output`。

## 3. 运行顺序（宿主调试）
1. 宿主登录（宿主在浏览器 `localStorage` 写入 token，供插件前端复用）。
2. 安装插件包（包含 `payload/web-admin/.output` 与 `payload/backend/bin/*`）。
3. 宿主拉起插件前端（iframe `/ _p/<pluginId>/admin/...`）和插件后端（process）。
4. 前端初始请求：
   - 静态资源：`/_p/<pluginId>/admin/assets/...`（来自包内 `.output/public`）。
   - 会话检测：`/api/v1/admin/auth/me/context`（宿主接口，复用宿主 Cookie）。
   - 业务接口：宿主接口 `/api/v1/**` 或插件接口 `/_p/<pluginId>/api/v1/**`（视实现而定）。

## 4. 常见症状与排查
- **仍出现登录页**：检查浏览器 Network 中 `auth/me/context` 是否打到 `/api/v1/...` 且请求头包含宿主 token（来源于宿主写入的 localStorage）；若打到 `/_p/...` 或缺 token，说明用的是旧构建或缓存，清缓存/重装包并确认宿主 token 存在。
- **静态资源 404**：确认包内 `web-admin/.output/public/assets/*` 存在；确保 baseURL 带插件前缀，路径不是根 `/assets/...`；宿主反代需转发 `/_p/<pluginId>/admin/*filepath`。
- **i18n baseDir not found**：未在包内包含 `web-admin/i18n/locales`（或你自定义的 langDir）；重建前端确保资源被打入 `.output`。
- **插件接口 404**：业务接口应走 `/_p/<pluginId>/api/v1/**`；若走宿主 `/api/v1` 会找不到插件接口，需前端按场景区分宿主/插件 API。

## 5. 构建/打包要点
- 在插件根目录运行：`px-plugin package --entry .`（会自动 `npm --prefix web-admin run build`）。
- 确认包内：
  - `web-admin/.output/server/index.mjs`、`web-admin/.output/public/assets/*`。
  - `web-admin/.output/server/chunks/nitro/nitro.mjs` 中 `baseURL=/ _p/<pluginId>/admin/`、`apiBaseUrl=/api/v1`、`insidePowerX=true`。
  - `backend/bin/plugin`、`backend/bin/migrate` 等后台二进制存在。

## 6. 宿主/插件接口版本
- 宿主 API 版本由宿主控制（默认 `/api/v1`），插件前端宿主模式固定指向该前缀。
- 插件自有 API 前缀 `/_p/<pluginId>/api/<version>` 可独立升级，不影响宿主；需同步前端配置与后端路由。

## 7. 快速自检清单
- Network：`/api/v1/admin/auth/me/context` 返回 200 且请求附带宿主 localStorage 注入的 token。
- Network：静态资源请求形如 `/_p/<pluginId>/admin/assets/...`，不应是根 `/assets/...`。
- 包内 `nitro.mjs`：`insidePowerX=true`、`apiBaseUrl=/api/v1`、`baseURL=/ _p/<pluginId>/admin/`。
- 宿主日志：看到 `/api/v1/admin/auth/me/context` 200，`/_p/:id/admin/*filepath` 200。

若以上均正常仍提示登录，多半是浏览器/代理缓存旧前端资源，清缓存并重装包再试。

## 8. 跨域会话传递（postMessage 方案）
当宿主与插件不在同一 host（如宿主 `localhost:3030`，插件 `127.0.0.1:8077`）时，浏览器不会共享 Cookie/localStorage。可用 postMessage 注入短时 token：

- 宿主侧（PowerX Web Admin）在 iframe `onload` 后发送：

  ```ts
  iframe.contentWindow?.postMessage(
    {
      source: "powerx",
      type: "auth-token",
      accessToken: "<宿主 access_token>",
      refreshToken: "<可选>",
      tokenType: "Bearer",
      // 二选一：expiresIn 秒 或 expiresAt 毫秒时间戳
      expiresIn: 3600,
      pluginId: "<插件ID，可选>",
    },
    "http://127.0.0.1:8077" // 精确 targetOrigin
  );
  ```

- 插件侧（已在 `skeleton/web-admin` 实现）监听 `message`，校验 `origin` 并将 token 写入本域 localStorage，随后调用 `/api/v1/admin/auth/me/context` 即可通过。payload 字段说明：
  - `accessToken`（必填）：宿主 access_token
  - `refreshToken`（可选）
  - `tokenType`（可选，默认 Bearer）
  - `expiresIn` 或 `expiresAt`（二选一，前者为秒，后者为毫秒时间戳）
  - `pluginId`（可选，宿主多插件时可按需筛选）

- 安全要点
  - 宿主发送时必须设置精确 `targetOrigin`（插件实际 origin），避免使用 `*`。
  - 插件仅接受匹配 origin 且 `source === "powerx"` 的消息，收到无效/过期 token 会忽略。

使用此方案后，跨域 iframe 不再依赖共享 Cookie，插件前端拿到 token 即可继续宿主接口调用。
