---
name: crud-migration
description: 基于内嵌规则 的 Migration 规范。用于新增模型迁移脚本与 RLS 策略。
---

# CRUD Migration

## 步骤

1) 打开 `本文件内嵌规则`。
2) 在 `backend/migrations/**` 编写迁移，确保幂等与可回滚。

## 核对点

- 迁移脚本存在且覆盖所有模型。
- 启用 RLS 策略，并包含 tenant_uuid 过滤。

## 规则（内嵌）

### migration.yaml

```yaml
# Migration 纪律
kind: ruleset
id: crud_migration
version: 1.0.0
checks:
  - id: migration-exist
    desc: 所有模型均有迁移脚本；幂等/可回滚
    assert:
      type: path
      target: backend/migrations/**
  - id: migration-rls
    desc: RLS 策略存在且启用
    assert:
      type: grep
      target: backend/migrations/**
      pattern:
        - "ALTER TABLE"
        - "USING (tenant_uuid = current_setting('app.tenant_uuid')::uuid)"
```
