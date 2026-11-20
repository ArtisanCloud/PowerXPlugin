---
name: crud-dto
description: 基于内嵌规则 的 CRUD DTO 规范。用于新增/维护 Create/Update/List DTO 与校验标签。
---

# CRUD DTO

## 步骤

1) 打开 `本文件内嵌规则`。
2) 在 `backend/internal/dto/**` 创建/更新 DTO。

## 核对点

- Create/Update/List DTO 齐备，字段与模型对齐（必要字段除外）。
- 字段带 `binding/validate` 标签。

## 规则（内嵌）

### dto.yaml

```yaml
# DTO 与校验
kind: ruleset
id: crud_dto
version: 1.0.0
checks:
  - id: dto-exist
    desc: Create/Update/List DTO 均存在且字段与 Model 对齐（必要字段除外）
    assert:
      type: path
      target: backend/internal/dto/**
  - id: dto-validate-tags
    desc: DTO 字段有校验标签（binding/validate）
    assert:
      type: grep
      target: backend/internal/dto/**
      pattern:
        - validate:
```
