# Contract Drift Checklist（禁止 Next 私有接口漂移）

- [ ] 已执行 `scripts/check-contract-drift.sh`
- [ ] Next API 路径全部可映射到 `contracts/openapi.yaml`
- [ ] 若发现新增路径，已在差异单登记且阻断发布
- [ ] 未出现仅 Next 使用、Nuxt/Gin 基线不存在的接口
- [ ] 路由前缀符合规范（standalone `/api/v1` / host `/_p/{pluginId}/api/v1`）
- [ ] 差异处理结论已同步到 `parity-gap-log.md`
