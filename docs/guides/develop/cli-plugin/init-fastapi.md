# 使用 `px-plugin init` 初始化 FastAPI 插件

## 1) 目标

生成一个 **Python FastAPI 后端** + 管理端前端（Nuxt 或 Next）的插件项目。

## 2) 推荐命令

```bash
px-plugin init com.powerx.plugins.demo-fastapi --backend python-fastapi --admin nuxt --force
```

如需 React 技术栈管理端（Next）：

```bash
px-plugin init com.powerx.plugins.demo-fastapi --backend python-fastapi --admin next --force
```

## 3) 结果检查

初始化后应看到：

- `backend/`（FastAPI 代码）
- `web-admin/`（Nuxt）或 `web-admin/next/`（Next）
- `plugin.yaml`
- `make-files/` 与 `Makefile`

并且不应出现未选中的后端目录（例如你选了 `python-fastapi` 时，不应再生成 `backend/go-gin`）。

## 4) 首次验证

在项目根目录执行：

```bash
make manifest-align-fix
make manifest-align-check
```

然后按后端模式做测试：

```bash
make BACKEND=fastapi test-smoke
make BACKEND=fastapi test-regression
```

## 5) 常见误区

- `--admin react` 不是有效参数，React 路线对应 `--admin next`。
- `--backend` 不写时默认是 `go-gin`，要 FastAPI 必须显式传 `python-fastapi`。
