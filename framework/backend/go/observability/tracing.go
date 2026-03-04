package observability

import (
	"log/slog"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
)

// InitTracing 预留链路追踪初始化入口。
func InitTracing(app *bootstrap.App) error {
	// TODO: 接入 OpenTelemetry / Jaeger
	_ = app
	return nil
}

// CapabilityInvocationResult 表示一次能力调用的结果分类。
type CapabilityInvocationResult string

const (
	CapabilityInvocationResultSuccess      CapabilityInvocationResult = "success"
	CapabilityInvocationResultValidation   CapabilityInvocationResult = "validation"
	CapabilityInvocationResultUnauthorized CapabilityInvocationResult = "unauthorized"
	CapabilityInvocationResultRateLimited  CapabilityInvocationResult = "rate_limited"
	CapabilityInvocationResultUpstream     CapabilityInvocationResult = "upstream"
)

// CapabilityInvocationTrace 记录能力调用的追踪字段。
type CapabilityInvocationTrace struct {
	Logger            *slog.Logger
	CapabilityID      string
	TenantUUID        string
	Action            string
	PreferredProtocol string
	PayloadMethod     string
	PayloadEndpoint   string
	Status            string
	TraceID           string
	RequestID         string
	Result            CapabilityInvocationResult
	Duration          time.Duration
	ErrorCode         string
	ErrorMessage      string
	StatusCode        int
}

// EmitCapabilityTrace 输出标准化的能力调用日志事件。
func EmitCapabilityTrace(trace CapabilityInvocationTrace) {
	logger := trace.Logger
	if logger == nil {
		logger = slog.Default()
	}
	fields := []any{
		slog.String("capabilityId", trace.CapabilityID),
	}
	if trace.TenantUUID != "" {
		fields = append(fields, slog.String("tenantUUID", trace.TenantUUID))
	}
	if trace.Action != "" {
		fields = append(fields, slog.String("action", trace.Action))
	}
	if trace.PreferredProtocol != "" {
		fields = append(fields, slog.String("preferredProtocol", trace.PreferredProtocol))
	}
	if trace.PayloadMethod != "" {
		fields = append(fields, slog.String("payloadMethod", trace.PayloadMethod))
	}
	if trace.PayloadEndpoint != "" {
		fields = append(fields, slog.String("payloadEndpoint", trace.PayloadEndpoint))
	}
	if trace.Status != "" {
		fields = append(fields, slog.String("status", trace.Status))
	}
	if trace.TraceID != "" {
		fields = append(fields, slog.String("traceId", trace.TraceID))
	}
	if trace.RequestID != "" {
		fields = append(fields, slog.String("requestId", trace.RequestID))
	}
	if trace.Duration > 0 {
		fields = append(fields, slog.Float64("durationMs", float64(trace.Duration.Microseconds())/1000.0))
	}
	if trace.ErrorCode != "" {
		fields = append(fields, slog.String("code", trace.ErrorCode))
	}
	if trace.ErrorMessage != "" {
		fields = append(fields, slog.String("message", trace.ErrorMessage))
	}
	if trace.StatusCode > 0 {
		fields = append(fields, slog.Int("statusCode", trace.StatusCode))
	}

	switch trace.Result {
	case CapabilityInvocationResultSuccess:
		logger.Info("capability.invoke.success", fields...)
	case CapabilityInvocationResultRateLimited:
		logger.Warn("capability.invoke.rate_limit", fields...)
	case CapabilityInvocationResultValidation:
		logger.Warn("capability.invoke.validation_failed", fields...)
	case CapabilityInvocationResultUnauthorized:
		logger.Warn("capability.invoke.unauthorized", fields...)
	default:
		logger.Error("capability.invoke.failure", fields...)
	}
}
