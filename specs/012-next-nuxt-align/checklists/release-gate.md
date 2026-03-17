# Release Gate Checklist

- [ ] 已完成 US1/US2 E2E 关键链路回归（登录、模板 CRUD、IAM、Capabilities、Integration/Ops/Security）
- [ ] `parity-map.md` 状态已更新为最新迁移结果并标注 residual risks
- [ ] `parity-gap-log.md` 无超 2 工作日未归因条目
- [ ] 阻断级与高优先级问题为 0
- [ ] 中低优先级问题具备书面风险评估与处置计划
- [ ] 若存在 Gin 修改，已附“缺陷确认 + 最小修复 + 双端回归”证据
- [ ] `check-contract-drift.sh` 执行结果已归档
- [ ] 发布产物路径校验通过（与宿主消费路径一致）
- [ ] 回滚预案已演练并通过
- [ ] 发布审批人签字完成
