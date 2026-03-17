# Parity Triage Drill（演练脚本）

## 目标
- 在 2 个工作日内完成一次联调差异从发现到归因闭环。
- 演练结论必须可审计、可复盘、可用于发布门禁。

## 演练输入
- 差异编号：`GAP-DRILL-001`
- 场景：Next `templates/crud` 删除成功提示与 Nuxt 不一致
- 基线证据：Nuxt 录屏 + 请求响应快照

## 演练步骤
1. 在 `parity-gap-log.md` 记录初始条目（`root_cause=unknown`）。
2. 由前端同学复现实验并提交 Next 侧证据（HAR、控制台、截图）。
3. 由后端同学核对 Gin 契约与日志，判断是否契约缺陷。
4. 若 4 小时内无法定性，升级到联调负责人并冻结发布候选。
5. 24 小时内给出初判：`next_deviation` / `gin_defect` / `unknown`。
6. 48 小时内必须关闭：
   - `next_deviation`：修复 Next 并补 E2E。
   - `gin_defect`：按 `gin-defect-policy.md` 最小修复并双端回归。
   - `unknown`：输出阻塞原因、风险评级、临时门禁策略。
7. 在 `regression-evidence-template.md` 填写证据并归档。

## 通过标准
- `Resolved At - Opened At <= 2 工作日`
- 证据包齐全（请求、响应、页面、日志、结论）
- 门禁结论明确（放行/阻断）
