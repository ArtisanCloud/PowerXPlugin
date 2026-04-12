---
name: nuxt-layout
description: 基于内嵌规则 的 Nuxt Layout 规范。用于侧边栏导航与 RBAC 可见性配置。
---

# CRUD Nuxt Layout

## 步骤

1) 打开 `本文件内嵌规则`。
2) 在 `web-admin/app/app.vue` 更新导航与菜单规则。

## 核对点

- 路由/权限与 RBAC 规范一致。
- Sidebar 规则符合说明。

## 规则（内嵌）

### nuxt_layout.yaml

```yaml
kind: ruleset
name: plugin/crud/frontend/nuxt_layout
version: 1

layout:
  nav:
    - label: "Templates"
      to: "/templates"
      icon: "i-lucide-file"
      permission: "base:template:read"    # UI 可见性关联 RBAC
  menu:
    - |
      Sidebar Rule（Standalone 模式 / 创意模块）
      - 一级菜单可折叠展示子菜单，仅控制展开状态，不触发跳转或高亮
      - 子页面激活时才标记选中
      - 子菜单需保持视觉缩进，整体结构与其他一级菜单平级
  files:
    - target: web-admin/app/app.vue
      template: builtin/nuxt_app_shell_with_nav

gates.require: [PG-FE-RBAC-001]
```
