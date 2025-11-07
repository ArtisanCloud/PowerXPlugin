# PowerX Publish Hub · 在线发布指南

本指南说明如何使用 `px-plugin publish` 将插件版本推送到 Publish Hub（Marketplace 在线审核链路）。

## 1. 环境准备

1. Node.js 18+、npm 9+、Go 1.24+。
2. 在仓库根目录执行 `npm install`（确保 Nuxt workspace）以及 `cd tools/cli && npm install`（CLI 依赖）。
3. 配置 Marketplace API 访问凭据：
   - `PX_MARKETPLACE_API_URL`
   - `PX_MARKETPLACE_TOKEN`
   - `PX_MARKETPLACE_PUBLIC_KEY`（PEM，用于加密 `.pxp` 密钥）
4. 确保 `manifest.json`/`plugin.yaml` 同步且版本号递增，`publish.yml` 中声明 channel、灰度策略、回滚方案。

## 2. 运行 publish 命令

```bash
px-plugin publish \
  --manifest dist/manifest.json \
  --channel stable \
  --notes ./CHANGELOG.md \
  --receipt ./artifacts/publish-receipt.json
```

命令内部执行：

1. **预检**（`src/lib/publish/precheck.ts`）：验证 manifest 字段、语义化版本、权限重复、Stable 渠道必须提供 notes/changelog。
2. **流水线**（`src/lib/publish/pipeline.ts`）：产生 `publishId`/`versionId`/`reviewQueueId`，拼接 changelog，生成上传地址。
3. **回执**：以 JSON 写入 `publish-receipt.json`（可自定义路径），包含 `publishId`、`reviewQueueId`、`submittedAt`。
4. **Telemetry**：`src/lib/telemetry/emitter.ts` 输出 `plugin.publish` 事件，便于后续指标采集。

### publish-receipt.json 示例

```json
{
  "publishId": "d6b3c1c0-5f07-4bfa-b083-6f8cc9a4b9de",
  "versionId": "demo-plugin-1.4.0",
  "reviewQueueId": "b8b47ad1-5d90-45e3-b3b3-a870e63f9d00",
  "uploadUrl": "https://upload.marketplace.powerx.local/plugins/demo-plugin/uploads/d6b3c1c0-5f07-4bfa-b083-6f8cc9a4b9de",
  "channel": "stable",
  "submittedAt": "2025-11-07T08:42:12.000Z"
}
```

## 3. 常见问题

| 问题 | 处理方式 |
|------|----------|
| `manifest permissions must be an array` | 检查 `plugin.yaml`/`manifest.json` 权限声明格式。 |
| `stable releases must include release notes` | 传入 `--notes` 或在 `publish.yml` 指向 changelog。 |
| Changelog 读取失败 | 确认 `--notes` 或 `--changelog` 路径存在。 |

## 4. 后续步骤

- 完成 publish 命令后，参照 `docs/guides/publish/marketplace-review.md` 查看审核进度。
- 若需离线包或 `.pxp` 产出，请继续阅读 `docs/guides/publish/offline.md`。
