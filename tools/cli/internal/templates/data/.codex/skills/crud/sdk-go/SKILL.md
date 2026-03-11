---
name: crud-sdk-go
description: 基于内嵌规则 的 gRPC SDK 依赖规范。用于校验/补充 backend go.mod 依赖。
---

# CRUD gRPC SDK

## 步骤

1) 打开 `本文件内嵌规则`。
2) 在 `backend/go.mod` 确认依赖 `github.com/ArtisanCloud/PowerX/api/grpc/gen/go`。

## 规则（内嵌）

### sdk_go.yaml

```yaml
# PowerX gRPC SDK 依赖
kind: ruleset
id: crud_sdk_go
version: 1.0.0
checks:
  - id: sdk-module
    desc: go.mod 中必须依赖宿主提供的 gRPC SDK
    assert:
      type: grep
      target: backend/go.mod
      pattern:
        - "github.com/ArtisanCloud/PowerX/api/grpc/gen/go"
```
