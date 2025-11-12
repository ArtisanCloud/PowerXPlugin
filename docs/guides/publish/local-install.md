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

1. **准备产物**：先完成「路径 A」生成 dist（必选），如需审计再执行「路径 B」生成可选的 `.pxp`。
2. **传输到 PowerX**：将 `dist/` 或 `.pxp` 拷贝到 PowerX 服务器可访问的路径，或上传到内网对象存储/HTTP 服务。
3. **调用 Admin API**：
   - 目录模式：`POST {apiPrefix}/admin/plugins/install/local`，参数为服务器上的 `src_dir`。
   - URL 模式：`POST {apiPrefix}/admin/plugins/install/url`，参数为可下载的包地址。
4. **启用与验证**：通过 `POST /admin/plugins/:id/enable`、`GET /admin/plugins/:id/status`、`GET /admin/plugins/:id/logs` 确认状态。

---

## 前置条件

| 项目 | 说明 |
| ---- | ---- |
| PowerX 版本 | 启用了 `backend/internal/transport/http/admin/plugin` 路由；`config/config.yaml` 里的 `server.apiPrefix` 默认为 `/api`，如改成 `/api/v1`，下面的示例 URL 也要同步。 |
| CLI & Toolchain | `px-plugin`（Go 1.24+ 编译的 CLI）、Node.js 18+/npm 9+、Go 1.24+、GNU Make。 |
| 权限 | 调用 Admin API 的 Token 需要 `platform_ops` / `plugin_admin` 权限，能访问 `/admin/plugins/**`。 |
| 服务器目录 | PowerX 需要能够读取你提供的 `src_dir`。通常把产物放在 `/srv/powerx/plugins/<id>/dist` 或 `/opt/powerx/uploads/<version>/`。 |

环境变量示例：

```bash
export POWERX_ROOT=/private/var/www/html/ArtisanCloud/X/PowerX/Core/PowerX
export API_BASE="https://dev-api.powerx.local/api/v1"
export ADMIN_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

---

## 1. 路径 A：构建可运行的 dist（必选）

Local install 的核心是让 PowerX 读到一份可直接运行的 `dist/`：包含后端二进制、Nuxt 构建结果、`plugin.yaml`、`publish.yml`、`manifest.json` 等。**这一部分是所有流程都必须完成的基础**，可以用 Makefile 或手动方式完成。

### 1.1 使用脚手架 Makefile（推荐）

Phase 14 合并后，`px-plugin init` 会自动把 `skeleton/Makefile` 与 `make-files/` 带入新项目。若当前版本尚未同步，可参考 `skeleton/` 或 `com.powerx.plugin.base` 手动复制。一旦具备 Makefile，可直接执行：

```bash
# 构建后端 + 前端，输出到 dist/
make dist

# 基于 dist/ 生成 .pxp 元数据（调用 px-plugin pack）
make pack KEY_ID=marketplace-dev PUBLIC_KEY=./keys/marketplace.pem
```

> `make dist` 默认依次执行 `go build`, `npm install`, `npm run build` 并将产物归档到 `dist/`。若你的项目尚未集成 Makefile，可先运行下方“手动构建”步骤再回到这里。

### 1.2 手动构建（当前可用方案）

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

## 2. 路径 B：生成 `.pxp` 元数据包（可选）

如果你需要在 PowerX Marketplace 或离线审核链路中保留完整的签名/完整性信息，可以在完成 dist 之后再执行这一路径。`.pxp` **不会取代** dist，只是附加的校验材料；不需要这类审计时可直接跳到后续步骤。

### 2.1 CLI（规划中）

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

### 2.2 使用 Makefile

当 Phase 14 的 Makefile 集成完成后，可直接运行：

```bash
make pack KEY_ID=marketplace-dev PUBLIC_KEY=./keys/marketplace.pem
```

脚本内部会调用 `px-plugin pack` 并把 `.pxp` 与配套校验文件放到 `dist/artifacts/`。若本地环境尚未升级，可暂时跳过 `.pxp` 步骤，只保留 dist 即可完成 local install。

> 当前实现中 `.pxp` 仍是 JSON 元数据，不包含真实文件。无论是否生成 `.pxp`，PowerX 都必须能读到 dist 目录或已解包的文件。

---

## 3. 将产物同步到 PowerX

根据部署环境选择其一：

### 3.1 拷贝 dist 目录

```bash
# 示例：拷贝到 PowerX 服务器
rsync -avz dist/ powerx-admin:/srv/powerx/plugins/com.powerx.helloworld/dist
```

### 3.2 上传压缩包或 .pxp

```bash
# 打包 dist
cd dist && zip -r ../com.powerx.helloworld-0.1.0.zip .

# 上传到对象存储/内网 HTTP
aws s3 cp ../com.powerx.helloworld-0.1.0.zip s3://internal-artifacts/
# 或
curl -F file=@../com.powerx.helloworld-0.1.0.zip https://upload.local/artifacts
```

> 如果只得到 `.pxp`，需要在 PowerX 服务器上解包（`mkdir /tmp/pxp && tar -xf xxx.pxp -C /tmp/pxp` 或 `px-plugin artefact inspect`）后再执行 local install。

---

## 4. 执行 Admin API 安装

### 4.1 目录模式：`/admin/plugins/install/local`

```bash
curl -X POST "$API_BASE/admin/plugins/install/local" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
        "src_dir": "/srv/powerx/plugins/com.powerx.helloworld/dist",
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

### 4.2 URL 模式：`/admin/plugins/install/url`

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

### 4.3 关于 `apiPrefix`

`backend/etc/config.yaml`（或相应环境的配置）中的 `server.apiPrefix` 决定了最终 URL：

- 默认：`server.apiPrefix: /api`，API 地址为 `https://<host>/api/admin/plugins/install/local`。
- 如果设为 `/api/v1`，需要改成 `https://<host>/api/v1/admin/plugins/install/local`。`px-plugin dev --watch` 的 Dev API 也需保持一致前缀。

---

## 5. 安装后的验证

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
