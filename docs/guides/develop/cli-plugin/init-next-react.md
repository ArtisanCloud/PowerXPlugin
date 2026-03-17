# 使用 `px-plugin init` 初始化 Next（React）管理端插件

## 1) 说明

在当前 CLI 参数中，React 管理端对应 `next`，即使用 `--admin next`。

## 2) 常用组合

### Go Gin + Next（React）

```bash
px-plugin init com.powerx.plugins.demo-next --backend go-gin --admin next --force
```

### Python FastAPI + Next（React）

```bash
px-plugin init com.powerx.plugins.demo-next --backend python-fastapi --admin next --force
```

## 3) 初始化后检查

应包含：

- `web-admin/next/`（React/Next 代码）
- `backend/`（按你选择的后端）
- `plugin.yaml`

不应再包含未选中的前端目录（例如选了 `next`，不应再落 `web-admin/nuxt`）。

## 4) 验证命令

```bash
make manifest-align-fix
make manifest-align-check
```

Go 后端：

```bash
make test-smoke
make test-regression
```

FastAPI 后端：

```bash
make BACKEND=fastapi test-smoke
make BACKEND=fastapi test-regression
```

## 5) 补充

完整端到端流程（含安装到 PowerX）请看：

- `docs/guides/develop/cli-plugin/cli-plugin-tutorial.md`
- `docs/guides/publish/local-install.md`
