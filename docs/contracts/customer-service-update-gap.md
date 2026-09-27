# 客户资料更新服务合同对齐

## 当前接入

Framework Lab `/admin/templates/framework-lab`：Local 编辑使用插件本地客户表；Delegated 编辑通过 `AccountSelectorClient.UpdateBasicAccount` 调用 `com.corex.customer.accounts.service_manage`，固定 `INVOKE core://customer/accounts`、`operation=update`，不调用 Admin REST。

输入为 `customer_uuid` 与可选 `display_name`、`nickname`、`given_name`、`family_name`、`primary_email`、`primary_phone`、`avatar_url`、`locale`、`timezone`、`status`。Framework DTO 使用字符串指针：未传保留，显式空字符串清空可选字段。禁止 null、未知字段、类型与主要联系人变更、调用方租户或 actor 覆盖。名称和状态不能清空，格式校验以 Core 合同为准。

返回解码 `data.payload.item`，将 Core `uuid` 映射为 `customer_uuid`；可选字段缺失按空值处理。页面保存后重新加载列表，不合并旧资料。编辑状态包括 active/pending/suspended/disabled/expired/deleted。

## Core 行为边界

Core 正式合同见 PowerX 仓库 `docs/contracts/customer-service-update.md`。当前 Core 更新还会修复历史空类型和主要联系人，可能创建联系人或增加 primary 角色；这超出原先“资料更新保持 type/primary_contact_uuid 不变”的要求，需 Core 单独确认或拆出修复操作。本插件不实现额外身份或联系人修复。

客户资料更新不等于变更登录身份。联系人角色 primary 也不等于切换主要联系人。客户状态 deleted 是 Core 状态值，不表示执行软删除。

## 验收边界

- 聚焦测试覆盖固定 service capability、字段 presence、清空、非法字段拒绝以及 Local/Delegated 路由隔离。
- 插件后台由操作者热更新或重启，不由本次任务启动。
- 真实 API Key 更新、撤销 grant、跨租户拒绝以及安装后的 STS/页面验收仍需运行中的最新 Core 与插件完成；源码测试不作为实际安装联调证明。
