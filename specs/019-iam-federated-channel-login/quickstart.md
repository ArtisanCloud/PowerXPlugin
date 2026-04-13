# Quickstart: IAM 联邦渠道扫码登录（企微/钉钉/飞书）

## 1. 目标

验证 framework 提供的 federated factory/default providers 可被插件直接复用，并在 skeleton 仅装配接线后跑通扫码登录主链路。

## 2. 前置条件

1. 当前分支为 `019-iam-federated-channel-login`。  
2. 已配置至少一个渠道（推荐企业微信）测试租户参数。  
3. 插件后端与 web-admin 可正常启动。  
4. delegated 场景下可访问宿主鉴权链路。

## 3. 主链路验证

1. 访问“扫码登录”入口，发起 challenge。  
2. 检查 challenge 返回 `state/nonce/expires_at/authorize_url`。  
3. 完成渠道扫码并回调到插件。  
4. 验证登录成功后身份上下文包含 `tenant/user/member/roles/permissions/provider/trace`。  
5. 首次登录验证 JIT 默认策略：仅唯一匹配自动绑定，否则进入管理员处理路径。

## 4. 映射策略验证

1. 在管理端调整角色/部门映射策略版本。  
2. 触发同一成员再次扫码登录。  
3. 验证仅在映射版本变化时重算并生效。

## 5. 风控验证

1. 构造过期 state 回调，预期拒绝并返回风险错误码。  
2. 构造重复 code 回调（replay），预期拒绝并记录风险事件。  
3. 构造跨租户回调，预期拒绝并输出可审计拒绝记录。  
4. 前端展示统一通用失败文案，不暴露风控细节。

## 6. 双模式一致性验证

1. standalone 模式：验证身份上下文结构与错误语义。  
2. delegated 模式：验证以宿主会话/令牌为权威，插件仅输出适配上下文。  
3. 对比两模式字段结构一致。

## 7. 回归命令

```bash
cd framework/backend/go && go test ./...
```

```bash
cd skeleton/backend/go-gin && \
  GOCACHE=$PWD/../../tmp/gocache \
  GOMODCACHE=$PWD/../../tmp/gomodcache \
  go test ./...
```

## 8. 成功指标采样口径（SC-004 / SC-005）

1. **SC-004 基线窗口**：以“联邦登录功能在目标租户灰度开启前连续 7 天”为基线窗口。  
2. **SC-004 观测窗口**：以“灰度开启当日记为 D0，统计 D1-D30”作为观测窗口。  
3. **SC-004 统计口径**：  
   - 密码登录占比 = `password_login_success / total_login_success`。  
   - 仅统计同一租户同一登录入口（排除健康检查、机器人、测试账号）。  
4. **SC-005 基线定义**：以该插件“未复用 framework factory 的历史接入记录”作为自研基线步骤数。  
5. **SC-005 对比定义**：以当前文档标准接入步骤作为复用方案步骤数，按 `减少比例 = (自研步骤 - 复用步骤) / 自研步骤` 计算。  
6. **采样留痕要求**：SC-004 与 SC-005 的原始统计数据、计算过程与结论统一写入 `tmp/019-iam-federated-channel-login-regression.md`。

## 9. SC-005 接入效率度量清单

1. 记录“自研方案”步骤数（含 provider 接口定义、风险校验、callback 接线、上下文输出）。
2. 记录“framework 复用方案”步骤数（仅装配、配置、路由对接）。
3. 逐项对比并标注是否被 framework 复用覆盖。
4. 计算步骤减少比例并附原始计数表。
5. 将对比结论写入回归记录，作为 SC-005 验收依据。

## 10. 常见排障

1. `FEDERATED_PROVIDER_NOT_FOUND`：检查 provider key 是否为 `wecom`/`dingtalk`/`lark`，并确认 bootstrap 已注册。
2. `FEDERATED_INVALID_CHALLENGE`：检查 callback 的 `state/nonce/tenant_uuid` 是否和 challenge 响应一致。
3. `FEDERATED_RISK_*`：前端展示统一失败文案，详细原因通过审计事件与 `reason_code` 排查。
4. delegated 模式失败：优先确认宿主会话可用，再检查插件日志中的 `FEDERATED_AUTH_UNAVAILABLE`。
