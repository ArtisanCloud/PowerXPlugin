# PowerX Framework 发布指南

本指南覆盖仓库内 **Go 框架模块** 与 **Nuxt 前端包** 的发布流程，帮助在同步更新后端与前端能力时保持版本一致。建议在每次面向外部交付前，依照本文检查并发布所需版本。所有命令默认以仓库根目录 `/path/to/PowerXPlugin` 为起点，若需切换目录会在步骤中显式说明。

---

## 目标受众与适用场景
- 需要对外发布新的 `github.com/ArtisanCloud/PowerXPlugin/framework/backend/go` Go 模块版本。
- 需要发布 `@artisan-cloud/plugin-framework-admin` / `@artisan-cloud/plugin-framework-client` 到 npm。
- 需要同时调整 CLI / Skeleton 模板中的版本号。
- 需要撤回、回滚或重新发布前后端框架版本。

---

## 发布前通用检查
1. **代码状态**：所有相关改动已合并到目标分支，工作树干净（`git status` 无未提交变更）。
2. **测试通过**：
   - Go 框架：`cd framework/backend/go && go test ./...`
   - Nuxt 框架：`cd framework/frontend/nuxt && npm run lint`（必要时进入各包执行 `npm run build`）
3. **依赖一致**：Skeleton、示例、CLI 模板中的版本占位符已准备好（见后文）。
4. **账号与权限**：
   - **Git**：具备仓库标签推送权限；如需删除 tag，也需相应权限。
   - **npm**：
     1. 本机已登录：`npm login` 或配置 `npm set //registry.npmjs.org/:_authToken=...`
     2. `npm whoami` 可正常输出用户名。
     3. 确认拥有 `@artisan-cloud` scope 发布权限：`npm org ls artisan-cloud` / `npm access list packages <user>`。
     4. 若没有权限，请在 npm 官网或通过组织管理员授予。

---

## 一、Go 框架模块发布

1. **确认版本号**
   - 依据语义化版本选择 `vX.Y.Z`（例如 `v0.0.1-alpha`）。
   - 更新 `CHANGELOG` 或发布记录。

2. **同步代码引用**
   - `framework/backend/go/go.mod` 已更新 module 声明（若必要）。
   - 其他模块中引用的新版本需对应更新或通过 replace 指向。

3. **打 Tag & 推送**
   ```bash
   cd /path/to/PowerXPlugin
   git tag framework/backend/go/v0.0.1-alpha
   git push origin framework/backend/go/v0.0.1-alpha
   ```
   > 若需重发同名版本，先执行 `git tag -d framework/backend/go/v0.0.1-alpha && git push origin :refs/tags/framework/backend/go/v0.0.1-alpha`，再创建新的 tag。

4. **验证可用性**
   ```bash
   # 任意目录
   go list -m github.com/ArtisanCloud/PowerXPlugin/framework/backend/go@v0.0.1-alpha
   ```
   确保 Go proxy 能获取到最新版本。

---

## 二、Nuxt 框架包发布（`framework-admin` / `framework-client`）

### 1. 准备版本号
```bash
cd /path/to/PowerXPlugin
export ADMIN_VERSION=0.0.1-alpha
export CLIENT_VERSION=0.0.1-alpha
```
根据语义化版本递增，例如 `0.0.1-alpha.1`、`0.1.0` 等。请确保与 CLI 模板默认值一致。

### 1.1 先查看远程当前版本（避免误判）

注意：`npm view <pkg> version` 默认返回 **dist-tag `latest`** 指向的版本，与你刚发布到 `--tag alpha` 的版本可能不同。

```bash
npm view @artisan-cloud/plugin-framework-admin version
npm dist-tag ls @artisan-cloud/plugin-framework-admin

npm view @artisan-cloud/plugin-framework-client version
npm dist-tag ls @artisan-cloud/plugin-framework-client
```

### 2. 更新仓库内引用
需同步调整的位置：
1. `framework/frontend/nuxt/framework-admin/package.json` → `version`
2. `framework/frontend/nuxt/framework-client/package.json` → `version`
3. `tools/cli/cmd/init.go` → `defaultAdminVersion` / `defaultClientVersion`
4. `skeleton/web-admin/nuxt/package.json` 与 `examples/**/web-admin/package.json`
5. 文档示例（如 `docs/guides/develop/cli-plugin-tutorial.md`、`README.md`）
6. 外部仓库或脚本中若有固定版本号，也需更新

更新后可执行：
```bash
cd framework/frontend/nuxt
npm run lint
# 可选（逐包执行）：npm run build
```

### 3. 发布到 npm
在两个包目录分别执行：
```bash
cd framework/frontend/nuxt/framework-admin
npm version $ADMIN_VERSION --no-git-tag-version
npm publish --access public --tag alpha

cd ../framework-client
npm version $CLIENT_VERSION --no-git-tag-version
npm publish --access public --tag alpha
```

> 建议使用 `--tag alpha` 避免覆盖 `latest`。若需正式发布，可改为 `--tag latest` 或后续使用 `npm dist-tag add`。

### 4. 验证结果
```bash
npm whoami
npm view @artisan-cloud/plugin-framework-admin version --json   # latest
npm view @artisan-cloud/plugin-framework-admin@alpha version --json
npm dist-tag ls @artisan-cloud/plugin-framework-admin

npm view @artisan-cloud/plugin-framework-client version --json  # latest
npm view @artisan-cloud/plugin-framework-client@alpha version --json
npm dist-tag ls @artisan-cloud/plugin-framework-client
```
确认返回版本正确。也可在临时目录执行：
```bash
npm install @artisan-cloud/plugin-framework-admin@$ADMIN_VERSION
```
确保能正常安装。

### 4.1 （可选）将 `alpha` 版本提升为 `latest`

仅当你确认该版本要作为默认安装版本（不再是预发布）时才执行：

```bash
npm dist-tag add @artisan-cloud/plugin-framework-admin@$ADMIN_VERSION latest
npm dist-tag add @artisan-cloud/plugin-framework-client@$CLIENT_VERSION latest
```

### 5. 同步脚手架
1. **刷新锁文件**（根目录）：
   ```bash
   npm install --workspaces --package-lock-only
   ```
   若已有生成的插件示例，也需在该目录运行 `npm install`。
2. **验证 CLI 输出**：
   ```bash
   cd tools/cli
   go install ./cmd/px-plugin
   cd /tmp
   px-plugin init com.powerx.sample
   cd com.powerx.sample/web-admin
   npm install
   ```
   确认 `package.json` 中依赖指向新版本且能够成功安装。
3. **CLI 发布（可选）**：若本次改动需要同步发版 `px-plugin`，请在 `tools/cli` 内更新版本并打 tag。

---

## 三、Go 与 Nuxt 的协同
- 若同时更新后端接口与前端 Layer，务必在同一 PR 内更新所有版本号，避免引用不一致。
- `docs/release.md` 可用来记录每次发布的版本组合。
- 考虑添加自动化流程（GitHub Actions）在打 tag 时发布 npm 包，以减少遗漏。

---

## 四、回滚策略
| 场景 | 处理建议 |
| --- | --- |
| Go 模块 tag 发布错误 | 删除 tag（若允许），重新打新版本，例如 `v0.0.1-alpha.1` |
| npm 发布错误（<24h 内） | `npm unpublish <pkg>@<version>`；否则发布新的补丁版本覆盖 |
| npm 标签指向错误 | `npm dist-tag rm <pkg> <tag>`，再重新 `npm dist-tag add` |
| CLI 模板引用旧版本 | 检查 `tools/cli/cmd/init.go`、Skeleton/示例 `package.json` 是否同步更新 |

---

## 五、常见问题与排查

- **`403 Forbidden`**：检查 npm 是否登录正确账号，确认具备 `@artisan-cloud` scope 发布权限。
- **npm 包缺少文件**：确保 `package.json` 的 `files` 字段或 `.npmignore` 设置正确。
- **安装仍引用 `file:` 路径**：确认环境变量 `POWERXPLUGIN_USE_LOCAL_FRONTEND` 未设置，或在 CLI 模板中改用版本号。
- **Go 客户端仍在使用旧代理缓存**：必要时 `go clean -modcache` 或在发布后提升版本号。

---

## 六、发布后工作
1. 更新内部公告或 `CHANGELOG`。
2. 通知相关项目拉取新版本。
3. 将本指南中使用的示例版本替换为下一轮待发布版本，避免误导。

按以上流程执行即可完成 PowerX 框架前后端组件的发布。若需进一步自动化（如 CI/CD 发布 npm 包或创建 GitHub Release），请在项目规划中补充。 
| 步骤 | 目录 | 操作示例 | 产出 |
| --- | --- | --- | --- |
| 1 | `framework/backend/go` | `go test ./...`、`git tag framework/backend/go/vX.Y.Z`、`git push origin framework/backend/go/vX.Y.Z` | Go 模块版本可用 |
| 2 | `framework/frontend/nuxt/framework-admin` | `npm version $ADMIN_VERSION --no-git-tag-version`、`npm publish --access public --tag alpha` | Admin Layer 已发布 |
| 3 | `framework/frontend/nuxt/framework-client` | 同上 | Client 包已发布 |
| 4 | `skeleton` / `examples` / `tools/cli` | 更新依赖版本、`npm install --workspaces --package-lock-only` | 脚手架与示例同步版本 |
| 5 | 任意 | `px-plugin init ...`、文档更新 | 验证新版本可用并记录 |
