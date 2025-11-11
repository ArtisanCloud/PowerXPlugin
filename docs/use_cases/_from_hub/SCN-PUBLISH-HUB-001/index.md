---
title: SCN-PUBLISH-HUB-001 Usecase Seed Index
scn_id: SCN-PUBLISH-HUB-001
status: Generated
last_reviewed_at: 2025-10-29
---

# PowerX 插件开发与分发全链路 – Usecase Seed Index

> 本文件由 `generate-usecase-seed-index` 自动生成，请勿手工编辑。

- 场景文档：[`../../scenarios/publish/SCN-PUBLISH-HUB-001.md`](../../scenarios/publish/SCN-PUBLISH-HUB-001.md)
- docmap 入口：[`../../_data/docmap.yaml`](../../_data/docmap.yaml)

## Scope: powerx

| Doc ID | Layer | Domain | Optional | Seed | Status |
|--------|-------|--------|----------|------|--------|
| PX-DEV-HOTLOAD-001 | service | dev | 否 | [PX-DEV-HOTLOAD-001](PX-DEV-HOTLOAD-001.md) | 已生成 |
| PX-DEV-HOTLOAD-UI-001 | ui | dev | 否 | [PX-DEV-HOTLOAD-UI-001](PX-DEV-HOTLOAD-UI-001.md) | 已生成 |
| PX-PUBLISH-OFFLINE-001 | service | publish | 否 | [PX-PUBLISH-OFFLINE-001](PX-PUBLISH-OFFLINE-001.md) | 已生成 |
| PX-PUBLISH-OFFLINE-UI-001 | ui | publish | 否 | [PX-PUBLISH-OFFLINE-UI-001](PX-PUBLISH-OFFLINE-UI-001.md) | 已生成 |
| PX-PUBLISH-ONLINE-001 | service | catalog | 否 | [PX-PUBLISH-ONLINE-001](PX-PUBLISH-ONLINE-001.md) | 已生成 |
| PX-PUBLISH-ONLINE-UI-001 | ui | marketplace | 否 | [PX-PUBLISH-ONLINE-UI-001](PX-PUBLISH-ONLINE-UI-001.md) | 已生成 |

## Scope: powerx-marketplace

| Doc ID | Layer | Domain | Optional | Seed | Status |
|--------|-------|--------|----------|------|--------|
| MKP-PUBLISH-OFFLINE-001 | api | marketplace | 否 | [MKP-PUBLISH-OFFLINE-001](MKP-PUBLISH-OFFLINE-001.md) | 已生成 |
| MKP-PUBLISH-ONLINE-001 | api | marketplace | 否 | [MKP-PUBLISH-ONLINE-001](MKP-PUBLISH-ONLINE-001.md) | 已生成 |

## Scope: powerx-plugin

| Doc ID | Layer | Domain | Optional | Seed | Status |
|--------|-------|--------|----------|------|--------|
| PLG-DEV-HOTLOAD-001 | proto | dev | 否 | [PLG-DEV-HOTLOAD-001](PLG-DEV-HOTLOAD-001.md) | 已生成 |
| PLG-PUBLISH-OFFLINE-001 | proto | publish | 否 | [PLG-PUBLISH-OFFLINE-001](PLG-PUBLISH-OFFLINE-001.md) | 已生成 |
| PLG-PUBLISH-ONLINE-001 | proto | publish | 否 | [PLG-PUBLISH-ONLINE-001](PLG-PUBLISH-ONLINE-001.md) | 已生成 |

## Publish Hub 手动补充说明

> 以下内容为手动维护，帮助读者快速跳转到最新的 CLI / 审核 / 安装链路文档（自动生成脚本尚未涵盖）。

- **CLI 入口**：`specs/004-publish-hub-spec/spec.md` + `specs/004-publish-hub-spec/quickstart.md` 描述 `px-plugin dev/publish/dist` 命令、`.pxp` 加密与 Telemetry 要求。
- **Marketplace 审核**：`docs/guides/publish/marketplace-review.md` 收录在线/离线审核流程、SLA 计时器与告警策略，对应实现计划见 `specs/004-publish-hub-spec/plan.md`。
- **租户安装/回滚**：`README.md#publish-hub-链路_cli--审核--安装` 与 `specs/004-publish-hub-spec/tasks.md` 列出 Admin 安装入口、灰度策略与 quickstart 验证步骤。
