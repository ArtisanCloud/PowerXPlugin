---
name: crud-repository
description: 基于内嵌规则 的 Repository 规范。用于新增/修改仓储层读写与多租户事务逻辑。
---

# CRUD Repository

## 步骤

1) 打开 `本文件内嵌规则`。
2) 在 `backend/internal/domain/repository/**` 实现仓储。

## 核对点

- Repository 组合 `BaseRepository` 并通过构造函数返回实例。
- 读写在 `BeginTenantTx` 中执行，并设置 `app.tenant_uuid`。

## 规则（内嵌）

### repository.yaml

```yaml
# Repository 层
kind: ruleset
id: crud_repository
version: 1.0.0
checks:
  - id: repo-base-embed
    desc: Repository 结构体需组合 BaseRepository 并通过构造函数返回本地实例
    assert:
      type: grep
      target: backend/internal/domain/repository/**
      pattern:
        - BaseRepository
  - id: repo-tenant-tx
    desc: 读写在 BeginTenantTx 中执行，并设置 app.tenant_uuid
    assert:
      type: grep
      target: backend/internal/domain/repository/**
      pattern:
        - BeginTenantTx
        - "SET LOCAL app.tenant_uuid"
```
