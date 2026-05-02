package customer

import (
	"context"
	"strings"
	"time"

	pxlog "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
)

// AuditLogger 负责输出 customer 鉴权关键链路的结构化日志。
// 约束：字段必须包含 tenant_uuid/customer_uuid/request_id（如可得）。
type AuditLogger struct {
	requestID string
}

func NewAuditLogger(logger *pxlog.Entry) *AuditLogger {
	_ = logger
	return &AuditLogger{}
}

func (l *AuditLogger) WithRequest(requestID string) *AuditLogger {
	if l == nil {
		return NewAuditLogger(nil).WithRequest(requestID)
	}
	return &AuditLogger{requestID: strings.TrimSpace(requestID)}
}

func (l *AuditLogger) LogValidation(tenantUUID, customerUUID, mode string, ok bool, latency time.Duration, err error) {
	fields := map[string]interface{}{
		"module":        "customer",
		"component":     "customer.audit_logger",
		"event":         "customer.auth.validation",
		"tenant_uuid":   strings.ToLower(strings.TrimSpace(tenantUUID)),
		"customer_uuid": strings.ToLower(strings.TrimSpace(customerUUID)),
		"mode":          strings.ToLower(strings.TrimSpace(mode)),
		"ok":            ok,
		"latency_ms":    latency.Milliseconds(),
		"biz_scene":     "customer_auth_validation",
		"biz_domain":    "customer",
	}
	if l != nil && strings.TrimSpace(l.requestID) != "" {
		fields["request_id"] = strings.TrimSpace(l.requestID)
		fields["trace_id"] = strings.TrimSpace(l.requestID)
	}
	if err != nil {
		fields["error"] = err.Error()
		ctx := pxlog.WithLogFields(context.Background(), fields)
		pxlog.WarnCtx(ctx, "customer auth validation failed")
		return
	}
	pxlog.InfoCtx(pxlog.WithLogFields(context.Background(), fields), "customer auth validation ok")
}

func (l *AuditLogger) LogLogin(tenantUUID, customerUUID string, ok bool, latency time.Duration, err error) {
	fields := map[string]interface{}{
		"module":        "customer",
		"component":     "customer.audit_logger",
		"event":         "customer.auth.login",
		"tenant_uuid":   strings.ToLower(strings.TrimSpace(tenantUUID)),
		"customer_uuid": strings.ToLower(strings.TrimSpace(customerUUID)),
		"ok":            ok,
		"latency_ms":    latency.Milliseconds(),
		"biz_scene":     "customer_auth_login",
		"biz_domain":    "customer",
	}
	if l != nil && strings.TrimSpace(l.requestID) != "" {
		fields["request_id"] = strings.TrimSpace(l.requestID)
		fields["trace_id"] = strings.TrimSpace(l.requestID)
	}
	if err != nil {
		fields["error"] = err.Error()
		pxlog.WarnCtx(pxlog.WithLogFields(context.Background(), fields), "customer login failed")
		return
	}
	pxlog.InfoCtx(pxlog.WithLogFields(context.Background(), fields), "customer login ok")
}

func (l *AuditLogger) LogRegister(tenantUUID, customerUUID string, ok bool, latency time.Duration, err error) {
	fields := map[string]interface{}{
		"module":        "customer",
		"component":     "customer.audit_logger",
		"event":         "customer.auth.register",
		"tenant_uuid":   strings.ToLower(strings.TrimSpace(tenantUUID)),
		"customer_uuid": strings.ToLower(strings.TrimSpace(customerUUID)),
		"ok":            ok,
		"latency_ms":    latency.Milliseconds(),
		"biz_scene":     "customer_auth_register",
		"biz_domain":    "customer",
	}
	if l != nil && strings.TrimSpace(l.requestID) != "" {
		fields["request_id"] = strings.TrimSpace(l.requestID)
		fields["trace_id"] = strings.TrimSpace(l.requestID)
	}
	if err != nil {
		fields["error"] = err.Error()
		pxlog.WarnCtx(pxlog.WithLogFields(context.Background(), fields), "customer register failed")
		return
	}
	pxlog.InfoCtx(pxlog.WithLogFields(context.Background(), fields), "customer register ok")
}
