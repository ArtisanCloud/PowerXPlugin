# PowerX Publish Hub · 离线发布指南

本指南详细说明如何为网络隔离环境创建 `.pxp` 离线包，通过离线上传审核链路分发到租户。如需跳过 Marketplace，直接在 PowerX 安装 dist/zip/`.pxp` 包，请参考《[本地安装指南](./local-install.md)》。

> ⚠️ 离线打包前请完成《[能力注册与暴露指南](./capabilities.md)》中的流程并确认能力状态为 approved，避免 `px-plugin dist --target offline` 因 capability 未审核而失败。

**适用对象**: 插件开发者 (plugin_developer) + 运维团队 (platform_ops)
**预计耗时**: 15 分钟（打包）+ 1 个工作日（审核 SLA）
**相关 SLA**: 离线审核 ≤ 1 个工作日

---

## 前置准备

### 1. 环境要求

| 组件 | 版本要求 | 验证命令 |
|------|----------|----------|
| Node.js | ≥ 18.0 | `node --version` |
| npm | ≥ 9.0 | `npm --version` |
| Go | ≥ 1.24 | `go version` |
| OpenSSL | ≥ 3.0 | `openssl version` |

### 2. 依赖安装

```bash
# 根目录依赖
npm install

# CLI 依赖
cd tools/cli
npm install

# 返回根目录
cd ../..
```

### 3. 权限验证

#### 3.1 开发者权限

确保您的账号具备 `plugin_developer` 角色权限：
- `publish:submit` - 提交发布
- `plugin:view` - 查看插件信息
- `system:view_logs` - 查看系统日志

#### 3.2 运维权限（离线上传）

运维团队需要 `platform_ops` 角色权限：
- `admin:configure` - 配置系统设置
- `admin:view` - 查看管理面板
- `marketplace:review` - 审核 Marketplace 提交
- `marketplace:offline` - 处理离线包

### 4. 签名材料准备

#### 4.1 使用 PEM 证书签名

```bash
# 生成 RSA 私钥（如果还没有）
openssl genrsa -out cert.pem 2048

# 生成自签名证书（用于开发/测试）
openssl req -new -x509 -key cert.pem -out cert.pem -days 365

# 验证证书
openssl x509 -in cert.pem -text -noout
```

**注意事项**:
- 生产环境应使用受信任的 CA 颁发的证书
- 私钥文件权限必须为 600（仅所有者可读写）
- 证书有效期建议 ≥ 1 年

#### 4.2 使用 KMS 密钥签名

```bash
# 使用 AWS KMS（示例）
export AWS_REGION="us-east-1"
export KMS_KEY_ID="arn:aws:kms:us-east-1:123456789012:key/your-key-id"

# 使用 Azure Key Vault（示例）
export AZURE_KEY_VAULT_NAME="your-vault"
export AZURE_KEY_NAME="your-key"

# 使用 GCP KMS（示例）
export GCP_PROJECT_ID="your-project"
export GCP_KEY_RING="your-ring"
export GCP_KEY_NAME="your-key"
```

### 5. 环境变量配置

```bash
# 离线签名方式（二选一）
export PX_OFFLINE_SIGNING_METHOD="pem"  # 或 "kms"

# PEM 证书路径
export PX_SIGNING_CERT_PATH="./cert.pem"
export PX_SIGNING_KEY_PATH="./cert.pem"

# KMS 配置（如果使用）
export PX_KMS_PROVIDER="aws"  # aws, azure, gcp
export PX_KMS_KEY_ID="your-key-id"

# 离线包输出目录
export PX_OFFLINE_ARTIFACT_DIR="./dist"
```

### 6. 配置文件检查

#### 6.1 dist.config.yml

```yaml
artifacts:
  - path: "dist/manifest.json"
  - path: "dist/backend/**"
  - path: "dist/web-admin/**"
  - path: "dist/migrations/**"
  - path: "dist/assets/**"

signing:
  method: "rsa-pss"  # 或 "rsa-oaep"
  hash: "sha256"
  key_size: 2048

compression:
  algorithm: "gzip"
  level: 6

integrity:
  ignore:
    - "**/*.tmp"
    - "**/.DS_Store"
```

#### 6.2 publish.yml（离线配置）

```yaml
channels:
  - name: offline
    rollout:
      batches:
        - percentage: 100
          startAt: "2025-11-07T00:00:00Z"

tenantFilters:
  enabled: true
  tenants:
    - "offline-tenant-001"
    - "offline-tenant-002"

autoUpgrade: false
rollbackPlan:
  enabled: true
  previousVersion: "1.3.0"
```

---

## 执行流程

### 步骤 1: 代码检查与构建

```bash
# 完整构建
npm run lint
npm test
go test ./...
npm run build

# 生成 manifest
npm run generate:manifest
```

**验证点**:
- ✅ 所有检查通过
- ✅ `dist/` 目录包含所有必要文件
- ✅ `dist/manifest.json` 格式正确

### 步骤 2: 执行离线打包（px-plugin pack）

`px-plugin pack` 基于 specs/004-publish-hub-spec 的新实现，会生成 `.pxp`、`integrity.txt`、`manifest.signature`、`release.manifest.json` 以及审计/报告文件，统一写入 `artifacts/` 目录。

#### 2.1 使用 PEM 证书/Marketplace 公钥

```bash
px-plugin pack \
  --manifest ./dist/manifest.json \
  --artefact ./dist \
  --output ./artifacts \
  --channel offline \
  --notes "隔离租户发版" \
  --sign ./cert.pem \
  --key-id marketplace-prod
```

- `--sign` 指向包含 Marketplace 公钥的 PEM 文件（或经审批的发行证书）。
- `--key-id` 用于在 `release.manifest.json` 中标记签名/加密材料，便于审核溯源。
- 可多次传入 `--artefact` 以追加 backend、web-admin、assets 等目录。

#### 2.2 使用 KMS 密钥签名/封装

```bash
px-plugin pack \
  --manifest ./dist/manifest.json \
  --artefact ./dist \
  --output ./artifacts \
  --channel offline \
  --notes "隔离租户发版" \
  --kms-key-id $KMS_KEY_ID \
  --kms-provider aws \
  --key-id marketplace-prod
```

- `--kms-key-id` 对接云厂商密钥（aws/azure/gcp），CLI 会调用各自 SDK 生成 RSA-OAEP 封装。
- `--kms-provider`（可选）显式指定厂商，默认 `aws`。

**参数说明**:
- `--manifest` - manifest 路径，会写入 `.pxp` 与完整性列表。
- `--artefact <path>` - 要打包的目录/文件，可重复。
- `--output` - 输出目录（默认 `./dist`，推荐设置为 `./artifacts`）。
- `--channel` - 记录在 `release.manifest.json`，便于审核查看（建议 `offline`）。
- `--notes` - 发布备注，同步到离线上传表单。
- `--sign` / `--kms-key-id` - 选择 PEM 或 KMS 模式。
- `--key-id` - Marketplace 端识别密钥/证书的 ID，需与运维约定。

### 步骤 3: 验证输出文件

离线打包完成后，`artifacts/` 目录应包含以下文件：

```
artifacts/
├── demo-plugin-1.4.0.pxp           # 离线包（加密 + 密钥封装）
├── integrity.txt                   # 完整性校验列表
├── manifest.signature              # manifest 签名
├── release.manifest.json           # Plan/导入使用的摘要
├── report.json                     # 打包报告
└── dist-audit.log                  # 审计日志（payload 中仍记录 dist/audit.log）
```

#### 3.1 检查 .pxp 包

```bash
# 查看包大小
ls -lh artifacts/*.pxp

# 解压验证（需要密钥）
# 注意：离线包使用对称密钥加密，密钥已用 Marketplace 公钥封装
```

#### 3.2 验证 integrity.txt

```bash
# 查看完整性列表
cat artifacts/integrity.txt
```

**示例内容**:
```
# Offline Package Integrity Report
# Generated: 2025-11-07T08:42:12.000Z
# Plugin: demo-plugin
# Version: 1.4.0

sha256:dist/manifest.json: abc123...
sha256:dist/backend/main.go: def456...
sha256:dist/web-admin/index.js: ghi789...
```

#### 3.3 验证 manifest.signature

```bash
# 查看签名信息
cat artifacts/manifest.signature
```

**示例内容**:
```json
{
  "algorithm": "rsa-pss-sha256",
  "signature": "base64-encoded-signature",
  "cert_fingerprint": "SHA256:...",
  "signed_at": "2025-11-07T08:42:12.000Z"
}
```

#### 3.4 检查 release.manifest.json

```bash
cat artifacts/release.manifest.json | jq
```

**示例内容**:
```json
{
  "channel": "offline",
  "notes": "隔离租户发版",
  "package": "demo-plugin-1.4.0.pxp",
  "integrityFile": "integrity.txt",
  "generatedAt": "2025-11-07T08:42:12.000Z",
  "keyId": "marketplace-prod"
}
```

将该文件随同 `.pxp` 附在审批工单，可帮助审核员快速定位产物、渠道与密钥来源。

#### 3.5 检查报告文件

```bash
# 查看打包报告
cat artifacts/report.json
```

**示例内容**:
```json
{
  "packageId": "demo-plugin-1.4.0.pxp",
  "createdAt": "2025-11-07T08:42:12.000Z",
  "size": 15728640,
  "files": 142,
  "compression": "gzip",
  "encryption": "aes-256-gcm",
  "signature": {
    "method": "rsa-pss",
    "hash": "sha256",
    "cert_fingerprint": "SHA256:..."
  },
  "integrity": {
    "total_files": 142,
    "verified_files": 142,
    "failed_files": 0
  }
}
```

#### 3.6 检查审计日志

```bash
# 查看审计日志（pack 输出 dist-audit.log，payload metadata 仍指向 dist/audit.log）
cat artifacts/dist-audit.log
```

**示例内容**:
```
2025-11-07T08:42:12.000Z INFO offline打包开始 pluginId=demo-plugin version=1.4.0
2025-11-07T08:42:13.000Z INFO 生成对称密钥 algorithm=aes-256-gcm key_size=256
2025-11-07T08:42:14.000Z INFO 加密 artefact encryption=aes-256-gcm
2025-11-07T08:42:15.000Z INFO 封装密钥 wrapper=rsa-oaep
2025-11-07T08:42:16.000Z INFO 签名 manifest algorithm=rsa-pss-sha256
2025-11-07T08:42:17.000Z INFO 生成完整性列表 total_files=142
2025-11-07T08:42:18.000Z INFO 离线打包完成
```

**验证点**:
- ✅ `.pxp` 包大小 < 300MB
- ✅ `integrity.txt` 包含所有文件的哈希
- ✅ `manifest.signature` 格式正确
- ✅ `report.json` 显示所有检查通过
- ✅ `audit.log` 记录完整操作链

### 步骤 4: 离线上传（运维执行）

⚠️ **此步骤由运维团队执行**

#### 4.1 使用 CLI 预注册离线包（px-plugin import --offline）

004-publish-hub-spec 要求运维在上传前先通过 CLI 把 `.pxp` 与完整性/签名材料注册到 Marketplace，生成 `reviewQueueId` 与审计痕迹：

```bash
export PX_MARKETPLACE_API_URL="https://api.powerx.dev"   # 如 CI 中已配置可省略

px-plugin import --offline \
  --pkg ./artifacts/demo-plugin-1.4.0.pxp \
  --integrity ./artifacts/integrity.txt \
  --signature ./artifacts/manifest.signature \
  --whitelist tenant-a,tenant-b \
  --notes "隔离租户发版"
```

- `--pkg`：`px-plugin pack` 生成的 `.pxp`。
- `--integrity` / `--signature`：与包绑定的校验文件。
- `--whitelist`：逗号分隔或多次传入，CLI 会序列化为数组。
- `--notes`：补充信息，将显示在审核页面。
- CLI 默认读取 `PX_MARKETPLACE_API_URL`；如需覆盖，可在命令前设置环境变量。

示例响应：

```json
{
  "reviewQueueId": "offline-queue-1730979100123",
  "status": "pending_review",
  "whitelist": ["tenant-a", "tenant-b"],
  "submittedAt": "2025-11-07T09:00:00.000Z"
}
```

若目标环境网络隔离，仍可直接使用 Admin UI（见下文），但建议事后使用 CLI 同步记录生成的 `reviewQueueId`。

#### 4.2 登录 Marketplace 离线入口

访问 Admin 控制台：
- URL: `https://admin.powerx.dev/marketplace/offline-review`
- 角色要求: `platform_ops` 或 `marketplace_reviewer`

#### 4.3 上传离线包

1. 点击 "上传离线包" 按钮
2. 选择 `artifacts/demo-plugin-1.4.0.pxp`
3. 上传 `artifacts/integrity.txt`
4. 上传 `artifacts/manifest.signature`
5. 上传 `artifacts/release.manifest.json`
6. 上传 `artifacts/report.json`
7. 上传 `artifacts/dist-audit.log`（或 CLI 输出的 `dist/audit.log`，二者任选其一）

#### 4.4 配置租户白名单

在 "目标租户" 字段中：
- 输入租户 ID（每行一个）
- 或上传 CSV 文件批量导入
- 示例：
  ```
  offline-tenant-001
  offline-tenant-002
  offline-tenant-003
  ```

#### 4.5 提交审核

点击 "提交审核" 按钮，系统会：
1. 验证 `.pxp` 包完整性
2. 解封对称密钥
3. 验证 RSA-PSS 签名
4. 检查哈希一致性
5. 入队离线审核

---

## 内部执行流程（开发者参考）

### 1. 密钥生成与封装

**文件位置**: `tools/cli/src/lib/security/keyEnvelope.ts`

执行流程：
1. 生成 256 位对称密钥（用于加密 `.pxp`）
2. 使用 AES-256-GCM 加密 artefact
3. 使用 Marketplace 公钥（RSA-OAEP）封装对称密钥
4. 将封装后的密钥随包上传

### 2. 离线打包

**文件位置**: `tools/cli/src/lib/dist/offlinePackager.ts`

执行流程：
1. 收集所有 artefact 文件
2. 计算每个文件的 SHA-256 哈希
3. 生成 `integrity.txt` 完整性列表
4. 使用对称密钥加密 `.pxp` 包
5. 使用 RSA-PSS 对 `manifest.json` 签名
6. 生成 `manifest.signature` 文件
7. 创建 `report.json` 打包报告
8. 记录 `dist-audit.log` 审计日志（`.pxp` metadata 仍输出 `dist/audit.log` 以兼容旧工具）
9. 写出 `release.manifest.json`（`pack.ts` 追加的摘要文件），供 CLI/运维在导入和工单中引用

### 3. 离线验证

**文件位置**: `framework/backend/go/runtime/marketplace/services/offline_validator.go`

Marketplace 端验证流程：
1. 使用私钥解封对称密钥
2. 使用对称密钥解密 `.pxp` 包
3. 验证 RSA-PSS 签名
4. 检查 `integrity.txt` 中每个文件的哈希
5. 验证证书链和有效期
6. 记录验证结果

### 4. 审核入队

**文件位置**: `framework/backend/go/runtime/marketplace/handlers/offline_upload.go`

执行流程：
1. 创建离线审核记录
2. 生成 `reviewQueueId`
3. 记录 SLA 开始时间
4. 发送 `plugin.offline.submitted` 事件
5. 加入离线审核队列

---

## 离线审核流程（审核员参考）

### 1. 审核员登录

访问 Marketplace 审核界面：
- URL: `https://admin.powerx.dev/marketplace/offline-review`
- 角色要求: `marketplace_reviewer`

### 2. 查看待审核包

在审核队列中查看：
- `reviewQueueId` - 审核队列 ID
- `packageId` - 离线包 ID
- `pluginId` - 插件 ID
- `version` - 版本号
- `submitter` - 提交者
- `submittedAt` - 提交时间
- `slaDeadline` - SLA 截止时间

### 3. 审核检查清单

#### 3.1 完整性检查

- [ ] `integrity.txt` 存在且格式正确
- [ ] 所有文件哈希验证通过
- [ ] 无缺失或多余文件
- [ ] 文件大小符合预期

#### 3.2 签名验证

- [ ] `manifest.signature` 格式正确
- [ ] RSA-PSS 签名有效
- [ ] 证书链验证通过
- [ ] 证书未过期
- [ ] 证书指纹匹配

#### 3.3 包内容检查

- [ ] `manifest.json` 格式正确
- [ ] 版本号符合语义化规范
- [ ] 权限声明合理
- [ ] 无敏感信息泄露
- [ ] 依赖关系完整

#### 3.4 安全检查

- [ ] 无已知安全漏洞
- [ ] 无恶意代码
- [ ] 加密实现正确
- [ ] 密钥封装安全

#### 3.5 租户白名单

- [ ] 目标租户已正确配置
- [ ] 租户存在且状态正常
- [ ] 租户有权限使用此插件

### 4. 执行审核操作

#### 4.1 批准

```bash
# API 方式
curl -X POST "https://api.powerx.dev/marketplace/offline/$REVIEW_QUEUE_ID/approve" \
  -H "X-Powerx-User-Id: $USER_ID" \
  -H "X-Powerx-Role: marketplace_reviewer" \
  -d '{
    "reason": "审核通过",
    "notes": "所有检查项通过"
  }'
```

批准后：
1. 系统广播 `plugin.offline.approved` 事件
2. 目标租户管理员收到通知
3. 租户可在 Admin 中看到新版本

#### 4.2 拒绝

```bash
# API 方式
curl -X POST "https://api.powerx.dev/marketplace/offline/$REVIEW_QUEUE_ID/reject" \
  -H "X-Powerx-User-Id: $USER_ID" \
  -H "X-Powerx-Role: marketplace_reviewer" \
  -d '{
    "reason": "签名验证失败",
    "details": "manifest.signature 与 manifest.json 不匹配"
  }'
```

拒绝后：
1. 系统广播 `plugin.offline.rejected` 事件
2. 开发者收到失败通知
3. 可查看拒绝原因并重新提交

---

## SLA 监控

### 目标指标

| 指标 | 目标值 | 监控方式 |
|------|--------|----------|
| 离线审核时间 | ≤ 1 个工作日（1440 分钟） | `plugin_offline_approval_duration_minutes` |
| 离线包验证成功率 | 100% | `plugin_offline_validation_success_rate` |
| 签名验证成功率 | 100% | `plugin_offline_signature_verify_rate` |

### 查看 SLA 仪表板

**Grafana URL**: https://grafana.powerx.dev/d/publish-hub-sla

查看面板：
1. 离线审核时延（95th percentile）
2. 离线包验证成功率
3. 签名验证统计
4. 告警状态总览

### 告警处理

如果 SLA 超时，会触发以下告警：

**告警名称**: `PublishOfflineSLAExceeded`
- **触发条件**: `plugin_offline_approval_duration_minutes > 1440m for 1h`
- **级别**: Warning
- **处理**: 参见 `docs/operations/publish-hub-sla.md` 第 133-223 行

**通知渠道**:
- Slack: `#powerx-publish-hub`
- PagerDuty: On-call 工程师
- Email: `ops@powerx.dev`

---

## 故障排除

### 开发者常见问题

#### 问题 1: `.pxp` 加密失败

**症状**:
```
ERROR: failed to encrypt .pxp package
```

**原因**:
- 对称密钥生成失败
- Marketplace 公钥无效
- 磁盘空间不足

**解决方案**:
```bash
# 1. 检查磁盘空间
df -h

# 2. 验证 Marketplace 公钥
openssl rsa -pubin -in $PX_MARKETPLACE_PUBLIC_KEY -text -noout

# 3. 重新生成密钥
rm -f ~/.powerx/cli/symmetric.key
px-plugin pack --manifest ./dist/manifest.json --artefact ./dist --sign ./cert.pem
```

#### 问题 2: RSA-PSS 签名失败

**症状**:
```
ERROR: failed to sign manifest
```

**原因**:
- 私钥文件不存在
- 私钥格式错误
- 私钥权限不足

**解决方案**:
```bash
# 1. 检查私钥文件
ls -l ./cert.pem

# 2. 设置正确权限
chmod 600 ./cert.pem

# 3. 验证私钥
openssl rsa -in ./cert.pem -check

# 4. 重新签名
px-plugin pack --manifest ./dist/manifest.json --artefact ./dist --sign ./cert.pem
```

#### 问题 3: KMS 签名失败

**症状**:
```
ERROR: KMS signature failed: access denied
```

**原因**:
- IAM 权限不足
- KMS 密钥不存在
- 网络连接问题

**解决方案**:
```bash
# 1. 检查 AWS 凭证
aws sts get-caller-identity

# 2. 检查 KMS 权限
aws kms describe-key --key-id $KMS_KEY_ID

# 3. 测试签名
aws kms sign --key-id $KMS_KEY_ID --message file://manifest.json --output text
```

#### 问题 4: 完整性验证失败

**症状**:
```
ERROR: integrity check failed: hash mismatch for dist/backend/main.go
```

**原因**:
- 文件在打包后被修改
- 文件系统问题
- 压缩算法不一致

**解决方案**:
```bash
# 1. 重新构建
rm -rf dist/
npm run build

# 2. 验证文件哈希
sha256sum dist/backend/main.go

# 3. 重新打包
px-plugin pack --manifest ./dist/manifest.json --artefact ./dist --sign ./cert.pem
```

#### 问题 5: 包体积超限

**症状**:
```
ERROR: package size exceeds 300MB limit
```

**原因**:
- Artefact 体积过大
- 未启用压缩
- 包含不必要文件

**解决方案**:
```bash
# 1. 检查包大小
du -h dist/

# 2. 排除不必要文件
# 更新 dist.config.yml
integrity:
  ignore:
    - "**/*.tmp"
    - "**/.DS_Store"
    - "**/node_modules/**"
    - "**/.git/**"

# 3. 启用压缩
# 更新 dist.config.yml
compression:
  algorithm: "gzip"
  level: 9

# 4. 重新打包
px-plugin pack --manifest ./dist/manifest.json --artefact ./dist --sign ./cert.pem
```

### 运维常见问题

#### 问题 6: 离线包上传失败

**症状**:
```
ERROR: failed to upload offline package
```

**原因**:
- 网络连接问题
- 包文件损坏
- 权限不足

**解决方案**:
```bash
# 1. 验证包完整性
px-plugin verify --package ./artifacts/demo-plugin-1.4.0.pxp

# 2. 检查文件权限
chmod 644 ./artifacts/*.pxp
chmod 644 ./artifacts/*.txt
chmod 644 ./artifacts/*.signature

# 3. 重新上传
```

#### 问题 7: 密钥解封失败

**症状**:
```
ERROR: failed to unwrap symmetric key
```

**原因**:
- Marketplace 私钥错误
- 密钥封装数据损坏
- 算法不匹配

**解决方案**:
1. 验证 Marketplace 私钥
2. 重新上传包
3. 联系开发团队检查密钥封装逻辑

#### 问题 8: 租户白名单无效

**症状**:
```
ERROR: tenant whitelist validation failed
```

**原因**:
- 租户 ID 不存在
- 租户状态异常
- 格式错误

**解决方案**:
```bash
# 1. 验证租户存在
curl -H "X-Powerx-User-Id: $USER_ID" \
     $PX_MARKETPLACE_API_URL/tenants/$TENANT_ID

# 2. 检查租户状态
curl -H "X-Powerx-User-Id: $USER_ID" \
     $PX_MARKETPLACE_API_URL/tenants/$TENANT_ID/status
```

---

## 安全注意事项

### 1. 证书管理

- 私钥文件权限必须为 600
- 定期轮换签名证书（建议 1 年）
- 使用强密码保护私钥
- 不要在代码仓库中提交私钥

### 2. 包加密

- 离线包必须使用对称密钥加密
- 对称密钥必须用 Marketplace 公钥封装
- 密钥有效期 ≤ 24 小时
- 加密算法使用 AES-256-GCM

### 3. 签名验证

- 使用 RSA-PSS 或 ECDSA 签名
- 哈希算法使用 SHA-256 或更强
- 验证证书链和有效期
- 记录签名验证日志

### 4. 传输安全

- 使用 HTTPS 传输
- 验证 TLS 证书
- 启用 mTLS（如适用）
- 记录所有传输日志

### 5. 审计日志

- 保留审计日志 ≥ 180 天
- 记录所有关键操作
- 定期审计访问日志
- 监控异常活动

---

## 最佳实践

### 开发者最佳实践

- [ ] 使用语义化版本号
- [ ] 提供详细的发布说明
- [ ] 启用所有安全检查
- [ ] 保持证书有效
- [ ] 定期更新依赖
- [ ] 测试离线包完整性
- [ ] 避免在包中包含敏感信息

### 运维最佳实践

- [ ] 定期检查离线审核队列
- [ ] 及时处理超时审核
- [ ] 验证租户白名单
- [ ] 监控签名失败率
- [ ] 备份重要离线包
- [ ] 定期审计审核记录
- [ ] 保持 Marketplace 私钥安全

### 安全最佳实践

- [ ] 启用所有安全检查项
- [ ] 定期轮换证书
- [ ] 监控异常上传
- [ ] 审计权限使用
- [ ] 加密敏感数据
- [ ] 备份审计日志
- [ ] 及时修复安全漏洞

---

## 相关资源

### 文档
- [Quickstart 指南](../../004-publish-hub-spec/quickstart.md)
- [在线发布指南](./online.md)
- [Marketplace 审核指南](./marketplace-review.md)
- [SLA 运行手册](../../operations/publish-hub-sla.md)
- [安全文档](../../security/publish-hub.md)
- [RBAC 指南](../../operations/publish-hub-rbac.md)

### 代码位置
- CLI 离线命令: `tools/cli/src/commands/dist.ts`
- 离线打包器: `tools/cli/src/lib/dist/offlinePackager.ts`
- 密钥封装: `tools/cli/src/lib/security/keyEnvelope.ts`
- 离线验证: `framework/backend/go/runtime/marketplace/services/offline_validator.go`
- 离线上传: `framework/backend/go/runtime/marketplace/handlers/offline_upload.go`
- 离线 SLA: `framework/backend/go/runtime/marketplace/services/offline_sla_tracker.go`

### 监控
- Grafana 仪表板: https://grafana.powerx.dev/d/publish-hub-sla
- Prometheus: https://prometheus.powerx.dev
- 告警管理: https://alertmanager.powerx.dev

### 支持
- Slack: `#powerx-publish-hub`
- 邮箱: `ops@powerx.dev`
- 文档反馈: https://github.com/ArtisanCloud/PowerXPlugin/issues
