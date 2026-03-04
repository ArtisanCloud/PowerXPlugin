# Event Fabric 命名规范（插件侧）

> 对齐 PowerX：`topic / subscriber / kind` 三类语义分离。

## 1. 前缀规范

1. Topic：`_topic.<domain>.<name>`
2. Subscriber：`_subscriber.<domain>.<handler>`
3. Kind：`_kind.<domain>.<action>`

## 2. Topic 规范

1. 逻辑 topic：`_topic.<domain>.<name>`
2. 插件侧默认不手写 `full_topic`，由运行时根据模式处理作用域
3. 示例：
   - `_topic.template.update`
   - `_topic.system.notification`

## 3. Subscriber 规范

1. 结构：`_subscriber.<domain>.<handler>`
2. 示例：
   - `_subscriber.event_fabric.replay`
   - `_subscriber.system.notification_dispatch`

## 4. Kind 规范

1. 结构：`_kind.<domain>.<action>`
2. 示例：
   - `_kind.event_fabric.replay.task`

## 5. 插件落地约束

1. `skeleton/plugin.yaml` 的 `events.publish` 仅声明 `_topic.*`
2. WS `subscribe.topics` 仅允许 `_topic.*`
3. 指标/日志中的 `topic` 必须与事件 topic 完全一致，不做别名映射
4. API Key 权限通过底座 profile `permission_ids` 绑定，topic 命名必须可被 ACL 精确匹配
