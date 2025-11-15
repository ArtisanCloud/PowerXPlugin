# Data Model — Publish Hub

## Plugin Artefact (.pxp)
| Field | Type | Description | Constraints |
|-------|------|-------------|-------------|
| `manifest` | JSON | 插件 id/version/runtime/permissions/targets | 必须与 Marketplace 记录一致，语义版本递增 |
| `backend_bundle` | Binary | 编译后的 Go 服务/任务产物 | 仅允许签名列表内文件；大小 < 200MB |
| `frontend_bundle` | Binary | Nuxt Admin UI 输出 | 支持多语言，大小 < 100MB |
| `migrations` | Files | 数据迁移脚本 | 需声明幂等、附带回滚描述 |
| `assets` | Files | 可选资源（图标、配置） | 需在 `integrity.txt` 中列出 |
| `integrity.txt` | Text | SHA256 列表（path → hash） | 生成于 dist 流程，上传前必须校验 |
| `manifest.signature` | Binary | 签名+证书链 | 由本地 PEM 或 KMS 产生，Marketplace 验证 |
| `report.json` | JSON | 打包报告（大小、依赖、flag） | 供审核与审计使用 |
| `audit.log` | Text | CLI 产出操作日志 | 审核与溯源必需 |

**Relationships**: 一对一关联 `Publish Request`；同一版本只允许唯一 `.pxp`。

## Publish Request
| Field | Type | Description | Notes |
|-------|------|-------------|-------|
| `publishId` | UUID | CLI 生成的幂等键 | 贯穿在线/离线队列 |
| `pluginId` | String | 插件唯一 ID | 来自 manifest |
| `version` | SemVer | 版本号 | 必须递增并与 `.pxp` 一致 |
| `channel` | Enum(stable,beta) | 发布渠道 | 决定灰度策略 |
| `submitter` | UserRef | CLI 执行者 | 用于审计/告警 |
| `approvalStatus` | Enum(draft,pending,approved,rejected,withdrawn) | 审核状态 | 依据 Marketplace 流程流转 |
| `reviewQueueId` | UUID | Marketplace 队列编号 | 区分在线/离线 SLA |
| `autoUpgrade` | Bool | 是否自动升级订阅租户 | 影响通知/安装策略 |
| `notes` | Text | 发布说明/风险 | 展示给运营与租户 |

**State transitions**: `draft → pending → (approved | rejected)`；`approved → broadcasted → installed`；任意状态可 `withdrawn`。

## Tenant Deployment Record
| Field | Type | Description |
|-------|------|-------------|
| `tenantId` | UUID | 租户编号 |
| `pluginId` | String | 插件 ID |
| `version` | SemVer | 安装版本 |
| `installMethod` | Enum(url,local) | 在线拉取 / 离线上传 |
| `status` | Enum(pending,installing,success,failed,rolled_back) | 当前状态 |
| `rollbackLink` | UUID | 指向先前成功记录 |
| `logsRef` | URL | Admin/Backend 日志位置 |
| `timestamp` | ISO8601 | 操作时间 |

**Relationships**: 多对一 `Publish Request`；回滚时引用同租户上一条 success 记录。

## Offline Integrity Report
| Field | Type | Description |
|-------|------|-------------|
| `integrityFile` | Path | `integrity.txt` 存储位置 |
| `signatureFile` | Path | `manifest.signature` | 
| `report` | JSON | CLI 生成的 dist 报告 |
| `auditLog` | Path | CLI 操作日志 |
| `verdict` | Enum(pass,fail,pending) | Marketplace 校验结果 |

**Relationships**: 一对一 `Plugin Artefact (.pxp)`。

## Dev Hotload Session
| Field | Type | Description |
|-------|------|-------------|
| `sessionId` | UUID | Dev API 注册 ID |
| `tenantId` | UUID | 调试租户 |
| `bundleHash` | SHA256 | 最近一次差异包 hash |
| `reloadSeq` | Int | Reload 次数 |
| `status` | Enum(active,error,stopped) | 当前状态 |
| `logsRef` | URL | Admin SSE / CLI log 链接 |
| `expiresAt` | ISO8601 | 会话失效时间 |

**State transitions**: `active → (active|error)` per reload；`error → active`（修复后）或 `stopped`（delete/register）。
