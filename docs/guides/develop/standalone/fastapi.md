# Standalone 启动（Python FastAPI 后端）

本页仅覆盖 Python FastAPI 后端的 Skeleton 启动流程。

## 目录

- `skeleton/backend/python-fastapi`

## 快速启动（开发）

```bash
cd skeleton/backend/python-fastapi
python -m venv .venv
. .venv/bin/activate
python -m pip install -U pip
pip install -r requirements.txt
bash scripts/dev.sh
```

## 健康检查

```bash
curl http://127.0.0.1:8277/healthz
```

## 说明

- 目前 FastAPI 为最小空壳，仅提供 `/healthz` 示例。
- 端口与路由后续需要与插件协议保持一致。
