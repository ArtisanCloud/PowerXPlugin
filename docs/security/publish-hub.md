# Publish Hub Security Notes

1. **Artefact 信任链**：所有 `.pxp` 必须通过签名 + 加密 (`pxp-schema.yaml`)，Marketplace 只接受由官方证书或 KMS 颁发的密钥。CLI 离线包默认启用 AES-256-GCM，对称密钥使用 Marketplace 公钥封装。
2. **RBAC 与审计**：`framework/backend/go/runtime/common/middleware/rbac_guard.go`、`runtime/admin/middleware/rbac_guard.go` 区分开发者/审核员/租户管理员权限；`runtime/common/audit/log_retention.go` 确保操作日志保留 ≥180 天。
3. **Dev API 安全**：`px-plugin dev` 相关接口必须使用 mTLS + 幂等 `x-reload-id`，并在 SSE 日志中标注 sessionId 以追踪。
4. **SLA 告警**：`config/alerts/publish-hub.yaml` 将在线/离线审核与回滚超时视为安全/运营事件；触发后 Ops 需跟进 `docs/operations/publish-hub-sla.md`。
5. **密钥管理**：`tools/cli/src/lib/security/keyEnvelope.ts` 在 24h 内过期密钥，Marketplace key vault 定期轮换；上传时仅传输 wrapped key，不暴露对称密钥。
6. **mTLS 证书管理**：
   - 证书有效期：客户端证书 90 天，CA 根证书 3 年
   - 证书轮换：建议提前 30 天预警，提前 7 天执行轮换
   - 监控：Grafana 面板 `Security > Certificates` 展示证书过期时间

## mTLS 证书轮换指南

### 概述
本文档描述如何在不停机的情况下轮换 mTLS 证书。Publish Hub 使用双向 TLS 验证 CLI 与 Dev API 之间的通信。

### 证书结构
- **CA 根证书** (`ca.crt`): 颁发机构证书，有效期 3 年
- **客户端证书** (`client.crt`): CLI 使用的客户端证书，有效期 90 天
- **客户端私钥** (`client.key`): 客户端私钥，与证书匹配

### 轮换准备

#### 1. 检查证书有效期
```bash
# 检查证书过期时间
openssl x509 -in ~/.powerx/cli/client.crt -text -noout | grep "Not After"

# 检查证书序列号
openssl x509 -in ~/.powerx/cli/client.crt -text -noout | grep "Serial Number"

# 验证证书链
openssl verify -CAfile ~/.powerx/cli/ca.crt ~/.powerx/cli/client.crt
```

#### 2. 预警检查（提前 30 天）
当证书剩余有效期 ≤ 30 天时：
1. 在 Grafana 中创建告警：`Certificate expires in 30 days`
2. 通知运维团队开始准备轮换
3. 生成新证书（不替换现证书）
4. 在测试环境验证新证书

### 轮换步骤

#### 阶段 1：新证书生成（提前 7 天执行）

1. **生成新的客户端证书**：
```bash
# 创建证书签名请求 (CSR)
openssl req -new -key ~/.powerx/cli/client.key \
  -out client.csr \
  -subj "/CN=PowerX CLI Client/O=ArtisanCloud/C=CN"

# 使用 CA 签发新证书（有效期 90 天）
openssl x509 -req -in client.csr \
  -CA ~/.powerx/cli/ca.crt \
  -CAkey ~/.powerx/cli/ca.key \
  -CAcreateserial \
  -out ~/.powerx/cli/client-new.crt \
  -days 90

# 验证新证书
openssl verify -CAfile ~/.powerx/cli/ca.crt ~/.powerx/cli/client-new.crt

# 清理 CSR
rm client.csr
```

2. **分发新证书**：
```bash
# 复制到所有需要的位置
cp ~/.powerx/cli/client-new.crt ~/.powerx/cli/client.crt

# 更新文件权限
chmod 600 ~/.powerx/cli/client.key
chmod 644 ~/.powerx/cli/client.crt
chmod 644 ~/.powerx/cli/ca.crt
```

3. **验证新证书**：
```bash
# 验证证书链完整性
openssl x509 -in ~/.powerx/cli/client.crt -text -noout | grep "Subject:\|Issuer:\|Not After"

# 测试与 Dev API 的连接
curl --cert ~/.powerx/cli/client.crt \
  --key ~/.powerx/cli/client.key \
  --cacert ~/.powerx/cli/ca.crt \
  https://api.powerx.dev/internal/dev/plugins/ping
```

#### 阶段 2：部署验证

1. **测试环境验证**：
   - 使用新证书启动 CLI
   - 执行 `px-plugin dev --watch` 验证 mTLS 握手
   - 检查 Dev API 日志：`client certificate verification succeeded`
   - 运行完整热加载流程测试

2. **生产环境灰度**：
   - 部署新证书到 10% 的开发者机器
   - 监控错误率：`mtls_handshake_failures_total`
   - 观察 24 小时无异常后继续

3. **全量部署**：
   - 逐步更新所有开发者的证书
   - 每次更新后监控 2 小时
   - 确保所有服务正常工作

#### 阶段 3：旧证书清理（证书过期后）

1. **确认所有客户端已更新**：
```bash
# 检查连接中的证书版本
grep "client certificate" /var/log/powerx/dev-api.log | tail -100

# 统计证书使用情况
grep -o "CN=PowerX CLI Client" /var/log/powerx/dev-api.log | sort | uniq -c
```

2. **撤销旧证书（如需要）**：
```bash
# 生成 CRL (Certificate Revocation List)
echo "生成撤销列表..."
openssl ca -gencrl -out ca.crl -config ca.conf

# 分发 CRL 到所有服务
cp ca.crl /etc/powerx/ssl/
```

### 自动轮换脚本

#### 证书监控脚本 (`scripts/security/monitor-certs.sh`)
```bash
#!/bin/bash
CERT_PATH="${1:-~/.powerx/cli/client.crt}"
THRESHOLD_DAYS=30

# 获取证书过期时间
EXPIRY_DATE=$(openssl x509 -in "$CERT_PATH" -noout -enddate | cut -d= -f2)
EXPIRY_EPOCH=$(date -d "$EXPIRY_DATE" +%s)
NOW_EPOCH=$(date +%s)
DAYS_LEFT=$(( (EXPIRY_EPOCH - NOW_EPOCH) / 86400 ))

# 检查是否需要轮换
if [ $DAYS_LEFT -le $THRESHOLD_DAYS ]; then
  echo "ALERT: Certificate expires in $DAYS_LEFT days!"
  # 发送告警到 Slack/PagerDuty
  curl -X POST -H 'Content-type: application/json' \
    --data "{\"text\":\"🚨 mTLS Certificate expires in $DAYS_LEFT days!\"}" \
    "$SLACK_WEBHOOK_URL"
  exit 1
else
  echo "Certificate valid for $DAYS_LEFT days"
fi
```

#### 证书轮换脚本 (`scripts/security/rotate-certs.sh`)
```bash
#!/bin/bash
set -e

CA_PATH="${CA_PATH:-~/.powerx/cli}"
DAYS="${DAYS:-90}"

echo "开始轮换 mTLS 证书..."

# 生成新证书
NEW_CERT="$CA_PATH/client-new.crt"
NEW_KEY="$CA_PATH/client-new.key"

# 生成私钥
openssl genrsa -out "$NEW_KEY" 2048

# 生成 CSR
openssl req -new -key "$NEW_KEY" \
  -out "$CA_PATH/client.csr" \
  -subj "/CN=PowerX CLI Client/O=ArtisanCloud/C=CN"

# 签发证书
openssl x509 -req -in "$CA_PATH/client.csr" \
  -CA "$CA_PATH/ca.crt" \
  -CAkey "$CA_PATH/ca.key" \
  -CAcreateserial \
  -out "$NEW_CERT" \
  -days $DAYS

# 验证证书
openssl verify -CAfile "$CA_PATH/ca.crt" "$NEW_CERT"

# 替换旧证书
mv "$NEW_CERT" "$CA_PATH/client.crt"
mv "$NEW_KEY" "$CA_PATH/client.key"
mv "$CA_PATH/ca.crt.srl" "$CA_PATH/ca.srl" 2>/dev/null || true

# 清理临时文件
rm "$CA_PATH/client.csr"

# 设置权限
chmod 600 "$CA_PATH/client.key"
chmod 644 "$CA_PATH/client.crt"

echo "证书轮换完成！新证书有效期：$DAYS 天"
openssl x509 -in "$CA_PATH/client.crt" -noout -enddate
```

### 故障排除

#### 常见问题

1. **证书验证失败**：
   ```
   Error: certificate verify failed
   ```
   - 检查 CA 根证书是否正确
   - 验证证书链完整性
   - 确认系统时间正确

2. **私钥不匹配**：
   ```
   Error: private key does not match certificate
   ```
   - 重新生成证书和私钥
   - 确保证书和私钥来自同一次生成

3. **证书已过期**：
   ```
   Error: certificate has expired
   ```
   - 立即执行轮换脚本
   - 检查系统时间同步

4. **mTLS 握手超时**：
   - 检查防火墙规则
   - 验证证书格式（PEM）
   - 查看 Dev API 日志获取详细错误

#### 紧急轮换（证书已过期）

当证书已过期且无法建立连接时：

1. **使用临时证书**（5 分钟有效期）：
```bash
openssl x509 -req -in client.csr \
  -CA ca.crt -CAkey ca.key \
  -CAcreateserial -out client-temp.crt -days 1
```

2. **恢复服务**：
```bash
cp client-temp.crt client.crt
systemctl restart powerx-dev-api
```

3. **尽快执行完整轮换**：5 分钟内完成正式证书轮换

### 监控指标

#### Grafana 面板
- `mtls_cert_expiry_days`: 证书剩余有效期（天）
- `mtls_handshake_success_rate`: mTLS 握手成功率
- `mtls_handshake_failures_total`: 握手失败次数
- `cert_renewal_rate`: 证书续订成功率

#### Prometheus 告警规则
```yaml
- alert: CertificateExpiringSoon
  expr: mtls_cert_expiry_days < 30
  for: 1h
  labels:
    severity: warning
  annotations:
    summary: "mTLS certificate expires in {{ $value }} days"
```

### 安全最佳实践

1. **密钥存储**：
   - 私钥文件权限：600（仅所有者可读写）
   - 证书文件权限：644（所有者可读写，其他用户可读）
   - 使用专用证书目录：`~/.powerx/cli/`

2. **访问控制**：
   - CA 私钥仅运维团队可访问
   - 定期审计证书使用情况
   - 立即撤销离职人员证书

3. **日志审计**：
   - 记录所有证书签发/撤销操作
   - 监控异常握手失败
   - 审计证书分发记录

4. **备份策略**：
   - 每日备份 CA 证书和私钥
   - 异地存储备份文件
   - 测试证书恢复流程
