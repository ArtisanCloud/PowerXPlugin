# PowerX Publish Hub · 离线打包（`.pxp`）指南

`px-plugin dist --target offline` 负责生成 `.pxp` Artefact、完整性列表、签名/加密元数据以及审计日志，本指南介绍命令输入与输出位置。

## 1. 前置条件

1. 完成 `px-plugin build`（或类似构建脚本），确保 backend/frontend artefact 已放入 `dist/`。
2. 准备 `manifest.json` 与 `plugin.yaml`，版本号与权限声明需与在线发布一致。
3. 获取 Marketplace 公钥 (`PX_MARKETPLACE_PUBLIC_KEY`) 及对应 `keyId` 以便加密 `.pxp` 对称密钥。
4. 确保配置文件 `config/feature-flags.yaml` 已启用 `PX_MARKET_OFFLINE_UPLOAD`、`PX_PLUGIN_PUBLISH`。

## 2. 命令示例

```bash
px-plugin dist \
  --manifest ./dist/manifest.json \
  --artefact ./dist/backend/bin/app \
  --artefact ./dist/web-admin/.output \
  --public-key ./certs/marketplace.pub.pem \
  --key-id marketplace-prod-key-1 \
  --output ./release/offline
```

## 3. CLI 行为（实现参考）

- `src/commands/dist.ts` 解析输入路径，调用 `packageOfflineBuild`。
- `src/lib/dist/offlinePackager.ts`：
  - 读取 manifest 并生成 Integrity 列表（`integrity.txt`）。
  - 构建 `.pxp` JSON（包含 manifest、artefact 元数据、integrity、signature、audit 事件）。
  - 写入 `report.json`（统计文件数量、体积）以及 `dist-audit.log`（记录操作人/时间）。
- `src/lib/dist/encryptor.ts` + `src/lib/security/keyEnvelope.ts`：生成随机 AES-256-GCM 密钥并使用 Marketplace 公钥封装，输出 `wrappedKey` 等字段。
- `src/lib/telemetry/emitter.ts` 可在未来扩展 `plugin.offline.package` 事件，用于统计离线产出频率。

## 4. 输出目录结构

```
release/offline/
├── demo-plugin-1.4.0.pxp
├── integrity.txt
├── report.json
└── dist-audit.log
```

`.pxp` 内容遵循 `docs/contracts/artefacts/pxp-schema.yaml` 契约，Marketplace 可直接解密+验证后入库。

## 5. 故障排查

| 现象 | 排查建议 |
|------|----------|
| `marketplace public key is required` | 确保传入 `--public-key` 路径且文件存在。 |
| `changelog file not found`（来自 publish 流程） | 确认 `--notes/--changelog` 路径。 |
| `.pxp` 文件为空或损坏 | 检查 artefact 路径，确保不是空目录，必要时删除旧的 `release/offline` 重新生成。 |
| Marketplace 解密失败 | 确认 `keyId` 与公钥匹配，公钥 PEM 是否包含 BEGIN/END 行。 |
