package customerfw

import "time"

type AuditFields map[string]any

func ValidationAuditFields(tenantUUID, customerUUID string, source CustomerAuthSource, ok bool, latency time.Duration, err error) AuditFields {
	fields := AuditFields{
		"tenant_uuid":   normalizeID(tenantUUID),
		"customer_uuid": normalizeID(customerUUID),
		"source":        string(NormalizeSource(source)),
		"ok":            ok,
		"latency_ms":    latency.Milliseconds(),
	}
	if err != nil {
		fields["error_code"] = string(CodeOf(err))
	}
	return fields
}

func SourceDiagnostics(source CustomerAuthSource) AuditFields {
	return AuditFields{"source": string(NormalizeSource(source))}
}
