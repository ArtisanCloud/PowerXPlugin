# PowerX Publish Hub · 本地安装（Local Install）指南

本指南梳理「直接向 PowerX 部署插件」的本地安装流程，覆盖三种常见产物：

1. 构建后的 `dist/` 目录（解压即可运行）。
2. 手工压缩包（`.zip/.tar.gz` 等）。
3. `px-plugin pack` 生成的 `.pxp` 元数据包。

> ⚠️ 与 `px-plugin dev --watch` 不同，本地安装依赖 PowerX 的 Admin API 将产物写入插件运行目录。它不会帮你热重载代码，而是一次性部署一个可运行版本。
>
> 🚧 当前 `px-plugin dist/pack` 仍会打印 “experimental” 提示。本文描述的是 Phase 14「Local install fast path」落地后的目标流程，便于提前评审。如果你使用现有 CLI，请按照“手动构建”小节执行。

---

## 适用场景

- 研发/测试团队需要在自建 PowerX 环境快速验证插件，而不希望走 Marketplace 提交流程。
- 运维需要在隔离网络中手动安装 `.pxp` 包（或 dist 目录）并立即启用。
- Marketplace 尚未接入，但 PowerX 的插件管理能力已经上线。

---

## 流程总览

- **方案 A（推荐）**：构建 dist → 将 `dist/<version>` 拷贝到 PowerX → 调 `POST /admin/plugins/install/local` 安装 → 验证。
- **方案 B（可选）**：在 dist 基础上额外生成 `.pxp` → 传输 `.pxp` → 在 PowerX 上解包成 dist 目录 → 再按方案 A 调用 Admin API。
- URL 模式（`/install/url`）和 `apiPrefix` 说明对两种方案都适用。

---

## 前置条件

| 项目 | 说明 |
| ---- | ---- |
| PowerX 版本 | 启用了 `backend/internal/transport/http/admin/plugin` 路由；`config/config.yaml` 里的 `server.apiPrefix` 默认为 `/api`，如改成 `/api/v1`，下面的示例 URL 也要同步。 |
| CLI & Toolchain | `px-plugin`（Go 1.24+ 编译的 CLI，且**内置模板需与 skeleton 同步**）、Node.js 18+/npm 9+、Go 1.24+、GNU Make。 |
| 权限 | 调用 Admin API 的 Token 需要 `platform_ops` / `plugin_admin` 权限，能访问 `/admin/plugins/**`。 |
| 服务器目录 | PowerX 需要能够读取你提供的 `src_dir`。通常把产物放在 `/srv/powerx/plugins/<id>/dist` 或 `/opt/powerx/uploads/<version>/`。 |

> ⚠️ 模板同步检查（避免 `npm install` 报 `MODULE_NOT_FOUND`）
>
> `px-plugin init` 生成的 Nuxt 管理端模板包含 `web-admin/package.json` 的 `postinstall: node ./scripts/postinstall-lightningcss.mjs`。
> 如果你使用的 `px-plugin` 二进制内置模板仍是旧版本（未包含 `web-admin/scripts/postinstall-lightningcss.mjs`），那么在新项目里执行 `npm install` 会直接报错：
>
> - `Error: Cannot find module './scripts/postinstall-lightningcss.mjs'`
>
> 这不是你的插件项目写错了，而是**CLI 内嵌模板与 skeleton 发生过一次不同步**导致的回归。解决方式是：确保你使用的 `px-plugin` 已包含该脚本的模板文件后再执行 `px-plugin init`。
>
> - 从 PowerXPlugin 源码编译：`go build -o ./bin/px-plugin ./tools/cli/cmd/px-plugin`（参考 `tools/cli/README.md`）
> - 或升级你机器上的 `px-plugin` 到包含该模板修复的版本，然后重新生成项目

环境变量示例：

```bash
export POWERX_ROOT=/private/var/www/html/ArtisanCloud/X/PowerX/Core/PowerX
export API_BASE="https://dev-api.powerx.local/api/v1"
export ADMIN_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

---

## （可选）从零创建并启动一个插件项目

如果你还没有插件项目，或需要从头创建一个“可按本文构建 dist 并 local install”的项目，可以按以下最小流程生成并验证模板可用：

```bash
# 1) 确认 px-plugin 可用（示例：从 PowerXPlugin 源码构建）
cd /path/to/PowerXPlugin/tools/cli
go build -o ./bin/px-plugin ./cmd/px-plugin
./bin/px-plugin --version

# 2) 生成插件项目
./bin/px-plugin init --force com.example.helloworld
cd com.example.helloworld

# 3) 启动后端（开发）
make dev

# 4) 启动 Web Admin（开发）
cd web-admin
npm install
npm run dev
```

完成以上步骤后，再继续本文的 “方案 A / 方案 B” 构建 `dist/` 并执行安装即可。

## 方案 A：dist 直装（推荐）

Local install 的核心是让 PowerX 读到一份可直接运行的 `dist/`：包含后端二进制、Nuxt 构建结果、`plugin.yaml`、`publish.yml`、`manifest.json` 等。**这一部分是所有流程都必须完成的基础**，可以用 Makefile 或手动方式完成。

### 步骤 1：准备依赖

在执行 `make dist` 之前，请先在插件项目根目录安装好后端与前端依赖，否则 Nuxt 构建阶段会尝试下载临时包，导致版本不一致或出现 “Cannot find module 'nuxt/config'” 之类的错误。

```bash
cd /path/to/your-plugin        # 例如 plugins/com.powerx.helloworld

# Go 依赖
# 若项目未启用 go.work，可忽略 go work sync 的提示
go work sync 2>/dev/null || true
cd backend
go mod tidy
cd ..

# Web Admin 依赖
cd web-admin
npm install
cd ..
```

若之前在 web-admin 目录执行过 `npm install` 但后来清理了 `node_modules/`，请重新安装，确保 `web-admin/node_modules/.bin/nuxi` 与 `nuxt/config` 均来自模板声明的 Nuxt 4.2.x。

### 步骤 2：使用脚手架 Makefile 构建 dist

Phase 14 合并后，`px-plugin init` 会自动把 `skeleton/Makefile` 与 `make-files/` 带入新项目。完成上一步依赖安装后，即可直接运行；若需要覆盖默认版本号，可一并在命令前设置 `VERSION=`：

```bash
# 构建后端 + 前端，输出到 dist/
make dist

# 构建特定版本
VERSION=0.2.0 make dist
```

如需额外生成 `.pxp`，请跳转到「方案 B · 步骤 2」使用 `make pack`。

`make dist` 默认依次执行 `go build`, `npm install`, `npm run build` 并将产物归档到 `dist/`。若你的项目尚未集成 Makefile，可先运行下方“手动构建”步骤再回到这里。

### 步骤 3：手动构建（可选）

如果 CLI/Makefile 尚未完善，可直接在插件项目手工构建：

```bash
# 1) 编译后端
pushd backend
go mod tidy
GOOS=linux GOARCH=amd64 go build -o ../dist/backend/bin/plugin ./cmd/plugin
popd

# 2) 构建前端
pushd web-admin
npm install
npm run build
popd

# 3) 整理 dist 目录（包含 plugin.yaml、publish.yml、manifest.json 等）
mkdir -p dist/web-admin dist/backend
cp -R backend/dist/* dist/backend/
cp -R web-admin/.output/public dist/web-admin/
cp plugin.yaml dist/
cp publish.yml dist/
```

完成后即可进入“传输与安装”步骤。

---

## 方案 B：`.pxp` + dist 解包（可选）

如果你需要在 PowerX Marketplace 或离线审核链路中保留完整的签名/完整性信息，可以在完成 dist 之后再执行这一路径。`.pxp` **不会取代** dist，只是附加的校验材料；如果只想快速安装，可跳回方案 A。

### 步骤 1：使用 CLI 生成 `.pxp`（规划中）

Go 版本 CLI 已提供 `px-plugin dist`/`px-plugin pack` 子命令，但目前仍是 “experimental” 占位（见 `tools/cli/cmd/dist.go`, `package.go`）。Phase 14 将把 TypeScript 的打包逻辑迁移到 Go CLI，届时命令形态如下：

```
px-plugin dist \
  --manifest ./dist/manifest.json \
  --artefact ./dist/backend \
  --artefact ./dist/web-admin \
  --output-dir ./artifacts \
  --marketplace-public-key ./certs/marketplace.pem \
  --key-id marketplace-dev

px-plugin pack --manifest ./dist/manifest.json --artefact ./dist --output-dir ./artifacts
```

生成结果包含：
- `${pluginId}-${version}.pxp`
- `integrity.txt`
- `report.json`
- `manifest.signature`

### 步骤 2：使用 Makefile 生成 `.pxp`

当 Phase 14 的 Makefile 集成完成后，可直接运行：

```bash
make pack KEY_ID=marketplace-dev PUBLIC_KEY=./keys/marketplace.pem
```

脚本内部会调用 `px-plugin pack` 并把 `.pxp` 与配套校验文件放到 `dist/artifacts/`。运行该命令不会动 `dist/<version>/` 里的运行时代码，典型输出包括：

- `dist/artifacts/<pluginId>-<version>.pxp`
- `dist/artifacts/integrity.txt`
- `dist/artifacts/manifest.signature`
- `dist/artifacts/report.json`

> `.pxp` 只是带签名/校验信息的元数据包，不要把它塞进 `dist/<version>` 的运行目录；PowerX 仍然直接读取 `dist/<version>/backend`、`dist/<version>/web-admin` 等子目录。

#### 准备 `./keys/marketplace.pem`

`PUBLIC_KEY` 应指向一份 PEM（`-----BEGIN PUBLIC KEY-----` 或证书）文件，`px-plugin pack` 会使用它封装 `.pxp` 内部的对称密钥。通常把该文件放在仓库根目录 `keys/` 下，并通过 `.gitignore` 排除。

1. **使用官方 Marketplace 公钥（推荐）**
   ```bash
   mkdir -p keys
   cp ~/Downloads/marketplace-dev.pem ./keys/marketplace.pem
   ```
   拿到的是 `.crt` 时，可执行 `openssl x509 -pubkey -noout -in marketplace.crt > ./keys/marketplace.pem` 提取纯公钥。

2. **仅供本地验证的自签名公钥**
   ```bash
   mkdir -p keys
   openssl genrsa -out ./keys/marketplace-dev.key 2048
   openssl req -new -x509 -key ./keys/marketplace-dev.key \
     -subj "/CN=marketplace-dev" -days 365 \
     -out ./keys/marketplace-dev.crt
   openssl x509 -pubkey -noout -in ./keys/marketplace-dev.crt \
     > ./keys/marketplace.pem
   ```
   请勿把测试密钥用于生产；生产环境必须使用官方下发的公钥或受信任 CA 证书。

若本地环境尚未升级，可暂时跳过 `.pxp` 步骤，只保留 dist 即可完成 local install。

> 当前实现中 `.pxp` 仍是 JSON 元数据，不包含真实文件。无论是否生成 `.pxp`，PowerX 都必须能读到 dist 目录或已解包的文件。

---

### 步骤 4：传输 dist 目录（可选）

如果 PowerX Admin API 与打包机部署在同一节点，可直接在本地路径上执行安装，跳过本步骤。只有在 PowerX 运行于远程服务器、需要把 dist 搬过去时再执行以下操作。

dist 输出位于 `dist/<version>/...`，可将整个版本目录同步到 PowerX 服务器，例如：

```bash
DIST_VERSION=0.1.0
rsync -avz dist/$DIST_VERSION \
  powerx-admin:/srv/powerx/plugins/com.powerx.helloworld/dist/$DIST_VERSION
```

也可以先压缩再上传：

```bash
cd dist && zip -r ../com.powerx.helloworld-$DIST_VERSION.zip $DIST_VERSION
aws s3 cp ../com.powerx.helloworld-$DIST_VERSION.zip s3://internal-artifacts/
```

到服务器后解压至目标目录（例如 `/srv/powerx/plugins/com.powerx.helloworld/dist/0.1.0`）。

### 步骤 5：调用 `/admin/plugins/install/local`

不论 dist 在本机还是远程服务器，最终都需要调用 Admin API 告诉 PowerX 去读取那个目录。

- **本地环境**：`src_dir` 填写绝对路径（例如 `$(pwd)/dist/0.1.0`）。
- **远程环境**：`src_dir` 填远程服务器上的同步路径（配合上方步骤 4）。

本地示例：

```bash
DIST_VERSION=0.1.0
curl -X POST "$API_BASE/admin/plugins/install/local" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
        \"src_dir\": \"$(pwd)/dist/$DIST_VERSION\",
        \"enable\": true,
        \"force\": false
      }"
```

远程示例：

```bash
curl -X POST "$API_BASE/admin/plugins/install/local" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
        "src_dir": "/srv/powerx/plugins/com.powerx.helloworld/dist/0.1.0",
        "enable": true,
        "force": false
      }'
```

- `src_dir`：PowerX 服务器可读的绝对路径。应包含 `backend/`, `web-admin/`, `plugin.yaml`, `publish.yml` 等文件。
- `enable`：安装完成后是否立即启用并切换成当前版本。
- `force`：若已有相同版本，是否强制覆盖。

成功响应示例：

```json
{
  "installed": {
    "id": "com.powerx.helloworld",
    "version": "0.1.0",
    "state": "installed"
  }
}
```

### 步骤 6：URL 模式（可选）

用于 zip/pxp 已上传到可访问的 HTTP/S 位置：

```bash
curl -X POST "$API_BASE/admin/plugins/install/url" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
        "url": "https://upload.local/com.powerx.helloworld-0.1.0.zip",
        "sha256": "4f83c4a8...",
        "enable": false
      }'
```

- `sha256` 可选，提供时 PowerX 会校验包体完整性。
- `.pxp` 目前仍需在服务器侧解包后才能运行；此模式主要方便上传存档，安装逻辑仍复用 `InstallFromFile`。

### 步骤 7：关于 `apiPrefix`

`backend/etc/config.yaml`（或相应环境的配置）中的 `server.apiPrefix` 决定了最终 URL：

- 默认：`server.apiPrefix: /api`，API 地址为 `https://<host>/api/admin/plugins/install/local`。
- 如果设为 `/api/v1`，需要改成 `https://<host>/api/v1/admin/plugins/install/local`。`px-plugin dev --watch` 的 Dev API 也需保持一致前缀。

---

### 步骤 3：传输 `.pxp`

```bash
aws s3 cp dist/artifacts/com.powerx.helloworld-0.1.0.pxp s3://internal-artifacts/
# 或
scp dist/artifacts/com.powerx.helloworld-0.1.0.pxp powerx-admin:/opt/powerx/uploads/
```

### 步骤 4：在 PowerX 解包并调用 local install

```bash
make local-install-pxp \
  PACKAGE=./dist/artifacts/com.powerx.helloworld-0.1.0.pxp \
  API_BASE=https://dev-api.powerx.local/api/v1 \
  TOKEN=eyJhbGciOi...
```

或手工：

```bash
mkdir -p /tmp/pxp && unzip com.powerx.helloworld-0.1.0.pxp -d /tmp/pxp
curl -X POST "$API_BASE/admin/plugins/install/local" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
        "src_dir": "/tmp/pxp/com.powerx.helloworld-0.1.0",
        "enable": false,
        "force": false
      }'
```

> 解包后若目录结构不同，请确保 `src_dir` 指向含有 `plugin.yaml`/`backend`/`web-admin` 的根，随后重复方案 A 的验证/启用步骤。

---

## 安装后的验证

```bash
# 查看插件列表
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  "$API_BASE/admin/plugins" | jq

# 查看单个插件状态
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  "$API_BASE/admin/plugins/com.powerx.helloworld/status" | jq

# 启动/停止
curl -X POST -H "Authorization: Bearer $ADMIN_TOKEN" \
  "$API_BASE/admin/plugins/com.powerx.helloworld/enable"

# 查看日志
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  "$API_BASE/admin/plugins/com.powerx.helloworld/logs?tail=200"
```

> 若需要切换版本，可使用 `POST /admin/plugins/:id/switch_version`，请求体 `{"version": "0.1.0", "enable": true}`。

---

## 6. 常见问题 / 故障排查

| 问题 | 说明与解决办法 |
| ---- | --------------- |
| `install/local` 返回 404 | 检查 `server.apiPrefix` 是否与请求一致；确认 Admin API 路由已注册。 |
| `src_dir` 找不到 | 目标目录必须存在于 PowerX 服务器本地磁盘；如果你在开发机上执行 curl，需要通过 `ssh` 或 CI 让命令运行在服务器上。 |
| 版本已存在 | 若只是想覆盖当前版本，传 `force=true`；否则建议 bump `plugin.yaml` 里的 `version`。 |
| `.pxp` 无法直接运行 | 目前 `.pxp` 只包含 manifest/integrity/audit 信息，仍需解包后把 dist 目录传给 `install/local`。后续计划由 PowerX 直接解析 `.pxp`。 |
| 与 `px-plugin dev --watch` 混淆 | `dev --watch` 通过 Dev API 注册临时会话，关闭终端即销毁。Local install 则写入正式的插件版本，需显式卸载或切换版本。 |

---

## 7. 下一步

- 完成本地安装后，建议回到《[离线发布指南](./offline.md)》了解 `.pxp` 在 Marketplace 审核链路中的用法。
- 如需自动化，可参考即将上线的 `make local-install`、`make local-install-pxp` 目标，把上述 curl 调用写入 CI/CD。完成后请在 `docs/guides/publish/online.md` 记录你的实践经验。 
