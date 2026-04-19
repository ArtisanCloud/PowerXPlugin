# Challenge/Risk 子模块说明

本子模块用于承载扫码授权挑战值与回调风控的通用能力。

## Challenge 目标

1. 生成并管理 `state`、`nonce`、`ttl`。
2. 提供一次性消费与过期判定能力。
3. 输出统一 trace 字段用于审计关联。

## Risk 目标

1. 识别并拦截 expired/replay/cross-tenant/signature 风险。
2. 输出可区分错误码，供上层统一文案呈现。
3. 输出风险证据字段，供审计与回归验证。

## 设计约束

- 保持与 provider 无关，避免业务耦合。
- 默认实现可被 standalone/delegated 两种模式复用。
