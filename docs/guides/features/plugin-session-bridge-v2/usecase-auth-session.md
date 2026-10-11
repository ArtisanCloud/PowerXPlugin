# 会话、请求与退出验收

## 目标与字段来源

Token、expiry、sessionId、revision、sequence 来自 Core v2；用户状态、租户、成员 UUID 与 permission_codes 来自本条 Token 的正式 context。两个输入验证成功后才发布 ready；没有 refreshToken、旧 Cookie 或本地目录补齐路径。

## 接入步骤

1. **更新消费入口**：配置本地 `file:` 依赖／工作区链接，参考 Skeleton `auth.client.ts` 注册一次适配器。预期只保留 Core 注入消费者的认证订阅；失败检查 protocolVersion 与来源。
2. **返回验证 Promise**：`onAuthToken` 返回 `session.apply(...)`，`readContext` 用本条候选快照调用正式 context；返回值只读，不能提前提交到 store。验证失败抛出，Core 发送 rejected。
3. **提交与清理**：成功后提交 context，Root 的实际权限来自 context；退出清自身 Pinia／页面业务缓存和连接。Skeleton 会卸载受保护页面并处置自身缓存 store。普通请求不能等待退出后的下一位用户。
4. **统一 API**：ofetch 使用 `createPowerXApiFetchClient`；原生入口使用 `usePluginApi`。API base 是当前插件作用域；嵌入请求不接受备用 Token、旧 Authorization 或跨插件地址。

正式 context 接口示例（只使用当前开发会话的 accessToken；不要记录值）：

```bash
curl --fail-with-body --silent \
  -H "Authorization: Bearer ${PLUGIN_TEST_ACCESS_TOKEN}" \
  'http://127.0.0.1:3030/_p/com.powerx.plugins.base/api/v1/admin/user/auth/me/context'
```

预期响应片段为正式 Core envelope：

```json
{"code":200,"data":{"is_root":false,"current_tenant_uuid":"<tenant-uuid>","current_member_uuid":"<member-uuid>","user":{"status":1},"members":[{"tenant_uuid":"<tenant-uuid>","member_uuid":"<member-uuid>"}],"permission_codes":[]}}
```

占位符表示实际响应中的内部 UUID，不作为页面显示名称。API Key／STS 服务凭据不是这个浏览器 context 接口的用户登录凭据。

## 页面验收矩阵

| 动作与入口 | 成功判定 | 失败处理 |
| --- | --- | --- |
| 在 Core 登录，打开基础插件介绍页 `/_p/com.powerx.plugins.base/admin/intro` 并刷新 | 实际用户身份通过验证，未调用 scoped refresh，页面继续可用 | 看会话提示、context HTTP 状态、Core auth:applied |
| 使用当前 locale 的会话提示重试按钮 | 等待新的验证结果，不在 300ms 后盲重试 | 8s 无结果明确超时，检查注入与 Promise |
| Core 退出，再用另一用户登录 | 旧记录／旧连接被清，新用户仅收到新操作结果 | 检查 late 401／context 是否影响了新状态 |
| 两个宿主标签页同时触发续期 | 插件仅向宿主请求，各自消费正确新快照 | 插件不能持有 refreshToken 或自行并发 refresh |
| 执行无权限操作 | 403 原样返回，不进入刷新循环 | 修正授权，不重放给别的用户 |
| 连接 WS／SSE 后退出／换租户 | 关闭旧连接，旧事件不进入新用户页面 | 检查 session 订阅与资源 dispose |
| 查看头像／图片，下载授权资源 | 资源实际加载且不破坏宿主认证 | 区分 Core 资源代理与 SDK 请求，不用匿名入口 200 代替验收 |

独立运行回归：打开 `http://127.0.0.1:3131/users/login`，按插件自己的账号登录、过期续期和退出。即使后端 Provider 是 Delegated，也不使用 iframe 桥接来管理这个独立会话。

## 自动化与范围

`host-session-v2.spec.ts` 覆盖候选状态、旧消息、等待合并、有限重试、退出隔离、Root scope、WS/SSE；`bridge-v2-interoperability.spec.ts` 使用 Core 注入脚本夹具验证 applied/rejected。运行命令与浏览器连接缺口见 [总览](guide.md)。
