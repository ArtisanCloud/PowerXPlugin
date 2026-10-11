# Framework Go 语义 Host 真实验收

这是可选的独立验收模块，使用本地 `replace` 的 Framework SDK。Core 的模型与设置服务只用于创建和清理临时夹具，SDK 本身不依赖 Core。测试需要同工作区的 Core checkout、已经迁移并加载新代码的 Core HTTP/gRPC、真实 Ollama 与同库 pgvector。

运行前确认 8077/9001 对应目标开发实例，且开发 Host Key 已具有正式 retrieval.read/document.manage grant。测试不补写开发 Key 授权。它创建有唯一前缀的空间、文档、临时 STS 凭据和只读 Key，并在结束后清理；不用于生产实例。

```bash
cd tests/acceptance/knowledge-semantic-sdk
GOWORK=off \
POWERX_KNOWLEDGE_CONFIG=/absolute/private/core-config.yaml \
POWERX_KNOWLEDGE_HTTP=http://127.0.0.1:8077 \
POWERX_KNOWLEDGE_GRPC=127.0.0.1:9001 \
POWERX_KNOWLEDGE_EVIDENCE=/absolute/path/framework-semantic-evidence.json \
go test -run '^TestLiveFrameworkSemanticIndexAndRetrieval$' -count=1 -timeout=10m -v
```

该夹具要求已有激活的 `ollama/bge-m3` 开发空间作配置模板。覆盖原文/画像实际向量、640/80 分块、完整需求检索、过滤、来源区间、同文档多块、单篇/空间重建、失败旧索引保护、模型切换、跨租户、只读 Key、撤销 grant、版本冲突与来源损坏。Admin JWT 正向验收不在此模块内；真实 STS 请求 Admin 平面必须拒绝。

证据分类必须分别统计：

- `framework_sdk:*`：实际 SDK 发出的 HTTP 请求及原始受控响应、Trace。
- `framework_local_validation:*`：客户端拒绝，没有 HTTP 请求，不能计入真实 HTTP 调用数。
- `core_raw_negative_or_admin:*`：Core 的原始非法 DTO / Registry / Admin 隔离负例。
- `real_sts_exchange`、`real_pgvector_readback`：真实交换和向量存储核对。

证据不记录凭据。配置验证使用仅限测试加载器的临时 JWT key，不签发 HTTP Admin 用户身份，也不更改运行实例的签名配置。生产消费代码不得复制这种夹具加载方式。
