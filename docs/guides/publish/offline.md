# PowerX Publish Hub · 离线上传审核

本章节补充 Marketplace 离线 steward 的操作步骤：

1. 接收开发者提供的 `.pxp`、`integrity.txt`、`manifest.signature`、`dist/audit.log`。
2. 访问后台 `/marketplace/offline` 页面，选择目标租户白名单并上传 artefact。
3. 平台会通过 `offline_upload.go` 记录 upload 并调用 `offline_validator.go` 校验 hash/manifest，一旦通过即加入审核队列。
4. 通过 `offline_sla_tracker.go` 确认审批是否在 1 个工作日内完成，若触发告警，参见 `docs/operations/publish-hub-sla.md`。
