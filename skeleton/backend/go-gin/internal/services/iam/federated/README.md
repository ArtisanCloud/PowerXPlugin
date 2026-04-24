# Skeleton Federated 装配说明

本目录仅负责插件运行时对 framework 联邦能力的装配与适配。

## 责任范围

1. 注入 framework provider factory 与配置。
2. 编排插件登录流程中的 service 调用。
3. 输出插件侧统一身份上下文。

## 非责任范围

1. 不重复实现渠道 SDK 底层对接。
2. 不在此目录定义 framework contract。
3. 不绕过 framework 直接实现独立 provider 协议栈。

## 与路由层关系

- `transport/http/public/auth` 调用本目录服务。
- 本目录调用 framework 能力并返回标准化结果给上层。
