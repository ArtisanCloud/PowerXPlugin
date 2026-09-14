# Host Contract Status Page 验证记录

更新时间：2026-09-12。

“底座合同调试”只自动请求各 Framework 模块的 `status`，用于显示 adapter 是否已装配。它不提供手工 JSON 输入、结果复用面板或任何业务写操作。

```bash
go test ./framework/backend/go/runtime/powerx/hostcontract ./skeleton/backend/go-gin/internal/transport/http/admin/host_contract -count=1
npm run check:host-contract-lab
npm run sync:templates -- --check
```

页面 status 通过只表示当前进程已装配对应 adapter；真实 PowerX 授权、上游连通性及业务调用仍应在“PowerX 能力调试”或相应业务页面验证。
