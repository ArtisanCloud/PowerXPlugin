# 插件鉴权 Token 与上下文规范

> 本页目标：定义 PowerX 平台与插件之间的 token 使用边界、用户/member claims 标准，以及旧 Signed-Context 的兼容范围。
> 读者对象：后端开发者 / 安全工程师 / 插件集成方。

---

## 一、设计目标

PowerX 平台与插件间通过反代通信，为了保证：

- **租户隔离**：每个请求仅携带自身租户上下文；
- **成员身份明确**：租户模式下必须区分全局用户 `uid` 和当前租户成员 `mid`；
- **权限验证**：请求中附带当前成员的授权权限；
- **防篡改安全**：通过 JWT 或受控兼容 Header 防止伪造；
- **跨语言兼容**：插件可用任何语言实现验证逻辑。

正式链路优先使用 JWT Bearer token。历史 `X-PowerX-CTX` Signed-Context 仅作为本地开发或旧版本兼容入口，不作为新功能主路径。

## 二、Token 类型

| Token 类型 | 方向 | audience | 用途 | 主路径 |
| --- | --- | --- | --- | --- |
| STS access token | 插件 -> PowerX | `powerx:api` | 插件服务调用 PowerX Gateway、Capability、Scheduler、ws-bus 等底座接口 | 是 |
| Plugin request token | PowerX -> 插件 | `plugin:<plugin_id>` | PowerX 动态代理当前登录用户请求到插件后端 | 是 |
| Root debug token | Root/Admin -> 插件 | `plugin:<plugin_id>` | 开发调试手工签发 | 否 |

`PX_PLUGIN_TOOL_TOKEN` / `PX_TOOL_TOKEN` 已废弃。宿主模式不得再注入、读取或依赖它们作为业务调用凭证。

## 三、Claims 标准

PowerX 统一使用 `CoreXClaims` 语义。租户模式下，用户态 token 必须同时表达全局用户和当前租户成员：

| Claim | 含义 | 用户态 token / Plugin request token | STS access token |
| --- | --- | --- | --- |
| `tid` | 当前租户 UUID | 必填 | 必填 |
| `tid_n` | 当前租户数值 ID | 必填 | 必填 |
| `uid` | 全局用户 UUID | 必填 | 不填 |
| `uid_n` | 全局用户数值 ID | 必填 | 不填 |
| `mid` | 当前租户成员 UUID | 必填 | 不填 |
| `mid_n` | 当前租户成员数值 ID | 必填 | 不填 |
| `email` / `phone` | 当前用户联系信息 | 可选 | 不填 |
| `scope` | token scope | `access` | `access` |
| `aud` | token audience | `user` 或 `plugin:<plugin_id>` | `powerx:api` |
| `sub` | JWT subject | `mid`，即当前成员 UUID | `client:<client_id>` |

标准语义：

- `uid/uid_n` 表示全局账号。同一用户可以加入多个租户。
- `mid/mid_n` 表示该用户在当前租户下的成员身份。权限、部门、角色、状态、通知订阅和审计归属必须优先使用 `tenant + member`。
- 用户登录 token 和 Plugin request token 都属于用户态 token，必须带 `tid + uid + mid`。
- STS access token 属于插件服务 token，代表某个插件实例在某个租户下调用 PowerX，不代表某个登录成员，因此不携带 `uid/uid_n/mid/mid_n`。
- 如果未来需要“插件代表某个用户/member 调用 PowerX”，必须新增明确的 on-behalf-of/delegated actor 机制，不得把普通 STS token 混用成用户态 token。

用户态 token 示例：

```json
{
  "tid": "00000000-0000-0000-0000-000000000001",
  "tid_n": 1,
  "uid": "user_uuid",
  "uid_n": 10,
  "mid": "member_uuid",
  "mid_n": 20,
  "roles": ["system.admin"],
  "perms": ["crm:lead:create"],
  "scope": "access",
  "iss": "powerx-auth",
  "aud": "plugin:com.powerx.plugins.ai-craft",
  "sub": "member_uuid"
}
```

STS token 示例：

```json
{
  "tid": "00000000-0000-0000-0000-000000000001",
  "tid_n": 1,
  "plugin_id": "com.powerx.plugins.ai-craft",
  "scope": "access",
  "iss": "powerx-sts",
  "aud": "powerx:api",
  "sub": "client:com.powerx.plugins.ai-craft.00000000-0000-0000-0000-000000000001"
}
```

## 四、PowerX 代理当前用户请求到插件

浏览器访问插件 API 的路径：

```text
Browser -> PowerX /_p/:plugin_id/api/* -> Plugin Backend
```

PowerX 动态代理负责：

1. 校验当前浏览器用户登录态。
2. 解析租户、全局用户、当前租户成员上下文。
3. 执行插件 RBAC 判定。
4. 签发短期 Plugin request token。
5. 将上游请求头改写为：

```http
Authorization: Bearer <plugin_request_token>
```

插件后端使用该 token 识别当前访问用户和租户上下文。业务归属、通知、审计和权限二次判断应使用 `tenant + member`，不要只使用 `user`。

插件不得把 Plugin request token 当作调用 PowerX 底座业务接口的凭证。

## 五、插件调用 PowerX 底座

插件主动调用 PowerX 底座业务接口时必须走 STS。

插件持有租户维度凭证：

```text
POWERX_STS_CLIENT_ID=<plugin_id>.<tenant_uuid>
POWERX_STS_CLIENT_SECRET=<rotated_secret>
POWERX_STS_AUDIENCE=powerx:api
POWERX_STS_SCOPE=access
```

STS Exchange 返回的 `access_token` 用于调用 PowerX：

```http
Authorization: Bearer <sts_access_token>
```

PowerX 底座业务接口应校验：

```text
audience = powerx:api
scope = access
tenant_uuid 存在且有效
subject = client:<client_id>
```

STS access token 的主体是插件服务账号，不是当前登录用户。该 token 的 claims 必须包含 `tid/tid_n`，不应要求或伪造 `uid/uid_n/mid/mid_n`。需要写审计时应记录 `actor_type=plugin` 或 `actor_type=service_account`，并记录 `client_id/plugin_id/tenant_uuid`。

## 六、上下文在插件中的使用

插件中间件在验证后，会注入上下文到 `gin.Context` 或 `req.Context()`：

```go
type PowerXContext struct {
    TenantUUID  string
    TenantID    int64
    UserID      int64
    MemberID    int64
    Permissions []string
}

func (c *gin.Context) GetPowerX() *PowerXContext {
    val, _ := c.Get("powerx_ctx")
    return val.(*PowerXContext)
}
```

然后业务层即可直接访问：

```go
tenantUUID := c.GetPowerX().TenantUUID
memberID := c.GetPowerX().MemberID
```

---

## 七、请求链路示例

```text
Client
  ↓
PowerX Gateway
  ↓ (Authorization: Bearer <plugin_request_token>)
  ↓
Plugin Reverse Proxy (/_p/:id/api/v1/...)
  ↓
Plugin Middleware (verify token)
  ↓
BeginTenantTx → SET LOCAL app.tenant_uuid
  ↓
Postgres (RLS) → Response
```

---

## 八、旧 Signed-Context 兼容范围

旧版本存在 `X-PowerX-CTX` / `X-PowerX-CTX-SIG` Header 链路。它只允许在以下场景继续存在：

- 本地开发或旧插件过渡。
- 配置显式开启 `allow_signed_context`。
- Payload 至少包含 `tid`、`uid`、`mid` 对应的租户、用户、成员数值上下文。

新代码不得把 Signed-Context 作为宿主模式主路径；PowerX 动态代理到插件后端的正式路径是 `Authorization: Bearer <plugin_request_token>`。

## 九、常见错误与排查

| 错误                   | 原因           | 解决方案                             |
| -------------------- | ------------ | -------------------------------- |
| `invalid token` | Bearer token 缺失、签名不匹配、issuer/audience 不匹配 | 检查 `Authorization`、issuer、audience 和验签密钥 |
| `token expired`      | JWT 已过期      | 检查 `POWERX_CTX_TTL` 或时间同步        |
| `member missing` | 用户态 token 未携带 `mid/mid_n` | 检查 PowerX 代理 token 或本地登录 token 签发逻辑 |
| `tenant mismatch` | 请求租户与鉴权上下文不一致 | 不要从前端 body/query 伪造租户，使用 token 内租户上下文 |
| `permission denied`  | 缺少对应权限       | 检查 PowerX 租户角色设置                 |

---

## 十、安全建议

- 生产部署使用短期 JWT Bearer token，token 有效期不应超过 5 分钟。
- 插件必须从 token 上下文提取 `tenant_uuid` / `member_id`，不可信任客户端传参。
- 日志不得打印完整 token，仅记录 request_id、tenant、member 或 token 前缀。
- STS token 不得伪造 `uid/mid`；用户态 token 不得被拿去调用 PowerX 底座业务接口。
- 旧 Signed-Context 需要显式开关和 HMAC 密钥，不得默认开启。

---

## 十一、测试工具

### Plugin request token 测试

```bash
curl http://localhost:8080/_p/com.powerx.plugins.base/api/v1/templates \
  -H "Authorization: Bearer <plugin_request_token>"
```

### STS access token 测试

```bash
curl http://localhost:8077/api/v1/admin/scheduler/jobs \
  -H "Authorization: Bearer <sts_access_token>"
```

---

## 十二、附录：环境变量总览

| 环境变量                     | 说明                |
| ------------------------ | ----------------- |
| `POWERX_CTX_JWKS_URL`    | JWKS 公钥集地址        |
| `POWERX_CTX_ISSUER`      | 签发方               |
| `POWERX_CTX_AUDIENCE`    | 受众                |
| `POWERX_CTX_TTL`         | Token 有效期         |
| `POWERX_STS_CLIENT_ID` | 插件调用 PowerX 的 STS client id |
| `POWERX_STS_CLIENT_SECRET` | 插件调用 PowerX 的 STS secret |
| `POWERX_STS_AUDIENCE` | 固定为 `powerx:api` |
| `POWERX_STS_SCOPE` | 固定为 `access` |
| `PLUGIN_CTX_HMAC_SECRET` | 旧 Signed-Context 兼容密钥 |
| `POWERX_DEBUG_MODE`      | 开发环境语义开关（不直接控制验签旁路） |
| `POWERX_DEV_MODE`        | 兼容入口（映射到 `logging.debug_mode`） |

---

## 十三、关联规范

| 模块               | 文档                                               |
| ---------------- | ------------------------------------------------ |
| Plugin 注册结构      | [plugin.yaml 规范](./plugin_yaml_spec.md)          |
| RBAC/Manifest 接口 | [rbac_manifest_spec.md](./rbac_manifest_spec.md) |
| Agent 注册协议       | [agent_contract.md](./agent_contract.md)         |
| PowerX 集成流程      | [powerx_integration.md](./powerx_integration.md) |

---

## 十四、总结

- 插件端必须实现签名验证；
- 用户态上下文必须携带 `tenant + user + member`；
- STS token 只代表插件服务身份，不携带用户/member；
- 验证通过后才能进入业务逻辑；
- 插件中可依赖 `TenantContext` 注入的作用域保证安全执行。

---

## 下一步阅读

- 🔄 [PowerX Integration 交互流程](./powerx_integration.md)
- 🧩 [plugin.yaml 规范](./plugin_yaml_spec.md)
