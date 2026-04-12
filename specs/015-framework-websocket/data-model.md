# Data Model: Framework WS Bus Adapter

## Entities

### PublishRequest

- **topic**: string (topic 名称，支持兼容别名)
- **payload**: object (业务载荷)
- **tenant_uuid**: string (租户 UUID)
- **trace_id**: string (可选)

### PublishResult

- **ok**: boolean
- **error**: string (可选)

### TopicWhitelist

- **allowed_topics**: string[]
- **aliases**: map (旧 topic → 新 topic)

## Relationships

- PublishRequest.topic 必须在 TopicWhitelist 中匹配。
- PublishRequest.tenant_uuid 必须与授权上下文一致。

## Validation Rules

- topic 不能为空且必须匹配白名单。
- tenant_uuid 必须存在且为有效 UUID。
- payload 必须可序列化为 JSON。
