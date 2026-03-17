---
name: crud-handler-http
description: 基于内嵌规则 的 HTTP Handler 规范。用于新增/调整 HTTP Handler 与参数校验。
---

# CRUD HTTP Handler

## 步骤

1) 打开 `本文件内嵌规则`。
2) 在 `backend/internal/transport/http/**` 实现 handler。

## 核对点

- Handler 只调用 Service，不含 SQL/事务/复杂流程。
- 写入接口具备参数校验。

## 规则（内嵌）

### handler_http.yaml

```yaml
# HTTP Handler 规范
kind: ruleset
id: crud_handler_http
version: 1.0.0
checks:
  - id: handler-slim
    desc: Handler 仅调用 Service；禁止 SQL/事务/复杂流程
    assert:
      type: ban-symbol
      target: backend/internal/transport/http/**
      symbols:
        - "db.Query"
        - "BeginTx("
  - id: handler-validate
    desc: 所有写入接口具备参数校验
    assert:
      type: grep
      target: backend/internal/transport/http/**
      pattern:
        - validate
```
