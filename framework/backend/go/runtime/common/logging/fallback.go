package logging

import "strings"

const (
	FallbackUnknown      = "unknown"
	ReasonMissingContext = "missing_context"
)

func ApplyRuntimeFallback(fields Fields) Fields {
	out := MergeFields(Fields{}, fields)

	tenantUUID := trimString(out[FieldTenantUUID])
	if tenantUUID == "" {
		tenantUUID = FallbackUnknown
	}
	out[FieldTenantUUID] = tenantUUID
	out[FieldTenantKey] = TenantKeyFromUUID(tenantUUID)

	missing := false
	if trimString(out[FieldTaskID]) == "" {
		out[FieldTaskID] = FallbackUnknown
		missing = true
	}
	if trimString(out[FieldSubscriber]) == "" {
		out[FieldSubscriber] = FallbackUnknown
		missing = true
	}

	if missing {
		if strings.TrimSpace(trimString(out[FieldStatus])) == "" {
			out[FieldStatus] = StatusSkipped
		}
		if trimString(out[FieldReason]) == "" {
			out[FieldReason] = ReasonMissingContext
		}
	}

	return out
}
