# 构建与打包指引

本文针对 `make-files/build.mk` 中的目标进行简要说明，帮助团队快速完成插件的构建、打包与发布。默认假设命令在项目根目录执行。

## 环境准备
- Go 1.21+（根据 `go.mod` 要求调整）。
- Node.js 18+/npm 9+（用于 Nuxt 4 前端构建）。
- `golangci-lint`（可通过 `make dev-setup` 自动安装到 `GOPATH/bin`）。

常用初始化命令：
```bash
make dev-setup
```

## 核心变量
所有变量均可在执行 `make` 时通过命令行覆盖，例如：
```bash
BUILD_DIR=backend/out VERSION=0.1.1 make build
```
主要变量说明：

| 变量名 | 默认值 | 作用 |
| --- | --- | --- |
| `VERSION` | `0.1.0` | 当前插件版本号，影响产物目录与包名 |
| `PLATFORM` | `host` | `dist` 后端目标平台：`host`（本机）或 `linux` |
| `TARGET_ARCH` | `amd64` | Linux 编译架构，只允许 `amd64` / `arm64`；不参与本机构建的 target 写入 |
| `DIST_OS` / `DIST_ARCH` | 无 | 无 Go 二进制的构建必须显式成对提供；例如 `BACKEND=fastapi` |
| `DIST_DIR` | `dist/$(VERSION)` | 本次唯一的产物目录；不同平台分别指定不同目录 |
| `BUILD_DIR` | `backend/bin` | Go 二进制输出目录 |
| `FRONTEND_BUILD_CMD` | `npm --prefix web-admin run build` | 前端构建命令，可结合 CI 需求调整 |
| `DIST_ROOT` | `dist` | 提供给 PowerX `install/local` 的目录根 |
| `RELEASE_ROOT` | `target` | 发布产物根目录（用于交付或归档） |
| `DOCKER_IMAGE` | `powerx-plugin-base:$(VERSION)` | Docker 镜像名称 |

## 常用命令

| 命令 | 说明 |
| --- | --- |
| `make build` | 构建本机平台的后端二进制到 `$(BUILD_DIR)` |
| `make build-linux` | 交叉编译 Linux/`$(TARGET_ARCH)` 二进制 |
| `make frontend-build` | 构建 `web-admin/.output` 前端产物 |
| `make dist` | 生成 `$(DIST_ROOT)/$(VERSION)`，可通过 `PLATFORM=linux` 输出 Linux 后端二进制 |
| `make dist-linux` | `make dist PLATFORM=linux` 的兼容别名 |
| `make release` | 生成 `$(RELEASE_ROOT)/$(VERSION)`，包含前后端完整发布产物 |
| `make package` | 打包 `dist/<version>` 为 zip（适合远程安装接口） |
| `make package-release` | 打包 `target/<version>` 为 zip（对外发布文件） |
| `make docker-build` | 构建 Docker 镜像 |

查看所有目标：
```bash
make help
```

## 本地安装目录模式
1. 构建后端 & 前端：
   ```bash
   make build frontend-build
   ```
2. 生成目录：
   ```bash
   make dist
   ```
   指定产物版本/目录名：
   ```bash
   make dist DIST_VERSION=0.1.1
   ```
   Linux 目标（常用于在 macOS 上构建可部署到 Linux 的后端）：
   ```bash
   make dist VERSION=0.1.1 PLATFORM=linux TARGET_ARCH=amd64 DIST_DIR=dist/linux-amd64
   ```
3. 调用 PowerX 的安装接口时，`src_dir` 指向 `$(pwd)/dist/$(VERSION)` 即可，目录内至少包含：
   - `plugin.yaml`
   - `config/event_fabric.yaml`（事件 topic 执行层映射）
   - `backend/bin/plugin`
   - （可选）`web-admin/.output` 前端静态资源

端口规则：

- `make dist` 只生成 release artifact，不把某个环境的 PowerX 端口写死进产物。
- `plugin.yaml` 中 `POWERX_BIND_ADDR` 使用 `:__POWERX_DYNAMIC_PORT__`，插件后端端口由 PowerX 宿主在启用时分配并注入。
- 插件后端访问宿主网关使用运行时注入的 `PX_GATEWAY_BASE_URL`，浏览器侧 public 地址使用运行时注入的 `NUXT_PUBLIC_POWERX_CORE_BASE` / `NUXT_PUBLIC_WS_ORIGIN`。
- dev 环境若 PowerX 后端为 `server.port: 8081`、Web Admin 为 `web_admin_port: 3001`，这些值应来自 `/etc/powerx-dev/config.yaml` 和 systemd env，而不是 dist 目录。

## 产物目标系统与架构

`make dist` 在复制清单并写入本次版本号之后，只修改产物目录里的 `plugin.yaml`，补上唯一的目标声明：

```yaml
target:
  os: darwin
  arch: arm64
```

源码 `plugin.yaml` 保持原样，不写系统或架构；产物中已有的 `target` 会被本次构建结果完整覆盖。两个字段必须成对出现：`os` 只允许 `linux`、`darwin`、`windows`，`arch` 只允许 `amd64`、`arm64`。

- 默认 `PLATFORM=host` 使用 `go env GOOS`、`go env GOARCH`，与本次 Go 编译目标一致。Apple Silicon 默认得到 `darwin/arm64`；默认的 `TARGET_ARCH=amd64` 不用于本机构建。
- `PLATFORM=linux` 使用 `os=linux`、`arch=$(TARGET_ARCH)`。
- 打包结束必须用 `go version -m` 校验产物的主程序及可选迁移程序；它们的 `GOOS`、`GOARCH` 必须与清单的 `target` 一致。缺少构建元数据或目标不一致会使 `make dist` 失败，此检查不受 `DIST_VERIFY=0` 影响。
- 没有 Go 二进制的插件由构建命令显式传入 `DIST_OS`、`DIST_ARCH`，不从目录名或本机 Go 配置猜测。FastAPI 示例：

  ```bash
  make dist BACKEND=fastapi DIST_OS=linux DIST_ARCH=arm64 DIST_DIR=dist/linux-arm64
  ```

  该路径复制 Python 应用及依赖/迁移声明，不要求 Go 二进制；声明的目标由构建命令负责。

多个系统或架构使用相同版本号、分别打一个目录：

```bash
make dist VERSION=0.1.1 DIST_DIR=dist/darwin-arm64
make dist VERSION=0.1.1 PLATFORM=linux TARGET_ARCH=amd64 DIST_DIR=dist/linux-amd64
make dist VERSION=0.1.1 PLATFORM=linux TARGET_ARCH=arm64 DIST_DIR=dist/linux-arm64
```

以上本机构建例子针对 Apple Silicon。每次只生成指定的 `DIST_DIR`，仓库根入口不再额外复制到 `dist/mac`；目录名 `mac` 不作为系统/架构约定。在本仓库根目录使用代理入口时，相对 `DIST_DIR` 仍相对于 `skeleton/`；生成插件项目中相对于项目根目录。

市场后续应只读产物 `plugin.yaml` 的 `target` 来填写上传页系统与架构。**当前这段信息尚未接入上传页**。

## Release 产物
`make release` 会在项目根目录下创建 `target/<version>/`，目录结构示例：
```
target/
  0.1.0/
    plugin.yaml
    backend/bin/plugin
    web-admin/.output/
    README.md (若存在)
```

该命令默认依赖 `make build` 与 `make frontend-build`，确保前后端均为最新编译结果。若需要压缩包，可继续执行：
```bash
make package-release
```
压缩包默认命名为 `powerx-plugin-base-<version>-release.zip`。

## 自定义构建输出
- 修改后端输出目录：
  ```bash
  BUILD_DIR=backend/out make build
  ```
- 选择不同二进制平台：
  ```bash
  make dist VERSION=0.1.1 PLATFORM=linux TARGET_ARCH=amd64 DIST_DIR=dist/linux-amd64
  ```
- 更换发布目录根位置：
  ```bash
  RELEASE_ROOT=/tmp/px-release make release
  ```

## 注意事项
- `make frontend-build` 仅执行 `npm run build`，需提前安装依赖（`npm --prefix web-admin install`）。
- 如果不打算包含前端产物，可以跳过 `make frontend-build`；此时 `make dist` 会提示 `.output` 不存在。
- 发布前建议执行 `make check`（lint + test）确保质量，并在必要时运行 `make test-coverage`。

如需进一步自动化（例如 CI/CD），可参考以上变量，结合环境注入版本号或发布路径。
