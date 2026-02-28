# ACL / Security（插件侧）

## 1. 范围

1. topic 动作授权（publish / subscribe）
2. 凭证分流（Bearer / ApiKey）
3. proxy 场景租户边界（由底座解析）

## 2. 核心原则

1. 插件侧事件白名单来自 `skeleton/plugin.yaml`
2. 底座侧授权来自 topic ACL + profile `permission_ids`
3. API Key 权限是快照，权限变更后需轮换/重建 key

## 3. 最小排障顺序

1. 查 `plugin.yaml` 是否允许该 `_topic.*`
2. 查底座 ACL / profile 是否允许该动作
3. 查插件日志是否为 `gateway_auth_scheme=apikey`（proxy 场景）
4. 查订阅 topic 是否与发布 topic 完全一致

## 4. Standalone+Proxy 二段校验（必过）

1. 资源存在性：topic 必须已创建到 `event_topics`
2. 访问许可：API Key 快照权限必须允许该 topic

结论：`grant` 只做授权绑定，不创建 topic 资源本体。
