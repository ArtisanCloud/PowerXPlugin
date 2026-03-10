# 插件安装前必检清单（Preflight）

适用范围：所有基于 PowerX / PowerXPlugin 生态发布到宿主环境的插件。  
目标：避免出现“`install/local` 成功但 `switch_version` 失败”的灰色状态。

---

## 1) 单一事实源（必须）

- 能力清单的事实源固定为：`contracts/capabilities/*.yaml`
- `plugin.d/{capabilities,exposure,rbac}.yaml` 必须由同步脚本生成，不建议手改
- `plugin.yaml` 顶层 `capabilities` 不应与 `catalogs.capabilities` 并存（避免 merge 冲突）

---

## 2) `plugin.d` 角色（必须理解）

- `plugin.d` 是运行时目录清单（网关放行、路由暴露、RBAC 映射均依赖它）
- 业务改动后若未同步 `plugin.d`，最常见结果是 `403 no permission rule`

---

## 3) 安装成功 ≠ 启用成功（必须区分）

- `POST /admin/plugins/install/local` 返回 `200` 仅表示包写入成功
- `switch_version` 阶段还会触发：
  - 进程启动
  - manifest 注册
  - 健康检查
- 任一失败都会导致最终启用失败（常见表现：健康检查超时）

---

## 4) 运行时 `manifestx` 约束（高频漏项）

- `manifestx.Plugin().Permissions` 必须使用三段式：`domain.resource.action`
- 仅允许字符：`[a-z0-9.-]`
- 示例：`ai-craft.template.read`
- 反例：`template.read`（两段式，进程启动阶段可能直接 Fatal）

---

## 5) 打包完整性（必须）

安装包中必须包含并且路径可解析：

- `plugin.yaml`
- `plugin.d/capabilities.yaml`
- `plugin.d/exposure.yaml`
- `plugin.d/rbac.yaml`
- `contracts/*` 中被 capability/schema 引用到的文件
- `migrations` 对应可执行入口（若声明了迁移）

---

## 6) 本地校验与宿主校验一致性（必须）

- 以可执行规则为准，不以“文档看起来正确”为准
- 建议固定流程：

```bash
make manifest-align-fix
make manifest-align-check
make dist
make skeleton-reinstall VERSION=<new_version> API_BASE=<api_base> TOKEN=<token>
```

---

## 7) 快速故障定位

- `400`（安装阶段）：
  - 先查 `plugin.yaml` / `catalogs` 结构冲突与缺失文件
- `403`（网关阶段）：
  - 先查 `plugin.d/exposure.yaml` 与 `plugin.d/rbac.yaml`
- `500`（运行阶段）：
  - 先查插件进程日志、数据库 schema、迁移执行情况、`manifestx` 字段合法性

