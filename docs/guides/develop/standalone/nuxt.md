# Standalone 启动（Nuxt 前端）

本页仅覆盖 Nuxt Web Admin 的 Skeleton 启动流程。

## 目录

- `skeleton/web-admin/nuxt`

## 快速启动

```bash
cd skeleton/web-admin/nuxt
npm install
npm run dev
```

## 说明

- 默认端口为 `3131`，冲突时自动找空闲端口。
- 如需宿主模式（`/_p/<plugin-id>/admin`），请在启动前设置 `POWERX_PROXY=1`。
