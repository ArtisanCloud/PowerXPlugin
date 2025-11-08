# PowerX Plugin Template Registry

该目录维护 `px-plugin init` 使用的模板索引。`index.yaml` 为单一真相，描述每个模板的元数据、支持的语言、最小运行时、依赖与 Hook。CLI/文档同步脚本会读取本目录生成 docmap，禁止擅自移动文件。

## 文件结构

```
packages/template-registry/
├── README.md            # 本说明
└── index.yaml           # 模板清单（versioned schema）
```

## Schema 概览

| 字段 | 说明 |
|------|------|
| `version` | Registry schema 版本（当前 `v0.1.0`）。 |
| `lastSynced` | 最近一次 `npm run sync:templates` / `go run ./cmd/px-plugin init` 验证时间。 |
| `templates[]` | 模板数组，包含 `id`、栈、最低运行时、依赖锁定等元数据。 |
| `templates[].paths` | 指向 `scaffold/templates/**` 中 backend/frontend/manifest 等原始模板路径。 |
| `templates[].languages` | 语言与运行时要求，例如 `go: ">=1.24"`、`node: ">=18"`, `packageManager: npm`. |
| `templates[].dependencies` | 模板默认依赖锁定，包括 PowerX Framework、Layer、核心 npm 包。 |
| `templates[].hooks` | 渲染生命周期 Hook（`preRender`/`postRender`），脚本路径相对仓库根目录。 |
| `templates[].rollback` | 模板灰度/回滚策略提示，供 CLI 输出 next steps。 |

## 约束

1. `id` 必须唯一，推荐使用 `kebab-case`。
2. 更新模板文件后必须同步更新依赖版本/运行时要求。
3. 如果模板需要额外 Hook，请在 `scripts/hooks/` 下放置 Node 脚本，并在索引里引用。
4. CLI/文档需以此文件为准生成选项和使用说明，修改后请运行 `npm run sync:templates -- --check` 确认差异。
