# Quickstart

> 目标：在不修改 Nuxt 与 Gin 契约的前提下，验证 Next 管理端全域迁移的联调与回归路径。

## 前置条件

- 位于分支 `012-next-nuxt-align`
- 已具备 Node.js 20+ 与 Go 1.24 环境
- `skeleton/backend/go-gin` 与 `skeleton/web-admin/next` 目录可用

## 1) 启动 Go-Gin 后端（契约基线）

```bash
cd /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/skeleton/backend/go-gin
go run ./cmd/plugin
```

默认后端监听地址以运行配置为准，请确保管理端 API 可通过 `/api/v1` 访问。

## 2) 启动 Next 管理端

```bash
cd /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/skeleton/web-admin/next
npm run dev
```

默认端口为 `3231`。

## 3) 执行核心验证（第一阶段全域）

- 登录与会话：登录成功、会话刷新、过期重定向
- Templates：创建、编辑、克隆、删除
- IAM：成员与角色主链路
- Capabilities：生命周期/注册主链路
- Integration / Operations / Security：主链路可访问并返回一致语义

## 4) 执行双模式一致性验证

- 独立模式：`/api/v1` 访问
- 宿主模式：`/_p/{pluginId}/api/v1` 与 `/_p/{pluginId}/admin/*` 访问
- 关键异常场景：会话过期、权限拒绝、路由切换

## 5) 联调差异归因与门禁

- 任何差异需在 2 个工作日内完成归因并记录到差异记录。
- 发布前必须满足：阻断级与高优先级问题清零。
- 中低优先级问题需包含书面风险评估与处置计划。

## 验证完成标准

- 全部页面域主链路回归通过率 100%
- 双模式主链路 + 关键异常场景通过率 100%
- 未新增任何仅 Next 可用的后端私有接口
- 若修复 Gin，必须有“缺陷确认 + 最小化补丁 + 双端回归”证据链
