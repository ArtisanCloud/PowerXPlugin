# PowerXPlugin Release Playbook

本指南描述 PowerXPlugin 仓库的发布流程，涵盖 Go Module、Nuxt Layer、CLI 二进制以及脚手架示例的同步校验。

## 发布节奏

- **触发方式**：推送符合 `v*` 的 Git Tag 或手动执行 `Release` Workflow。
- **输出物**：
  - `github.com/ArtisanCloud/PowerXPlugin/framework/backend/go` Go Module Tag
  - `@artisan-cloud/plugin-framework-admin`、`@artisan-cloud/plugin-framework-client` npm 版本
  - `px-plugin` CLI 二进制（压缩包 + 校验文件）
  - `examples/starter/` 中的最新 CLI 生成物快照
  - `CHANGELOG.md` 中的对应版本条目

## 发布前检查清单

1. `go test ./...`（分别在 `framework/backend/go/` 与 `tools/cli/` 模块内执行）
2. `npm test && npm run lint && npm run build`
3. 验证 `docs/contracts/**` 与框架/SDK 校验结果一致
4. 手动运行一次 `px-plugin init com.powerx.demo` 并启动示例（后端 `go run`、前端 `npm run dev`）
5. 更新 `CHANGELOG.md` 与 `specs/**/tasks.md` 状态

## GitHub Actions Release Workflow

文件位置：`.github/workflows/release.yml`

关键步骤：

1. 安装 Go/Node 工具链与 npm workspace 依赖
2. 针对 `framework/backend/go/`、`tools/cli/` 分别执行 `go test ./...`
3. 构建 `px-plugin` 可执行文件并打包归档
4. 在 `examples/` 目录使用 CLI 生成校验项目（默认 `com.powerx.release`）
5. 调用 `px-plugin package/dist/publish` —— 当前为实验性占位，实现稳态后将替换为真实构建/上传逻辑
6. 上传 CLI 二进制产物，供后续发布或验证使用

> ⚠️ `MARKETPLACE_TOKEN` Secret 目前为占位符，待 Marketplace API 上线后再接入真实凭据。

## 手动发布步骤

1. 在主分支完成发布前检查清单
2. 根据版本语义更新：
   - `framework/backend/go/go.mod` 与相关引用的版本号
   - npm 包的 `package.json` `version` 字段
   - `CHANGELOG.md` 新增条目
3. 提交并合并变更，随后创建 `framework/backend/go/vX.Y.Z` Tag
4. 确认 Release Workflow 运行通过
5. 将构建好的 npm 包发布到 Registry（待自动化）
6. 通过 `go list -m github.com/ArtisanCloud/PowerXPlugin/framework/backend/go@vX.Y.Z` 验证模块可用

## 待办与后续迭代

- [ ] 将 CLI `package/dist/publish` 命令替换为真实构建、打包、上传逻辑
- [ ] 在 Release Workflow 中集成 npm 发布与 GitHub Release 草稿
- [ ] 为 CLI 二进制增加多平台交付（macOS/Linux/Windows）
- [ ] 补充回滚流程与紧急修复指引
