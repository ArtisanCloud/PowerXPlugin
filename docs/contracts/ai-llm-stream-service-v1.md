# Framework LLM 流式适配 v1

Core 权威合同：`../../PowerX/docs/contracts/ai-llm-stream-service-v1.md`。

`runtime/powerx/ai.Client.LLMStream` 通过 `hostcontract.InvokeStream` 请求 `com.corex.ai.llm.stream`，明确使用 core_internal；不接受调用方任意 URL、方法、鉴权头或租户覆盖，也不回退到直接 REST 或非流式调用。

Core 的 start/delta/usage 原样映射为统一 LLM 事件；成功 done 必须含 ok=true，映射为 final 后 end。未知事件、旧 end/completed 终止帧、done 后继续输出、异常 EOF 或取消均不能返回成功。供应商 delta 不由模板替换。HTTP 错误保留 Core 的安全原因码及独立追踪 ID。

该适配只决定传输，不改变业务分类、资料提交、购买确认、协作授权或付款/生产状态。其他场景继续保留自身 Thinking 策略。受控流测试不代表已部署 Core 的服务 grant 或实际 Shopify 链路已验收。
