# Parity Gap Log

| Gap ID | Domain | Symptom | Baseline Evidence | Root Cause | Decision | Opened At | Resolved At |
|---|---|---|---|---|---|---|---|
| GAP-001 | TEMPLATE | 示例：列表排序不一致 | nuxt-page-video | unknown | pending | 2026-03-14T00:00:00Z | |
| GAP-002 | CONTRACT | Next 使用 `/admin/user/auth/register`，未在 openapi 基线声明 | `scripts/check-contract-drift.sh` 输出 | unknown | pending（阻断发布，待合同补齐或接口收敛） | 2026-03-15T00:00:00Z | |

## Rules

- Root Cause 仅允许：`next_deviation`、`gin_defect`、`unknown`。
- 若 Root Cause 为 `gin_defect`，Decision 必须包含“最小化修复 + 双端回归”。
- `Opened At` 到 `Resolved At` 需满足 2 个工作日内闭环。

## SLA 统计模板

| Week | Opened | Resolved | <=2WD Resolved | SLA Rate | Owner | Notes |
|---|---:|---:|---:|---:|---|---|
| 2026-W11 | 2 | 0 | 0 | 0.00% | FE+BE Joint | 含 contract drift 新增条目 GAP-002 |
