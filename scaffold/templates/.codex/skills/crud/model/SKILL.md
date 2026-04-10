---
name: crud-model
description: 基于内嵌规则 的 CRUD 模型规范。用于新增/调整 Gorm 模型、表结构常量、AutoMigrate 注册时。
---

# CRUD Model

## 步骤

1) 打开 `本文件内嵌规则`。
2) 在 `backend/internal/entity/models/**` 创建或更新模型。
3) 在 `backend/cmd/database/migrate/migrate.go` 注册 AutoMigrate。

## 核对点

- 模型含 `tenant_uuid` 与 `created_at/updated_at`。
- 字段具备 `gorm` 与 `json` 标签。
- `TableName()` 使用 `models.S(...)` 常量，不直接写字符串。

## 规则（内嵌）

### model.yaml

```yaml
# Model（含多租户字段与审计字段）
kind: ruleset
id: crud_model
version: 1.0.0
checks:
  - id: model-tenant-audit
    desc: 模型包含 tenant_uuid 与审计字段（created_at/updated_at）
    assert:
      type: schema
      target: backend/internal/entity/models/**
      schema:
        requires:
          - tenant_uuid
          - created_at
          - updated_at
  - id: model-gorm-json-tags
    desc: 持久化模型必须声明 gorm 列定义与 json 标签
    assert:
      type: grep
      target: backend/internal/entity/models/**
      pattern:
        - "gorm:\""
        - "json:\""
  - id: model-table-constants
    desc: 所有 TableName() 函数必须引用模型常量（backend/internal/entity/models/model.go）
    assert:
      type: grep
      target: backend/internal/entity/models/**
      pattern:
        - "TableName() string"
        - "models.S("
      deny:
        - "models.S(\""
  - id: model-register-migrate
    desc: 所有表结构需在 migrate.go 中通过 AutoMigrate 注册
    assert:
      type: grep
      target: backend/cmd/database/migrate/migrate.go
      pattern:
        - AutoMigrate(
```
