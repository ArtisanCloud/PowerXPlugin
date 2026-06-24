# Data Model: Framework Realtime Transport

## RealtimeScope

- `tenant_uuid` (string, required for tenant-scoped events)
- `member_uuid` (string, optional, required for member-scoped events)
- `user_uuid` (string, optional)
- `plugin_id` (string, required)
- `trace_id` (string, required for publish/stream)
- `request_id` (string, optional)

## RealtimeDescriptor

- `key` (string): declared topic/channel key
- `protocols` (array): `ws`, `sse`
- `actions` (array): `publish`, `subscribe`
- `scope` (string): `global`, `tenant`, `member`
- `event_types` (array): allowed event types
- `description` (string)

## RealtimeEnvelope

- `protocol` (string): `ws` or `sse`
- `topic` (string, optional for WS)
- `channel` (string, optional for SSE)
- `event_type` (string)
- `payload` (object)
- `tenant_uuid` (string)
- `member_uuid` (string)
- `plugin_id` (string)
- `trace_id` (string)
- `request_id` (string)
- `timestamp` (string, RFC3339)

## RealtimeConnectionState

- `status` (string): `idle`, `connecting`, `connected`, `reconnecting`, `error`, `closed`
- `protocol` (string)
- `url` (string, sanitized in diagnostics)
- `url_source` (string): `standalone`, `host`, `proxy`, `custom`
- `tenant_uuid` (string)
- `member_uuid` (string)
- `subscriptions` (array)
- `reconnect_attempts` (integer)
- `last_event_type` (string)
- `last_trace_id` (string)
- `last_error` (object)

## StreamThroughSession

- `session_id` (string)
- `source_url` (string, sanitized)
- `target_protocol` (string): `sse`
- `preserve_event_names` (bool)
- `headers_policy` (string): `forward_safe`, `framework_auth`, `none`
- `trace_id` (string)
- `started_at` (string)
- `ended_at` (string)

## RealtimePermissionDecision

- `allowed` (bool)
- `action` (string): `publish`, `subscribe`
- `key` (string)
- `reason` (string)
- `resource` (string)
- `trace_id` (string)

## Relationships

- `RealtimeDescriptor` is sourced from `plugin.d/events.yaml`.
- `RealtimeEnvelope` must reference one declared descriptor unless it is a stream-through raw Agent event.
- `RealtimeConnectionState` belongs to a frontend client instance and must be reset when `RealtimeScope` changes.
- `StreamThroughSession` is used by Agent SSE proxy and must preserve upstream event semantics.
