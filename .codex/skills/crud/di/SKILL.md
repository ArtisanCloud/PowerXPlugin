---
name: crud-di
description: 基于内嵌规则 的依赖注入规范。用于调整容器构造与依赖注入链路。
---

# CRUD DI

## 步骤

1) 打开 `本文件内嵌规则`。
2) 在 `backend/internal/di/**` 确保容器注入配置、日志、客户端、Repo、Service、Transport。

## 规则（内嵌）

### di.yaml

```yaml
# 依赖注入（DI）与构建
kind: ruleset
id: crud_di
version: 1.0.0
checks:
  - id: container
    desc: 存在容器构造，注入配置/日志/客户端/Repo/Service/Transport
    assert:
      type: path
      target: backend/internal/di/**
```
