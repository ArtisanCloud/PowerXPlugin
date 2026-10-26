# HTTP 启动就绪通知

`App.OnHTTPReady(func(HTTPReadyInfo))` 供插件在 HTTP 监听就绪后输出终端提示或记录启动指标。应在 `Run()` 前完成依赖、路由注册和回调配置。

`HTTPReadyInfo` 包含：

| 字段 | 含义 |
|---|---|
| `ListenAddr` | 实际绑定的 TCP 地址，使用动态端口时包含分配后的端口 |
| `ReadyAt` | 成功绑定后生成的 `time.Time`，保留单调时钟用于计算进程内耗时 |

`Run()` 先绑定监听地址，成功后同步通知，再进入 `http.Server.Serve`。绑定失败直接返回原错误，不触发就绪通知；未注册回调时照常提供 HTTP 服务。通过 `App.Shutdown()` 在启动前关闭的应用不会绑定或通知就绪。正常关闭不返回 `http.ErrServerClosed`。

回调应简短，不应同步等待本服务的 HTTP 响应，因为此时尚未进入请求处理循环。这是 HTTP 监听生命周期通知，不代表异步任务、外部模型、所有后台 worker 或业务数据已验收。Framework 不规定终端文案和语言，不输出含密钥的配置。插件负责本地化、终端输出与结构化日志。

验证：`go test -race ./bootstrap ./router`。测试使用临时 TCP 端口，覆盖实际绑定地址、请求处理、绑定冲突不通知、启动前关闭、无回调运行及关闭后端口释放。
