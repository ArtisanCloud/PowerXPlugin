# AI Media v2 模型目录限制字段交付

本次新增 `MediaModelOperation.max_image_bytes`、`max_total_image_bytes`、`max_image_pixels`。它们是 Core 目录中的可选限制；当前用于 `operations.image_generation`，不属于生成请求的 params。

Go SDK 使用 `*int64` 和 `omitempty`：缺省保持 nil，明确的 0 保留为非 nil 指针并再次序列化。Nuxt 公共类型使用可选 number；`listModels()` 原样保留返回值。SDK 不定义这些限制的默认值，不把缺省或 0 解释为无限制，不读取其他型号的限制作替代。

## 来源与样例

`ai-media-v2.md`、`ai-media-v2.openapi.yaml`、`fixtures/ai-media-v2.json`、`fixtures/capability-version-upgrade-errors.v1.json` 保持 Core 原文，四份摘要继续记录在 `ai-media-v2-source.json.sha256`。来源为当前 Core 未提交的源码快照，不能当作已发布环境的证明。

Core 当前共享 `ai-media-v2.json` 未包含新限制。Framework 增加独立的 `fixtures/ai-media-model-limits.v1.json`，明确标记 `fixture_owner=Framework`，摘要记录在 `framework_model_limits_fixture`。它提供声明值、未声明、显式零值、其他限制值四组样例。具体数值仅属于测试数据，未写入 SDK 的生产逻辑。

API Key 和 STS 受控测试均走现有 `com.corex.ai.catalog.models.read`、固定 typed Core binding；没有新增 capability、路径、凭据、任务或 SSE 协议。Skeleton 实际 `/admin/ai-settings/media/models` 转发测试覆盖完整的 SDK 解码 → 运行适配器 → HTTP 重新序列化路径。

## SDK 与生成项目

CLI `contracts.Files()` 导出新 fixture 和本说明；文档、SDK testdata、CLI embed 和生成项目使用相同内容。Go SDK 开发包包含新类型及解码测试数据，Nuxt tarball 包含更新后的 ai-media.ts 和公共导出。打包检查通过压缩包内真实客户端的 `listModels()` 测试和独立 TypeScript 编译，不能以相邻源码通过代替消费证明。

运行 `python3 scripts/contracts/package-ai-media-limits.py` 可生成本地可消费开发包。`artifacts/ai-media-model-limits-latest.json` 给出 SDK、Nuxt、CLI 的内容固定版本、Go checksums、tarball integrity、源码与产物 SHA256，以及生成项目验证记录。该产物不是公开发布版本；本轮不创建提交、tag 或推送。版本以 base commit 和内容摘要追踪。

## MediaXStudio 升级边界

MediaXStudio 当前使用自己的固定 Framework 快照；相邻源码变化不会更新该快照。消费方收到产物后，按自身快照流程更新 Go SDK 和 Nuxt 客户端，并核对 refs 中的摘要。若使用 Go module 开发包，应检查实际版本及 Replace，避免仍指向旧本地快照。

MediaXStudio 负责尝试创建前的大小／像素校验与界面提示，按当前型号实际声明判断。Framework 只负责类型、完整透传、合同测试和产物，不新增图片解析、压缩、换图或生成逻辑。本轮不修改 MediaXStudio、Core 数据库、授权和日常服务。

验收记录：`evidence/ai-media-model-limits-framework-validation-20261107.json`。受控 API Key／STS 测试只证明 Framework 传输与解码，不代替真实 Core 凭据、实际 Provider 和 MediaXStudio 页面验收。
