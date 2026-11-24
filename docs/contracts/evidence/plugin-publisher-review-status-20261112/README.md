# 送审制品集合状态回归：2026-11-12

修复范围为 Framework 送审页的状态来源。真实隔离 Market、实际 CLI／系统凭据库和独立插件 SDK tarball 参与验证；没有替换业务响应。输入为实验环境通过真实上传接口登记的受控 ZIP，不代表原生完整插件构建、Framework 构建上传、公开下载或 PowerX 安装通过。

## 四种状态截图

| 市场申请状态 | 桌面浅色（1440） | 窄屏深色（390） |
| --- | --- | --- |
| submitted：已送审，待审核 | [截图](submitted-desktop-light.png) | [截图](submitted-narrow-dark.png) |
| rejected：已驳回 | [截图](rejected-desktop-light.png) | [截图](rejected-narrow-dark.png) |
| approved：审核通过，待上架 | [截图](approved-desktop-light.png) | [截图](approved-narrow-dark.png) |
| published：已公开上架 | [截图](published-desktop-light.png) | [截图](published-narrow-dark.png) |

另有已上架的[桌面深色](published-desktop-dark.png)与[窄屏浅色](published-narrow-light.png)。所有视图的手动主题均与系统主题相反；每次检查默认、选中、悬停、聚焦和禁用文字，最低对比度 5.91:1，无横向溢出，五项导航均可见。导航切换保留准备集合及所选任务。

## 验证方法与结果

1. 在 Market 根目录运行 `python3 scripts/publisher-joint-lab.py`，读取本次新 handoff；独立插件安装当前本地 SDK tarball，启动本地 Go 执行器与 Nuxt 开发页。
2. 经真实市场页面签发 M2 凭据，私密 stdin 绑定系统凭据库；开发页读取实际资料选项与完整集合、预览并明确确认送审。
3. 市场后台驳回，Framework 显式刷新并截图；修订时读取到的准备快照状态确实为 rejected，再经页面确认创建新任务与申请。
4. 新申请待审核、后台批准后待上架、后台显式上架后，分别刷新并验证集合卡片与当前申请状态一致。批准时公开目录 404，上架后 200。
5. 切换旧申请并只读刷新，集合卡片仍显示当前版本 published，旧申请仍 rejected；原资料、完整集合和回执逐字段不变，旧控制台链接仍打开原资料。
6. 撤销本次凭据，删除系统凭据条目；关闭浏览器、实验服务及子进程，确认所有 8 个端口已释放、隔离 schema/bucket 与两个临时根目录均删除。

两次申请各只有一次送审写入。发布模块 6 个文件、54 项测试通过，新增 6 项覆盖四态、待核对及连接／版本隔离；严格 `vue-tsc`（TypeScript 5.9.3）与模板同步检查通过。

验证结果见[独立验证记录](../plugin-publisher-review-status-validation-20261112.json)和[消费及源码摘要](consumer-provenance.json)。[SHA256SUMS](SHA256SUMS) 可校验原始截图、消费摘要和本索引。

本次环境已经关闭；记录中的地址和对象仅用于追溯本次实验，不能复用。
