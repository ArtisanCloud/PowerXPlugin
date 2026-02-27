# 环境变量配置说明（Environment Variables Guide）

> 本页目标：说明 **PowerX 插件在运行与部署过程中涉及的全部环境变量**，  
> 包括数据库连接、上下文签名、日志、RLS、多租户与宿主通信等配置。  
>
> 读者对象：DevOps / 插件作者 / 平台运维工程师。

---

## 一、总体说明

PowerX 通过环境变量向插件注入运行参数，实现「**零代码配置启动**」。  
插件应在启动阶段自动加载并初始化这些变量。

推荐读取方式：

### Go 示例

```go
os.Getenv("POWERX_DB_DSN")
os.Getenv("POWERX_CTX_MODE")
os.Getenv("POWERX_PLUGIN_ID")
````

### Node.js / Nuxt 示例

```ts
process.env.POWERX_PLUGIN_ID
```

---

## 二、环境变量分类总览

| 类别           | 前缀                            | 用途            |
| ------------ | ----------------------------- | ------------- |
| 🌐 通用运行配置    | `POWERX_`                     | 平台与插件运行环境     |
| 🔐 签名上下文配置   | `POWERX_CTX_` / `PLUGIN_CTX_` | JWT / HMAC 校验 |
| 🗄️ 数据库配置    | `POWERX_DB_`                  | 连接与 Schema    |
| 🏷️ 插件信息     | `POWERX_PLUGIN_`              | 唯一标识与版本       |
| ⚙️ 调试与开发     | `POWERX_DEBUG_MODE` | 开发语义、内部调试路由开关 |
| 🧩 STS 与宿主通信 | `POWERX_STS_`                 | 短期凭据授权        |
| 🧾 日志与监控     | `POWERX_LOG_`                 | 统一日志级别、格式     |

---

## 三、核心配置变量

| 变量名                     | 示例值                                 | 说明           |
| ----------------------- | ----------------------------------- | ------------ |
| `POWERX_PLUGIN_ID`      | `com.powerx.plugins.base`           | 插件唯一标识       |
| `POWERX_PLUGIN_VERSION` | `0.1.0`                             | 插件版本号        |
| `POWERX_BIND_ADDR`      | `:8078`                             | 插件监听地址       |
| `POWERX_ENV`            | `dev` / `prod`                      | 当前运行环境       |
| `POWERX_DEBUG_MODE`     | `1` / `0`                           | 开发环境语义（不直接等同跳过验签） |
| `POWERX_DEV_MODE`       | `1` / `0`                           | 兼容入口（映射到 `logging.debug_mode`） |
| `POWERX_LOG_LEVEL`      | `info` / `debug` / `warn` / `error` | 日志级别         |
| `POWERX_TIMEZONE`       | `Asia/Shanghai`                     | 时区设置         |
| `POWERX_LOCALE`         | `zh-CN`                             | 默认语言环境       |

---

## 四、数据库相关配置

| 变量名                   | 示例                                                           | 说明             |
| --------------------- | ------------------------------------------------------------ | -------------- |
| `POWERX_DB_DSN`       | `postgres://user:pass@localhost:5432/powerx?sslmode=disable` | 数据库连接字符串       |
| `POWERX_DB_SCHEMA`    | `px_com_powerx_plugins_base`                                 | 插件专属 schema 名称 |
| `POWERX_DB_MAX_CONN`  | `10`                                                         | 最大连接数          |
| `POWERX_DB_IDLE_CONN` | `3`                                                          | 空闲连接数          |
| `POWERX_DB_CONN_TTL`  | `600s`                                                       | 连接最大生命周期       |
| `POWERX_DB_LOG_SQL`   | `1`                                                          | 是否打印 SQL       |

> ⚠️ 插件必须在启动时验证 schema 是否存在，否则执行迁移自动创建。

---

## 五、上下文签名（HMAC / JWT）

详见：[ctx_signing.md](../contract/ctx_signing.md)

### 1️⃣ 通用模式选择

| 变量                    | 示例              | 说明           |
| --------------------- | --------------- | ------------ |
| `POWERX_CTX_MODE`     | `jwt` / `hmac`  | 选择签名方式       |
| `POWERX_CTX_TTL`      | `300s`          | Token 有效期    |
| `POWERX_CTX_AUDIENCE` | `powerx-plugin` | JWT Audience |
| `POWERX_CTX_ISSUER`   | `powerx-auth`   | JWT Issuer   |

### 2️⃣ JWT 模式配置

| 变量                            | 示例                                | 说明             |
| ----------------------------- | --------------------------------- | -------------- |
| `POWERX_CTX_JWKS_URL`         | `http://powerx/_p/_internal/jwks` | JWKS 公钥分发 URL  |
| `POWERX_CTX_REFRESH_INTERVAL` | `3600`                            | JWKS 缓存刷新时间（秒） |

### 3️⃣ HMAC 模式配置

| 变量                       | 示例                           | 说明        |
| ------------------------ | ---------------------------- | --------- |
| `PLUGIN_CTX_HMAC_SECRET` | `base64:2ee95b10...`         | HMAC 签名密钥 |
| `PLUGIN_CTX_KID`         | `com.powerx.plugins.base:v1` | HMAC 密钥标识 |

---

## 六、多租户与 RLS 隔离配置

| 变量                         | 示例                 | 说明                      |
| -------------------------- | ------------------ | ----------------------- |
| `POWERX_TENANT_MODE`       | `multi` / `single` | 是否启用多租户隔离               |
| `POWERX_DEFAULT_TENANT_ID` | `1`                | 开发模式下默认租户               |
| `POWERX_RLS_ENABLED`       | `1`                | 是否启用 Row Level Security |
| `POWERX_TENANT_CTX_VAR`    | `app.tenant_uuid`    | 数据库中租户上下文变量名            |

Postgres 示例：

```sql
SET LOCAL app.tenant_uuid = <tenant_uuid>;
```

---

## 七、STS 短期凭据与宿主通信

| 变量                     | 示例                                        | 说明          |
| ---------------------- | ----------------------------------------- | ----------- |
| `POWERX_STS_ENDPOINT`  | `http://powerx/_p/_internal/sts/exchange` | STS 授权服务地址  |
| `POWERX_STS_TOKEN_TTL` | `300s`                                    | 临时令牌有效期     |
| `POWERX_STS_SCOPES`    | `crm:lead:read,crm:lead:create`           | 插件请求宿主的授权范围 |

示例（Go SDK）：

```go
token := powerx.GetSTSToken(scope)
httpRequest.Header.Set("Authorization", "Bearer "+token)
```

---

## 八、日志与监控配置

| 变量                        | 示例                           | 说明                    |
| ------------------------- | ---------------------------- | --------------------- |
| `POWERX_LOG_LEVEL`        | `debug`                      | 日志级别                  |
| `POWERX_LOG_FORMAT`       | `json` / `text`              | 日志输出格式                |
| `POWERX_LOG_PATH`         | `/var/log/powerx/plugin.log` | 日志路径（若未定义则输出至 STDOUT） |
| `POWERX_LOG_ENABLE_TRACE` | `1`                          | 启用请求追踪                |
| `POWERX_LOG_REQUEST_ID`   | 自动注入                         | 全局唯一请求 ID             |
| `POWERX_METRICS_PORT`     | `9101`                       | Prometheus 指标端口（可选）   |

示例输出：

```json
{"level":"info","msg":"create template","tenant_uuid":1024,"plugin":"com.powerx.plugins.base","request_id":"req-98af31"}
```

---

## 九、能力 Gateway 调用配置

PowerX 通用能力统一通过 Integration Gateway 调用，需要为宿主与 Skeleton 模式注入以下变量：

| 变量名 | 示例值 | 说明 |
| --- | --- | --- |
| `PX_GATEWAY_BASE_URL` | `https://gateway.powerx.dev/_tenant` / `http://localhost:8080` | Gateway HTTP 入口，宿主由运维注入，Skeleton `.env.local` 指向 Dev 环境。 |
| `PX_PLUGIN_TOOL_TOKEN` | `sts-1u8c5e...` | 宿主模式下由 Admin/部署系统注入的 Tool Grant，供后端/前端调用 Gateway。 |
| `PX_TOOL_TOKEN` | `sts-dev-9ad3...` | Skeleton 本地通过 `px-plugin login --manifest ./skeleton/plugin.yaml` 生成的临时 Token，写入 `.env.local`。 |
| `PX_TOOL_REFRESH_TOKEN` | `sts-dev-refresh-xxxx` | 配套 refresh token，Skeleton 可在 Token 过期或 24 小时内到期时自动调用 `/admin/user/auth/refresh` 获取新的 Tool Token。 |
| `PX_USE_MOCK` | `media,eventfabric` | Skeleton 可选，指定需要走内存 Mock 的能力模块。 |

使用建议：

1. **宿主模式**：在部署 YAML 或环境注入脚本中设置 `PX_GATEWAY_BASE_URL`、`PX_PLUGIN_TOOL_TOKEN`，后端读取后写入 Gateway Client，并通过插件自有 API 暴露调用入口；租户从 token 的 `tid` claim 自动推导。
2. **Skeleton 模式**：执行 `px-plugin login --manifest ./skeleton/plugin.yaml` 后会在 `~/.powerx/credentials` 生成 Token，使用脚本写入 `skeleton/.env.local` 中的 `PX_GATEWAY_BASE_URL`、`PX_TOOL_TOKEN`。
3. **Mock/降级**：当 Dev Gateway 不可达时，将 `PX_USE_MOCK` 设置为能力模块名（如 `media`），框架会自动切换内存实现并在日志中提示。
4. **安全性**：Token 属于短期凭证，建议在启动时检测其过期时间；自动刷新失败时应阻断调用并提示开发者重新登录或联系宿主运维。

后端调用示例：

```go
client := gateway.NewClient(gateway.Config{
    BaseURL:   os.Getenv("PX_GATEWAY_BASE_URL"),
    ToolToken: os.Getenv("PX_PLUGIN_TOOL_TOKEN"),
})
```

前端调用示例：

```ts
const config = useRuntimeConfig()
await $fetch(config.public.powerx.apiBase + config.public.powerx.capabilityEndpoint, {
  method: 'POST',
  body: { capabilityId: 'com.corex.media.assets.read', action: 'List', payload: {} },
})
```

---

## 十、前端（web-admin）运行环境

前端在构建时可通过 Nuxt `runtimeConfig` 动态注入：

| 环境变量                       | 示例                                   | 用途         |
| -------------------------- | ------------------------------------ | ---------- |
| `NUXT_PUBLIC_API_BASE`     | `/_p/com.powerx.plugins.base/api/v1` | API 前缀     |
| `NUXT_PUBLIC_PLUGIN_ID`    | `com.powerx.plugins.base`            | 插件标识       |
| `NUXT_PUBLIC_POWERX_PROXY` | `1`                                  | 是否处于宿主反代模式 |
| `NUXT_PUBLIC_LANG_DEFAULT` | `zh`                                 | 默认语言       |
| `NUXT_PUBLIC_POWERX_API_BASE` | `/_p/com.powerx.plugins.base/api/v1` 或 `http://localhost:8078/api/v1` | 前端访问插件后端时的 API Base，默认指向插件自身 `_p/<plugin>/api/v1` |
| `NUXT_PUBLIC_POWERX_CAPABILITY_ENDPOINT` | `/integration/capabilities/invoke` | 插件后端提供的能力调用端点（相对 `NUXT_PUBLIC_POWERX_API_BASE`） |

前端通过 `useRuntimeConfig()` 获取：

```ts
const config = useRuntimeConfig()
const api = config.public.apiBaseUrl
```

---

## 十一、调试与开发模式

| 变量                                         | 说明                |
| ------------------------------------------ | ----------------- |
| `POWERX_DEBUG_MODE=1`                      | 启用开发环境语义（默认策略偏开发） |
| `POWERX_DEV_MODE=1`                        | 兼容入口（等价于 `POWERX_DEBUG_MODE=1`） |
| `POWERX_LOG_LEVEL=debug`                   | 打印详细日志            |
| `POWERX_DB_LOG_SQL=1`                      | 打印 SQL 查询         |
| `POWERX_DEFAULT_TENANT_ID=1`               | 强制使用默认租户 ID       |
| `POWERX_BIND_ADDR=:8078`                   | 启动本地 HTTP 监听      |
| `POWERX_PLUGIN_ID=com.powerx.plugins.base` | 标识插件身份            |

本地启动命令：

```bash
POWERX_DEBUG_MODE=1 go -C backend run ./cmd/plugin
```

---

## 十二、部署环境推荐配置（生产）

| 类别      | 推荐值      | 说明           |
| ------- | -------- | ------------ |
| 日志级别    | `info`   | 减少日志量        |
| 签名模式    | `jwt`    | 使用 JWKS 公钥验证 |
| STS 有效期 | `300s`   | 短期令牌         |
| Dev 模式  | 禁用       | 不允许跳过签名      |
| 数据库连接   | 连接池 ≥ 10 | 生产性能优化       |
| 容器用户    | 非 root   | 避免宿主权限暴露     |

---

## 十三、示例 `.env` 文件

```
POWERX_PLUGIN_ID=com.powerx.plugins.base
POWERX_PLUGIN_VERSION=0.1.0
POWERX_DB_DSN=postgres://user:pwd@db:5432/powerx?sslmode=disable
POWERX_DB_SCHEMA=px_com_powerx_plugins_base
POWERX_CTX_MODE=jwt
POWERX_CTX_JWKS_URL=http://powerx/_p/_internal/jwks
POWERX_CTX_ISSUER=powerx-auth
POWERX_CTX_AUDIENCE=powerx-plugin
POWERX_LOG_LEVEL=info
POWERX_DEBUG_MODE=0
POWERX_STS_ENDPOINT=http://powerx/_p/_internal/sts/exchange
PX_GATEWAY_BASE_URL=https://gateway.powerx.dev/_tenant
PX_PLUGIN_TOOL_TOKEN=sts-prod-xxxxxxxx
PX_USE_MOCK=
```

加载方式：

```bash
source .env
```

---

## 十四、验证命令

```bash
echo $POWERX_PLUGIN_ID
curl $POWERX_STS_ENDPOINT
psql "$POWERX_DB_DSN" -c "SELECT current_schema()"
```

---

## 十五、变量优先级规则

1️⃣ CLI 参数（如 `make run POWERX_DEBUG_MODE=1`）
2️⃣ `.env` 文件加载的变量
3️⃣ 系统环境变量（`export`）
4️⃣ 默认配置（代码内硬编码）

---

## 十六、配置检查命令（建议实现）

插件可提供 `/api/v1/admin/config-check` 端点，输出运行配置摘要：

```json
{
  "plugin_id": "com.powerx.plugins.base",
  "schema": "px_com_powerx_plugins_base",
  "ctx_mode": "jwt",
  "tenant_mode": "multi",
  "db_status": "ok",
  "sts_endpoint": "http://powerx/_p/_internal/sts/exchange"
}
```

---

## 十七、总结

* 所有插件配置通过环境变量注入，支持 Docker、K8s、系统级运行；
* 开发与生产环境仅差异在签名模式与日志级别；
* 变量命名统一使用 `POWERX_` 前缀，保持平台一致性；
* 环境变量即构成插件运行的契约接口。

---

## 十八、关联文档

| 模块          | 文档                                                                     |
| ----------- | ---------------------------------------------------------------------- |
| 签名上下文规范     | [../contract/ctx_signing.md](../contract/ctx_signing.md)               |
| PowerX 通信协议 | [../contract/powerx_integration.md](../contract/powerx_integration.md) |
| 安全加固指南      | [./security_hardening.md](./security_hardening.md)                     |
| Docker 部署说明 | [./docker_guide.md](./docker_guide.md)                                 |
| 本地调试指南      | [./local_debug.md](./local_debug.md)                                   |

---

## 十九、下一步阅读

* 🧩 [本地调试与联调指南](./local_debug.md)
* 🧱 [部署与运行检查清单](./security_hardening.md)
