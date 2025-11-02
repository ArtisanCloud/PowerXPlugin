# Changelog

所有显著变更会记录在该文件中，并遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/) 与 [语义化版本号](https://semver.org/lang/zh-CN/) 规范。

## [Unreleased]

### Added
- 初始发布流程文档与 Release Workflow (`docs/release.md`, `.github/workflows/release.yml`)
- CLI 生成物示例 `examples/starter/`，对齐 Phase 6 用户故事
- Skeleton Templates CRUD 栈（后端内存仓储 + 前端页面 + `useTemplateApi` 示例）并同步至 CLI 脚手架模板
- `px-plugin init` 输出 Nuxt 项目新增 `lint` / `test:e2e` 占位脚本，便于后续扩展质量闸门

### Changed
- 脚手架模板 README 增加 Release 指引与多语言 TODO 提示
- Manifest Schema 追加菜单 `children` 递归定义，Skeleton/CLI 均可注册嵌套导航
- Quickstart / Standalone 指南补充多租户 CRUD 验证与延迟记录流程

### Deprecated
- 无

### Removed
- 无

### Fixed
- 无

### Security
- 无
