# 联调差异归因 SOP

## 1. 范围
- 适用于 Nuxt 与 Next 在同一 Gin 契约下出现的行为差异。
- 目标是在 2 个工作日内完成归因与处置闭环。

## 2. 角色
- Feature Owner：最终归因责任人。
- Frontend Owner：提交 Next/Nuxt 对照证据。
- Backend Owner：提交 Gin 契约与日志证据。
- QA：执行复现、回归、门禁判定。

## 3. 流程
1. 建立 `Gap ID`，落到 `parity-gap-log.md`，状态 `open`。
2. 收集基线证据：Nuxt 请求/响应、页面行为、可复现步骤。
3. 收集迁移证据：Next 请求/响应、页面行为、可复现步骤。
4. 契约对比：检查 `contracts/openapi.yaml` 与实际请求路径/方法。
5. 判定根因：
   - `next_deviation`：Next 偏离 Nuxt 基线。
   - `gin_defect`：Gin 返回或行为违反既有契约/基线。
   - `unknown`：证据不足或多因素耦合。
6. 处置：
   - `next_deviation`：优先修 Next，并补对应 E2E。
   - `gin_defect`：按 `gin-defect-policy.md` 执行最小修复。
   - `unknown`：升级评审并冻结发布候选。
7. 关闭：补齐 `Resolved At`、结论、回归证据链接。

## 4. SLA
- T+4h：完成初始证据采集。
- T+24h：完成初判。
- T+48h：完成修复/缓解与结案。

## 5. 产出物
- `parity-gap-log.md`
- `regression-evidence-template.md`（实例）
- E2E 执行日志（命令、结果、时间戳）
