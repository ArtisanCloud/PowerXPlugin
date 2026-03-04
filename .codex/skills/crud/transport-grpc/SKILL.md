---
name: crud-transport-grpc
description: 基于内嵌规则 的 gRPC 传输规范。用于新增/调整 gRPC handler 与注册入口。
---

# CRUD gRPC Transport

## 步骤

1) 打开 `本文件内嵌规则`。
2) 在 `backend/internal/grpc/**` 实现 handler。
3) 在 `backend/internal/grpc/server/**` 注册服务。

## 核对点

- 入口注册 `Register*ServiceServer`。
- gRPC handler 复用 Service，不含业务逻辑或 DB 直连。

## 规则（内嵌）

### transport_grpc.yaml

```yaml
# gRPC 传输层
kind: ruleset
id: crud_transport_grpc
version: 1.0.0
checks:
  - id: grpc-register
    desc: 统一入口注册 Register*ServiceServer
    assert:
      type: grep
      target: backend/internal/grpc/server/**
      pattern:
        - Register
  - id: grpc-reuse-service
    desc: gRPC handler 复用 Service，不得含业务逻辑
    assert:
      type: ban-symbol
      target: backend/internal/grpc/**
      symbols:
        - direct_db_access
        - long_business_flow
```
