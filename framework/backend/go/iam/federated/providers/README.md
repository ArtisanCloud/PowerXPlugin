# Providers 注册约定

本目录承载联邦 provider 的注册与扩展规范。

## 注册约定

1. provider key 必须全局唯一（例如 `wecom`/`dingtalk`/`lark`）。
2. provider 必须实现统一 contract（authorize url 构建、code 兑换、identity 解析）。
3. provider 错误必须映射到 framework 标准错误语义。

## 扩展约定

1. 新渠道以子目录形式新增：`providers/<name>/`。
2. 不在 provider 内处理插件本地绑定策略。
3. 回调风险判定交由 challenge/risk 子模块处理。

## 版本化约定

- 对外 contract 的破坏性变更需走版本升级，保障多插件兼容。
