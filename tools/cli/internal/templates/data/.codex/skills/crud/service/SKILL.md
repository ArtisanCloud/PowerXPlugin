---
name: crud-service
description: 基于内嵌规则 的 Service 规范。用于新增/调整 CRUD Service 并复用仓储。
---

# CRUD Service

## 步骤

1) 打开 `本文件内嵌规则`。
2) 在 `backend/internal/services/*/` 新增或修改 service 包。

## 核对点

- 每个资源独立 service 包，供 HTTP/gRPC 复用。
- 只通过 Repository 访问数据，不直接操作 DB 客户端。
- Service 结构体组合 Repository 字段。

## 规则（内嵌）

### service.yaml

```yaml
# Service 层唯一业务源
kind: ruleset
id: crud_service
version: 1.0.0
checks:
  - id: service-dir
    desc: 每个资源拥有独立 service 包，供 HTTP/gRPC 复用
    assert:
      type: path
      target: backend/internal/services/*/
  - id: service-repo-only
    desc: Service 通过 Repository 访问数据，不得直接操作 DB 客户端
    assert:
      type: ban-symbol
      target: backend/internal/services/**
      symbols:
        - direct_db_access
  - id: service-repo-field
    desc: Service 应组合 Repository（如 *FooRepository）而非裸 DB
    assert:
      type: grep
      target: backend/internal/services/**
      pattern:
        - Repository
```
