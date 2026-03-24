package logging

import "strings"

const (
	FieldTraceID     = "trace_id"
	FieldTaskID      = "task_id"
	FieldTenantUUID  = "tenant_uuid"
	FieldTenantKey   = "tenant_key"
	FieldSubscriber  = "subscriber_id"
	FieldTopic       = "topic"
	FieldStatus      = "status"
	FieldReason      = "reason"
	FieldPluginID    = "plugin_id"
	FieldComponent   = "component"
	FieldGatewayAuth = "gateway_auth_scheme"
	FieldTokenSource = "outbound_token_source"
)

const (
	StatusQueued     = "queued"
	StatusProcessing = "processing"
	StatusSucceeded  = "succeeded"
	StatusFailed     = "failed"
	StatusSkipped    = "skipped"
)

var RequiredFields = []string{
	FieldTraceID,
	FieldTaskID,
	FieldTenantUUID,
	FieldTenantKey,
	FieldSubscriber,
	FieldTopic,
	FieldStatus,
}

var ExtensionFields = []string{
	FieldGatewayAuth,
	FieldTokenSource,
	FieldPluginID,
	FieldComponent,
}

var StatusEnum = []string{
	StatusQueued,
	StatusProcessing,
	StatusSucceeded,
	StatusFailed,
	StatusSkipped,
}

var allowedStatusSet = map[string]struct{}{
	StatusQueued:     {},
	StatusProcessing: {},
	StatusSucceeded:  {},
	StatusFailed:     {},
	StatusSkipped:    {},
}

func IsAllowedStatus(status string) bool {
	normalized := strings.ToLower(strings.TrimSpace(status))
	_, ok := allowedStatusSet[normalized]
	return ok
}

func TenantKeyFromUUID(tenantUUID string) string {
	return strings.TrimSpace(tenantUUID)
}
