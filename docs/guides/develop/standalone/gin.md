# Standalone 启动（Go Gin 后端）

本页仅覆盖 Go Gin 后端的 Skeleton 启动流程。

## 目录

- `skeleton/backend/go-gin`

## 快速启动

```bash
# 1. 复制示例配置
cp skeleton/backend/go-gin/etc/config.example.yaml skeleton/backend/go-gin/etc/config.yaml

# 2. 初始化数据库（migrate + seed）
cd skeleton/backend/go-gin
export POWERX_PROXY=0
export POWERX_RBAC_DELEGATE=false
export PLUGIN_IAM_TENANT_KEY=00000000-0000-0000-0000-000000000001
export PLUGIN_IAM_TENANT_NAME="Local Tenant"
export PLUGIN_IAM_ADMIN_EMAIL=admin@local.test
export PLUGIN_IAM_ADMIN_PASSWORD='S3cret!!'
go run ./cmd/database/main.go setup

# 3. 启动后端
go run ./cmd/plugin
```

## 健康检查

```bash
curl http://127.0.0.1:8078/healthz
```

## 说明

- 运行模式请保持 `POWERX_PROXY=0`，否则会被视作宿主模式。
- 如果需要仅执行迁移或种子，可改用 `migrate` / `seed`。
