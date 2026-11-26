# Base 0.1.5：市场人工审核协作步骤

日期：2026-11-17。双方职责不变：Framework 操作开发者发布页面，Market 维护者操作市场审核后台；本轮不修改 Market、Core 或日常服务。

## 当前做到哪里

Framework 已用页面上传原始 Base 0.1.5 Linux/amd64 ZIP，并成功提交审核。页面显示“已送审，待审核”；送审响应丢失、刷新页面、重启本地执行器后保持原任务和 operation，没有重复申请。发现的“送审后刷新完整集合被草稿门禁拦截”已在 Framework 修正；提交写入仍检查 stable + draft/rejected。

本轮临时市场于 2026-11-17 15:07 到期并已清理。旧地址、登录引导及申请链接仅留作历史证据，不能继续操作。该包的驳回、重提、批准和上架尚未完成；不能用以前受控包的四态截图替代。

## 谁接下来做什么

1. **双方约好同一验收窗口**。Framework 使用 Market 提供的原样隔离工具准备环境，在发布页面重新上传同一冻结包、送审，交付新的有效申请链接及私密登录引导文件路径。用户不需要执行命令、复制令牌或改配置。
2. **Market 先驳回原申请**，理由填写“联调测试：验证驳回后重新送审”，完成后回复“已驳回”。此时不要批准或上架。
3. **Framework 刷新页面、核对已驳回截图，修改资料并再次确认送审**。产生新任务、新 operation、新申请链接；核对旧申请资料、包集合、回执及旧链接保持原值，再把新链接交给 Market。
4. **Market 批准新申请，先不要上架**，回复“已批准”。Framework 刷新页面并截图“审核通过，待上架”；确认后通知 Market 执行下一步。
5. **Market 明确上架新申请**，回复“已上架”。Framework 刷新页面截图“已公开上架”，检查旧申请仍显示已驳回并保持旧回执，然后整理联合记录并清理自己的临时环境和凭据。

市场不要修改 Framework 任务文件或替代客户端直接调用送审；Framework 不代替管理员驳回、批准或上架。下一轮必须使用新交接，不能把已关闭实例的 Publication UUID 带入新实例。

## 可直接转给 Market

> Framework 的 Base 0.1.5 已完成真实网页上传和送审，剩下需要你配合人工审核。请约一个双方在线的窗口。收到新的申请链接后，先驳回并告诉 Framework；Framework 在页面修改资料、重提后会给你新链接。再批准新申请但暂不上架，待 Framework 截图“通过待上架”后，再按通知上架。你负责市场后台三个动作，Framework 负责页面操作、恢复、截图和旧记录核对。

## 已保留的证据

- [本轮真实上传、送审及恢复记录](../contracts/evidence/plugin-publisher-base-page-joint-validation-20261117.json)
- [真实待审核页面](../contracts/evidence/plugin-publisher-base-page-20261117/review-submitted.png)
- [桌面深色](../contracts/evidence/plugin-publisher-base-page-20261117/review-submitted-desktop-dark.png)、[窄屏浅色](../contracts/evidence/plugin-publisher-base-page-20261117/review-submitted-narrow-light.png)、[窄屏深色](../contracts/evidence/plugin-publisher-base-page-20261117/review-submitted-narrow-dark.png)
- [Market 已交付的准入修复](../../../../PowerXPluginMarket/docs/marketplace/plugin-publisher-base-admission-delivery-20261116.md)

公开下载、PowerX 安装启用、Linux Secret Service、独立正式 SDK/CLI 消费仍另行验收；本轮没有发布 SDK/CLI。
