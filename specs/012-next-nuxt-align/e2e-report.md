# E2E Report (Phase 6 Full Run)

## 执行信息
- 执行日期: 2026-03-15
- 执行命令: `cd skeleton/web-admin/next && npm run e2e`
- 浏览器: Playwright Chromium

## 汇总

| Metric | Value |
|---|---:|
| Total | 22 |
| Passed | 15 |
| Failed | 2 |
| Skipped | 5 |
| Pass Rate (excluding skipped) | 88.24% |

## 失败用例

| Case | Failure Summary |
|---|---|
| `tests/e2e/error-semantics.spec.ts` -> IAM endpoint returns envelope code/message and UI keeps semantics | `getByRole('alert')` 命中 2 个元素（业务 alert + Next route announcer），严格模式冲突 |
| `tests/e2e/mode-parity-edge.spec.ts` -> standalone and host paths are both reachable | `/_p/{pluginId}/admin/integration` 未命中 `host-proxy-page` 断言 |

## 跳过用例说明
- `auth-delegated.spec.ts` / `auth-local.spec.ts` / `iam-local.spec.ts` 受环境变量门控（`PLAYWRIGHT_LOCAL_IAM`、`NEXT_PUBLIC_DELEGATED_IAM`），当前执行环境未启用对应模式。

## 结论
- US2/US3 主体场景在当前环境已可执行，核心回归通过率较高。
- 仍有 2 个未闭环问题，已纳入 parity 风险与发布门禁。
