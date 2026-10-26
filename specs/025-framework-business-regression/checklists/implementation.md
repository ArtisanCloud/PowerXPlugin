# Implementation verification

- [x] Go Scenario SDK 不依赖 testing.T 或 gin.Context；定义和必要断言冻结。
- [x] UUID 主键、跨表引用、去重、租户事务和 schema-qualified 表名。
- [x] 原子快照／事件，严格六项 Meta，游标限定运行和租户。
- [x] 默认禁用、开发／测试环境门控；缺依赖／缺表明确失败。
- [x] 串行执行、协作取消、独立清理、清理重试保持原始 verdict。
- [x] 三项模板业务断言、重复运行和故障结果／资源清理。
- [x] live 独立 adapter、测试租户／资源固定，无全局 ProviderMode 改写。
- [x] 页面 locale、名称选择、权限、快照／SSE、历史、报告。
- [x] CLI UUID／同一请求键、SSE／断线／退出码与真实 API 报告。
- [x] 双模板同步、两个生成 consumer 的本地 SDK 业务执行。
- [x] 原生 Go、SQLite、真实 PostgreSQL RLS、Nuxt／客户端、浏览器分层证据。
- [ ] 真 Core grant/revocation／真实集成闭环：用户确认暂时无专用环境。
- [ ] 已发布 Framework／安装态 consumer：未授权发布，本次不作此声明。
