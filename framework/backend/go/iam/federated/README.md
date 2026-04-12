# Federated IAM 模块边界说明

本目录承载 **framework 可复用** 的联邦登录能力，供多个插件复用。

## 责任范围

1. 定义 provider contract 与统一错误语义。
2. 提供 provider registry/factory 统一注册入口。
3. 提供 challenge/risk 等跨插件通用流程能力。
4. 提供默认渠道 provider 实现（企微/钉钉/飞书）。

## 非责任范围

1. 不处理插件特定页面交互与路由装配。
2. 不承载租户业务定制策略存储。
3. 不直接依赖具体插件的 service/repository 实现。

## 与 skeleton 协作

- skeleton 仅负责装配 framework factory、注入配置、接线路由。
- 任何渠道底层协议对接逻辑应优先放在 framework。
