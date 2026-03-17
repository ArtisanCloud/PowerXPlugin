---
name: crud-test
description: 基于内嵌规则 的测试规范。用于补齐合同测试、集成测试与迁移冒烟测试。
---

# CRUD Tests

## 步骤

1) 打开 `本文件内嵌规则`。
2) 在 `backend/test/**` 增加对应测试。

## 核对点

- HTTP/gRPC 合同测试齐备。
- 多租户隔离集成测试存在。
- 迁移冒烟测试存在。

## 规则（内嵌）

### test.yaml

```yaml
# 测试策略
kind: ruleset
id: crud_test
version: 1.0.0
checks:
  - id: contract-tests
    desc: HTTP/gRPC 合同测试齐备
    assert:
      type: path
      target: backend/test/contract/**
  - id: integration-rls
    desc: 多租户读写隔离的集成测试
    assert:
      type: path
      target: backend/test/integration/**
  - id: migration-smoke
    desc: 迁移冒烟测试存在
    assert:
      type: path
      target: backend/test/migration/**
```
