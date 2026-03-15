# E2E Report (US2)

## 统计范围
- 统计日期: 2026-03-14
- 统计对象: IAM 与 Capabilities 在 Next 端的 US2 回归
- 关联用例:
  - `tests/e2e/iam-local.spec.ts`
  - `tests/e2e/capability-invocation.spec.ts`
  - `tests/e2e/error-semantics.spec.ts`

## 一次通过率

| Domain | Cases | First-Pass Success | First-Pass Rate | 备注 |
|---|---:|---:|---:|---|
| IAM | 1 | 0 | 0% | Playwright 浏览器二进制缺失，未进入用例执行阶段 |
| Capabilities | 2 | 0 | 0% | Playwright 浏览器二进制缺失，未进入用例执行阶段 |
| Combined | 3 | 0 | 0% | 需执行 `npx playwright install` 后复测 |

## 本次执行记录
- `npx tsc --noEmit`: 通过
- `npx playwright test tests/e2e/route-parity.spec.ts --project=chromium --reporter=list`: 启动失败（缺少 Chromium headless shell）

## 结论
- 已完成 US2 的用例实现与断言编排，统计模板可直接用于 CI/本地实跑后回填。
- 当前阻塞点是浏览器运行时未安装，不是业务断言失败。
