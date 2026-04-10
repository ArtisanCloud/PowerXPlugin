# PowerX Framework 发布指南（Go + npm）

本文用于发布以下两个交付面：

- Go 模块：`github.com/ArtisanCloud/PowerXPlugin/framework/backend/go`
- npm 包：`@artisan-cloud/plugin-framework-admin`、`@artisan-cloud/plugin-framework-client`

> 命令默认在仓库根目录执行：`/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin`
>
> 本文只负责 framework 发布与版本同步。若你只是构建/安装 `px-plugin` 并初始化项目，请使用：
> `docs/guides/develop/cli-plugin/cli-plugin-tutorial.md`

## 1. 发布前检查

1. 工作区干净：`git status`
2. Go 框架测试通过：
   ```bash
   cd framework/backend/go && go test ./...
   ```
3. 前端框架包做发布前打包校验（当前两个包没有 `build` 脚本）：
   ```bash
   cd framework/frontend/nuxt/framework-admin && npm pack --dry-run
   cd ../framework-client && npm pack --dry-run
   ```
4. npm 权限可用：
   ```bash
   # 以下 3 条在任意目录都可执行（查询 npm 远端，不依赖本地包路径）
   npm whoami
   npm dist-tag ls @artisan-cloud/plugin-framework-admin
   npm dist-tag ls @artisan-cloud/plugin-framework-client
   ```
   说明：
   - `npm whoami`：确认当前登录账号（防止发到错误账号）。
   - `npm dist-tag ls <pkg>`：查看远端 `latest/alpha` 指向版本，避免误覆盖。

## 2. 发布 Go 模块

1. 选择版本并设置变量（示例）：
   ```bash
   export FRAMEWORK_GO_VERSION=v0.0.1-alpha
   export FRAMEWORK_GO_TAG=framework/backend/go/${FRAMEWORK_GO_VERSION}
   ```
2. 打 tag 并推送（在**仓库根目录**执行）：
   ```bash
   cd /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin
   git tag "${FRAMEWORK_GO_TAG}"
   git push origin "${FRAMEWORK_GO_TAG}"
   ```
3. 验证 Go proxy 可见（在**任意目录**执行）：
   ```bash
   go list -m github.com/ArtisanCloud/PowerXPlugin/framework/backend/go@${FRAMEWORK_GO_VERSION}
   ```
4. 一键对齐仓库引用（推荐）：
   ```bash
   # 仅修改文件（不打 tag）
   bash scripts/release/bump-framework-go-version.sh "${FRAMEWORK_GO_VERSION}"

   # 修改文件 + 打 tag + 推送
   bash scripts/release/bump-framework-go-version.sh "${FRAMEWORK_GO_VERSION}" --tag --push
   ```

## 3. 发布 npm 包

1. 准备版本号：
   ```bash
   export ADMIN_VERSION=0.0.3-alpha
   export CLIENT_VERSION=0.0.3-alpha
   ```
2. 发布 `framework-admin`：
   ```bash
   cd framework/frontend/nuxt/framework-admin
   npm version $ADMIN_VERSION --no-git-tag-version
   npm publish --access public --tag alpha
   ```
3. 发布 `framework-client`：
   ```bash
   cd ../framework-client
   npm version $CLIENT_VERSION --no-git-tag-version
   npm publish --access public --tag alpha
   ```
4. 验证 dist-tag：
   ```bash
   # 在任意目录执行即可
   npm dist-tag ls @artisan-cloud/plugin-framework-admin
   npm dist-tag ls @artisan-cloud/plugin-framework-client
   ```

## 4. 同步仓库引用（必须）

至少同步这些位置：

1. `framework/frontend/nuxt/framework-admin/package.json`（`version`）
2. `framework/frontend/nuxt/framework-client/package.json`（`version`）
3. `tools/cli/cmd/init.go`（`defaultAdminVersion` / `defaultClientVersion`）
4. `skeleton/web-admin/nuxt/package.json`（依赖版本）
5. 文档示例中的版本号（如 `README.md`、相关 guide）

同步后执行模板检查：

```bash
npm run sync:templates -- --check
```

## 5. 验证脚手架输出

```bash
cd tools/cli
go install ./cmd/px-plugin
cd /tmp
px-plugin init com.powerx.sample
cd com.powerx.sample/web-admin/nuxt
npm install
```

验证点：

- 新项目 `web-admin/nuxt/package.json` 依赖指向新版本
- 可正常安装，不出现缺包或 `file:` 误引用

## 6. 回滚策略

| 场景 | 处理 |
| --- | --- |
| Go tag 错误 | 删除 tag 后发布新版本（建议递增补丁后缀） |
| npm 版本错误 | 发布新补丁版本修复；慎用 `unpublish` |
| dist-tag 指向错误 | `npm dist-tag rm/add` 调整 |
| CLI 仍生成旧版本 | 检查 `tools/cli/cmd/init.go` 与模板同步 |

## 7. 备注（当前仓库结构）

1. 开发态清单在 `skeleton/plugin.yaml`，不要再用根目录旧路径。
2. 插件前端默认目录是 `web-admin/nuxt`，不是 `web-admin` 根目录。
3. 任何发布后回归命令，涉及 manifest 时优先用 `skeleton/plugin.yaml`。
