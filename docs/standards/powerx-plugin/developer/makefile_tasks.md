# Makefile 任务与构建说明（Makefile Tasks & Build System）

> 本页目标：帮助开发者理解 **PowerX Plugin Base** 的多文件 Make 体系，  
> 包括任务划分、变量继承、构建流程与发布模式。  
> 读者对象：工程师 / CI 维护者 / 发布负责人。

> 本文同时定义插件项目必须实现的 `make dist`、`make local-install`、`make local-reinstall` 合同，以及标准 `dist/<version>` 目录结构。

---

## 一、系统概述

PowerX 插件模板采用模块化的 **Makefile 体系**，  
所有命令集中定义在以下文件中：

```

make-files/
├── build.mk        # 后端编译与打包逻辑
├── common.mk       # 公共变量、颜色输出与帮助函数
├── dev.mk          # 本地开发初始化（lint、deps）
├── docker.mk       # Docker 构建与运行
├── migrate.mk      # 数据库迁移任务
├── project.mk      # 主入口（include 所有子 makefile）
└── test.mk         # 测试与覆盖率

````

项目根目录通常包含一个顶层 `Makefile`：

```makefile
include make-files/project.mk
````

这样可以直接在根目录执行：

```bash
make build
make package
make docker-build
```

---

## 二、核心设计思想

| 原则        | 说明                                                   |
| --------- | ---------------------------------------------------- |
| **模块化**   | 每个 `.mk` 文件负责一个构建领域（编译 / 测试 / 打包 / Docker）。          |
| **可覆盖变量** | 变量都可通过命令行覆盖，如 `VERSION=0.1.2 make build`。            |
| **显式依赖**  | 各任务显式调用子任务，例如 `release` 依赖 `build + frontend-build`。 |
| **CI 友好** | 所有路径、版本号均从变量读取，方便注入环境参数。                             |

---

## 三、关键变量

| 变量名                  | 默认值                                | 说明                                   |
| -------------------- | ---------------------------------- | ------------------------------------ |
| `VERSION`            | `0.1.0`                            | 当前插件版本号                              |
| `BUILD_DIR`          | `backend/bin`                      | Go 二进制输出路径                           |
| `FRONTEND_BUILD_CMD` | `npm --prefix web-admin run build` | 前端构建命令                               |
| `DIST_ROOT`          | `dist`                             | 本地安装目录根（PowerX `install/local` 模式使用） |
| `DIST_DIR`           | `$(DIST_ROOT)/$(VERSION)`          | 本次 `make dist` 的最终输出目录，可显式覆盖为 `dist/mac` 等 |
| `RELEASE_ROOT`       | `target`                           | 发布产物目录根                              |
| `DOCKER_IMAGE`       | `powerx-plugin-base:$(VERSION)`    | Docker 镜像名称                          |
| `PROJECT_NAME`       | `powerx-plugin-base`               | 插件名称（影响压缩包名）                         |
| `GO_BUILD_FLAGS`     | 空                                  | 额外 Go 构建参数（如 `-tags release`）        |

> 可在执行命令时覆盖任意变量：
>
> ```bash
> BUILD_DIR=backend/out VERSION=0.1.1 make build
> ```

---

## 四、主要任务分类

### 🧱 Build & Compile

| 命令                     | 说明                                                   |
| ---------------------- | ---------------------------------------------------- |
| `make build`           | 构建本机平台后端二进制（默认输出至 `backend/bin/plugin`）              |
| `make build-linux`     | 交叉编译 Linux 二进制（默认 amd64，可用 `TARGET_ARCH` 覆盖）         |
| `make frontend-build`  | 执行 `npm --prefix web-admin run build` 构建前端 `.output` |
| `make dist`            | 生成安装目录结构到 `dist/<version>/`（支持 `PLATFORM=linux`）      |
| `make dist-linux`      | `make dist PLATFORM=linux` 兼容别名                        |
| `make release`         | 构建完整发布产物 `target/<version>/`（含前后端）                   |
| `make package`         | 压缩 `dist/<version>` 目录为 zip                          |
| `make package-release` | 压缩 `target/<version>` 目录为 zip                        |
| `make clean`           | 清除缓存与临时文件                                            |

#### 产物目录结构示例

```
dist/
  0.1.0/
    plugin.yaml
    backend/bin/plugin
    backend/bin/migrate              # 可选，存在 cmd/database 时生成
    web-admin/.output/
    plugin.d/
    config/event_fabric.yaml
```

---

### 🧪 Test & Check

| 命令                   | 说明                           |
| -------------------- | ---------------------------- |
| `make lint`          | 运行 `golangci-lint run ./...` |
| `make test`          | 执行单元测试                       |
| `make test-coverage` | 输出测试覆盖率报告                    |
| `make check`         | 连续执行 lint + test             |

> 若未安装 `golangci-lint`，执行 `make dev-setup` 会自动安装到 `$(GOPATH)/bin`。

---

### 🧰 Dev Utilities

| 命令               | 说明                       |
| ---------------- | ------------------------ |
| `make dev-setup` | 初始化开发依赖（Go 工具、Node 模块）   |
| `make run`       | 启动后端服务（含日志与热重载配置）        |
| `make migrate`   | 运行数据库迁移（参照 `migrate.mk`） |
| `make seed`      | 初始化基础数据                  |
| `make fmt`       | 执行 go fmt 与 eslint 格式化   |

---

### 🐳 Docker

| 命令                  | 说明                              |
| ------------------- | ------------------------------- |
| `make docker-build` | 构建 Docker 镜像（使用 `DOCKER_IMAGE`） |
| `make docker-run`   | 运行容器并暴露端口                       |
| `make docker-clean` | 清理旧镜像与容器                        |

Docker 构建过程默认包含：

```bash
docker build -t $(DOCKER_IMAGE) .
```

> 可通过以下覆盖镜像标签：
>
> ```bash
> DOCKER_IMAGE=registry.mycorp.com/powerx/base:1.0.0 make docker-build
> ```

---

## 五、组合任务依赖关系

```
release
 ├─ build
 ├─ frontend-build
 └─ dist
      └─ package-release
```

每个高层任务自动依赖前置任务：

| 主任务               | 自动依赖                            |
| ----------------- | ------------------------------- |
| `release`         | `build + frontend-build + dist` |
| `package`         | `dist`                          |
| `package-release` | `release`                       |
| `docker-build`    | `build`                         |

---

## 六、CI/CD 建议集成示例

### GitHub Actions 示例

```yaml
name: Build & Release
on:
  push:
    tags: ['v*']

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - uses: actions/setup-node@v4
        with: { node-version: '18' }

      - name: Install deps
        run: make dev-setup

      - name: Build release
        run: VERSION=${GITHUB_REF_NAME#v} make release

      - name: Package zip
        run: make package-release
```

---

## 七、环境变量注入（CI/本地）

常见覆盖用法：

| 环境变量                             | 说明                |
| -------------------------------- | ----------------- |
| `POWERX_ENV=dev`                 | 当前环境名             |
| `VERSION=$(git describe --tags)` | 从 git tag 自动提取版本号 |
| `GOOS/GOARCH`                    | 交叉编译目标平台          |
| `FRONTEND_BUILD_CMD`             | 替换默认前端构建逻辑        |
| `RELEASE_ROOT`                   | 替换默认产物根目录         |

---

## 八、发布产物与安装方式

### dist 合同（插件项目必须对齐）

先区分两类项目：

| 场景 | 命令执行目录 | `make dist` 输出 |
|---|---|---|
| PowerXPlugin 主仓库调 skeleton | PowerXPlugin 仓库根目录 | `skeleton/dist/<version>` |
| 独立/二次插件仓库 | 插件项目根目录 | `dist/<version>` |

PowerXPlugin 主仓库是 framework + skeleton 模板仓库，根目录 `make dist` 只是代理到 `skeleton`。独立/二次插件仓库不得输出到 `skeleton/dist`，必须直接在自己的插件项目根目录输出 `dist/<version>`。

独立/二次插件仓库根目录必须支持：

```bash
make dist
make dist VERSION=0.1.0 DIST_DIR=dist/mac
make local-install API_BASE=http://127.0.0.1:8077/api/v1 TOKEN=<ADMIN_BEARER_TOKEN>
make local-reinstall VERSION=<version> API_BASE=http://127.0.0.1:8077/api/v1 TOKEN=<ADMIN_BEARER_TOKEN>
```

语义：

- `make dist`：只构建并生成 `DIST_DIR`，默认 `dist/<version>`，不调用 PowerX。
- `make local-install`：先执行 `dist`，再调用 PowerX `/admin/plugins/install/local`。
- `make local-reinstall`：执行 disable -> force install(enable=false) -> switch_version(enable=true)，用于本地反复验证同一插件。

标准 dist 目录结构：

```text
dist/<version>/ 或显式 DIST_DIR 指向的目录
  plugin.yaml
  plugin.d/
    capabilities.yaml
    exposure.yaml
    rbac.yaml
    events.yaml
  config/
    event_fabric.yaml
  backend/
    bin/
      plugin
      migrate              # 可选，存在 cmd/database 时生成
  web-admin/
    .output/
      public/
        icon.svg           # metadata.icon 指向的市场图标
    i18n/                # 可选
  README.md              # 可选
```

文件要求：

| 路径 | 要求 |
|---|---|
| `plugin.yaml` | 安装目录主清单，版本号必须与当前 `VERSION` 一致 |
| `plugin.yaml.metadata.icon` | 市场页图标相对路径，例如 `icon.svg` |
| `plugin.d/capabilities.yaml` | 能力目录，必须由能力同步工具生成或校验 |
| `plugin.d/exposure.yaml` | 路由/暴露目录，新增接口必须覆盖 |
| `plugin.d/rbac.yaml` | RBAC 目录，新增接口必须覆盖 |
| `plugin.d/events.yaml` | 事件声明目录，topic 必须与执行层一致 |
| `config/event_fabric.yaml` | PowerX 启用插件时播种 topic/ACL 的执行层配置，必须包含顶层 `version: v1` |
| `backend/bin/plugin` | 后端可执行文件，Go/Gin 插件必须存在且可执行 |
| `backend/bin/migrate` | 可选，声明迁移或存在 `cmd/database` 时应生成 |
| `web-admin/.output` | Nuxt 管理端生产产物，必须完整复制 `.output` |
| `web-admin/.output/public/<metadata.icon>` | PowerX 市场页会通过 `/_p/<plugin_id>/admin/<metadata.icon>` 读取并展示 |

`plugin.yaml` 中入口必须与 dist 结构一致：

```yaml
runtime:
  entry: backend/bin/plugin
backend:
  entry: backend/bin/plugin
migrations:
  - entry: backend/bin/migrate
```

`config/event_fabric.yaml` 最小合法结构：

```yaml
version: v1
topics:
  - topic: _topic.example.updated
    description: example event
    acl:
      - actions: [publish, subscribe]
```

缺少 `version` 会在启用阶段被 PowerX 严格拒绝，典型错误为：

```text
manifest version must be positive
```

`make dist` 至少完成：

1. 执行 `plugin-yaml-check`。
2. 构建后端；Linux 部署包使用 `make dist PLATFORM=linux TARGET_ARCH=amd64`。
3. 构建前端 Host 包：`POWERX_PROXY=1`，baseURL 使用 `/_p/<pluginId>/admin/`。
4. 写入当前插件项目的 `DIST_DIR/plugin.yaml`，并用 `VERSION=<version>` 覆盖版本；不得回写源码根目录 `plugin.yaml`。
5. 复制 `plugin.d/` 与 `config/event_fabric.yaml`。
6. 复制后端二进制到 `backend/bin/plugin`。
7. 若存在 `backend/cmd/database` 或 `plugin.yaml` 声明 migrations，则复制迁移二进制到 `backend/bin/migrate`。
8. 复制 Nuxt 产物到 `web-admin/.output`。
9. 执行内建 dist 验证。

dist 阶段必须 fail fast，最低验证项：

- `plugin.yaml` 存在。
- `plugin.d/rbac.yaml`、`plugin.d/exposure.yaml` 存在。
- `config/event_fabric.yaml` 存在。
- `config/event_fabric.yaml` 顶层 `version` 存在且为正版本，例如 `v1`。
- `backend/bin/plugin` 存在且非空、可执行。
- 如声明 migrations，`backend/bin/migrate` 必须存在且非空、可执行。
- `web-admin/.output` 存在。
- `plugin.yaml.metadata.icon` 配置后，`web-admin/.output/public/<metadata.icon>` 必须存在。
- `plugin.d/events.yaml` 与 `config/event_fabric.yaml` 的 topic 名称一致。

### 迁移与宿主注入合同

PowerX 本地安装会先把 dist 拷贝到安装目录，再执行 `backend/bin/migrate setup`。迁移进程的工作目录是安装包内的 `backend/`，并由宿主注入运行期环境变量。

插件迁移入口必须遵守：

- 迁移入口只加载迁移所需配置，例如 `LoadForMigration()`。
- 迁移阶段不得强制要求 gateway STS/API Key 等运行态凭证。
- 数据库连接以宿主注入的 `POWERX_DB_DSN` 为准。
- 若 `POWERX_DB_SCHEMA` 缺失，但 `POWERX_DB_DSN` 包含 `search_path=<schema>`，插件必须从 DSN 推导 schema。
- 插件迁移不得执行 `CREATE SCHEMA`。schema 生命周期由 PowerX 安装器负责，插件只在已分配 schema 内建表和写 seed。
- `refresh` 等破坏性迁移命令必须显式开关保护，不得在安装流程默认执行。

典型错误与原因：

| 错误 | 原因 | 修复 |
|---|---|---|
| `gateway config requires base_url and matching credential` | migrate 使用普通运行态配置加载，错误要求 gateway 凭证 | 改用迁移专用加载，跳过 runtime gateway 校验 |
| `permission denied for database powerx` | 插件迁移尝试 `CREATE SCHEMA` | 移除插件侧建 schema，改为校验宿主已创建的 schema |
| `schema "powerx_plugin_base" does not exist` | 未读取宿主注入的 `POWERX_DB_DSN search_path`，落回本地默认 schema | 从 `POWERX_DB_DSN` 的 `search_path` 推导 schema |
| `manifest version must be positive` | `config/event_fabric.yaml` 缺少顶层 `version` | 添加 `version: v1` |

`POST /admin/plugins/install/local` 的 `src_dir` 必须指向含 `plugin.yaml` 的版本目录：

```text
/absolute/path/to/plugin/dist/<version>
```

PowerXPlugin 主仓库调 skeleton 时是：

```text
/absolute/path/to/PowerXPlugin/skeleton/dist/<version>
```

不要传插件源码根目录、`dist/` 根目录、`web-admin/` 目录、`bin/` 目录或 `.pxp` 文件路径。

### 本地目录模式

```bash
make build frontend-build
make dist VERSION=0.1.0 DIST_DIR=dist/mac
```

PowerX 可直接安装：

```
install/local?src_dir=$(pwd)/dist/mac
```

### Release 模式（对外分发）

```bash
make release
make package-release
```

生成：

```
target/0.1.0/
└── powerx-plugin-base-0.1.0-release.zip
```

---

## 九、扩展与自定义建议

| 场景      | 建议操作                                             |
| ------- | ------------------------------------------------ |
| 添加新任务   | 在 `project.mk` 引入自定义 `.mk` 文件                    |
| 拆分多插件构建 | 使用变量 `PLUGIN_ID` + `PLUGIN_PATH` 控制              |
| 多平台构建   | 结合 `GOOS/GOARCH` 循环调用                            |
| 版本追踪    | 在 CI 中注入 `VERSION=$(git rev-parse --short HEAD)` |
| 环境隔离    | 在 `Makefile` 中使用 `.env.<stage>` 文件加载变量           |

---

## 十、常见问题（FAQ）

**Q:** 为什么 `make frontend-build` 报错 `.output` 不存在？
A: 先执行 `npm --prefix web-admin install`，再运行构建命令。

**Q:** 如何跳过前端构建？
A: 执行 `make build dist` 即可，`make release` 会自动跳过缺失的前端目录检查。

**Q:** Windows 下执行报错？
A: 推荐使用 WSL2 / Docker 环境执行构建任务。

---

## 下一步阅读

* 🚀 [部署与 Docker 指南](../deploy/docker_guide.md)
* 🧩 [plugin.yaml 规范](../contract/plugin_yaml_spec.md)
* 🔧 [构建与打包指引（build.mk 详细说明）](../../make-files/guide.md)
