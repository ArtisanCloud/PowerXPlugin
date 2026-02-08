# Standalone 启动（Go Gin 后端）

本页仅覆盖 Go Gin 后端的 Skeleton 启动流程。

## 目录

- `skeleton/backend/go-gin`

## 快速启动

```bash
# 1. 复制示例配置
cp skeleton/backend/go-gin/etc/config.example.yaml skeleton/backend/go-gin/etc/config.yaml

# 2. 复制统一环境变量配置
cp skeleton/backend/.env.example skeleton/backend/.env

# 3. 初始化数据库（migrate + seed）
cd skeleton/backend/go-gin
go run ./cmd/database/main.go setup

# 4. 启动后端
go run ./cmd/plugin
```

## 健康检查

```bash
curl http://127.0.0.1:8078/healthz
```

## 说明

- 后端会自动读取 `skeleton/backend/.env`；请在该文件中设置 `POWERX_PROXY/POWERX_RBAC_DELEGATE/PLUGIN_IAM_*` 等变量。
- Standalone 运行模式应保持 `POWERX_PROXY=0`，否则会被视作宿主模式。
- 如果需要仅执行迁移或种子，可改用 `migrate` / `seed`。
