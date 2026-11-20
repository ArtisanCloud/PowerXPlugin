---
name: sts
description: 基于内嵌规则 的 STS 出站访问规范。用于校对后端 STS 客户端、前端禁用与 token TTL 范围。
---

# STS Outbound Access

## 步骤

1) 打开 `本文件内嵌规则`。
2) 在后端检查 STS 客户端封装与配置。
3) 在前端确认无 STS 交换调用。

## 核对点

- STS 客户端仅存在于后端 `backend/internal/infra/sts/**`。
- 前端 `web-admin/**` 禁止调用 `_internal/sts/exchange`。
- Token TTL ≤ 300s 且带 scope。


## 指南

# STS — Plugin Outbound Access Guide

> 插件调用宿主 PowerX 的**唯一合法**方式：短期凭证（STS）交换与使用。  
> 目标：最小权限、最短有效期、显式失败与审计（PG-STS-001）。

## 1. 交换端点与语义

- **交换地址**：`/_p/_internal/sts/exchange`（由宿主暴露）  
- **输入**：插件当前会话/上下文（由宿主反代注入并在后端持有）  
- **输出**：`access_token`（短期） + `expires_in` + 可选 `scopes`

> 插件侧以 **Server-to-Server** 方式请求；不得从浏览器或前端直接交换 STS。

## 2. Token 要求

- **有效期**：≤ 5 分钟（默认 300s）  
- **权限**：必须带 **Scope**（最小权限）  
- **承载**：后续请求使用 `Authorization: Bearer STS.***`

## 3. 客户端实现建议

- **封装**：`backend/internal/infra/sts/client.go`（或 `client_gen.go`）  
- **策略**：  
  - 失败回退：过期 → 重新交换；网络错误 → 指数退避重试（最多 N 次）  
  - 缓存：内存缓存 token，基于过期时间提前刷新（如剩余 <60s 时刷新）  
  - 并发：使用单航班（singleflight）避免风暴  
  - 审计：成功/失败均记录结构化日志（不打印敏感值）

## 4. 典型调用流程

```

Service → STS Client (ensure token) → Host API (with Bearer STS.***)

```

- **错误处理**：  
  - 401/403：Scope 或 token 失效 → 记录并切换降级逻辑  
  - 5xx：宿主暂时不可用 → 重试（带抖动），超出阈值快速失败  
- **观测**：记录 `tenant_uuid/request_id/plugin_id`，便于宿主端关联追踪

## 5. 安全注意事项

- 不在前端/浏览器交换或存储 STS  
- 不写入日志/metrics 的敏感正文  
- 不持久化 STS（仅内存缓存）  
- 不把 STS 透传给第三方（只用于访问宿主 PowerX）

## 6. 与配置/环境联动

- 环境变量：`POWERX_STS_ENDPOINT`（可选，默认宿主固定路径）  
- 允许通过 `POWERX_STS_SCOPES` 明确本插件需要的最小权限清单，以便运营审计

## 7. 合规清单（Checklist）

- [ ] 仅通过 `/_p/_internal/sts/exchange` 交换 STS  
- [ ] Token TTL ≤ 300s，带 Scope  
- [ ] 有缓存/刷新/单航班/退避重试策略  
- [ ] 结构化日志（不含敏感正文）  
- [ ] 仅用于宿主 API；不在前端/浏览器层使用  
- [ ] 失败显式、可审计、可观测

（相关 Gate：PG-STS-001）

## 规则（内嵌）

### sts.yaml

```yaml
# STS — Outbound Access
# Gate: PG-STS-001
kind: ruleset
id: sts
version: 1.0.0
summary: "插件对宿主的出站访问必须通过短期凭证（STS）交换与使用"
checks:
  - id: sts-client-exists
    desc: 存在 STS 客户端封装且只在后端使用
    assert:
      type: path
      target:
        - backend/internal/infra/sts/**
  - id: sts-no-frontend
    desc: 不得在前端/浏览器层发起 STS 交换
    assert:
      type: fileban
      target: web-admin/**
      pattern:
        - "_internal/sts/exchange"
  - id: sts-token-ttl
    desc: Token TTL ≤ 300s，带 scope
    assert:
      type: config
      target: backend/config/**
      query: ensure_sts_ttl_le_300_and_scoped()
```
