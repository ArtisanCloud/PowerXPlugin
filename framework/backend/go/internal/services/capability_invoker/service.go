package capabilityinvoker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/internal/integration/gateway"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/observability"
)

// GatewayInvoker 定义 Gateway Client 最小调用能力，便于单测注入。
type GatewayInvoker interface {
	Invoke(ctx context.Context, req gateway.InvokeRequest) (*gateway.Response, error)
}

// Service 提供能力调用的统一入口。
type Service struct {
	gateway GatewayInvoker
	logger  *slog.Logger
}

// InvokeParams 描述一次能力调用输入。
type InvokeParams struct {
	CapabilityID      string
	Action            string
	PreferredProtocol string
	Payload           any
	Headers           map[string]string
	RequestID         string
	TenantUUID        string
}

// InvokeResult 返回 traceId 与解析后的数据。
type InvokeResult struct {
	TraceID string
	Status  string
	Data    map[string]any
	Raw     []byte
}

// ErrorCategory 用于标识错误类型，方便上层处理。
type ErrorCategory string

const (
	ErrorCategoryValidation   ErrorCategory = "validation"
	ErrorCategoryUnauthorized ErrorCategory = "unauthorized"
	ErrorCategoryRateLimited  ErrorCategory = "rate_limited"
	ErrorCategoryUpstream     ErrorCategory = "upstream"
)

// InvokeError 提供标准化错误对象。
type InvokeError struct {
	Category ErrorCategory
	TraceID  string
	Status   int
	Code     string
	Message  string
	Cause    error
}

func (e *InvokeError) Error() string {
	if e == nil {
		return "<nil>"
	}
	base := e.Message
	if base == "" && e.Cause != nil {
		base = e.Cause.Error()
	}
	return fmt.Sprintf("capability invoke failed (%s): %s", e.Category, base)
}

func (e *InvokeError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// NewService 构造默认 Service。
func NewService(gateway GatewayInvoker, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{gateway: gateway, logger: logger}
}

// Invoke 校验参数并调用 Gateway，返回统一结果。
func (s *Service) Invoke(ctx context.Context, params InvokeParams) (*InvokeResult, error) {
	if err := s.validate(params); err != nil {
		return nil, err
	}
	if s.gateway == nil {
		return nil, &InvokeError{Category: ErrorCategoryUpstream, Message: "gateway client not configured"}
	}
	start := time.Now()
	normalizedProtocol := normalizePreferredProtocol(params.PreferredProtocol, params.Payload)
	normalizedAction := normalizeAction(params.Action, normalizedProtocol, params.Payload)
	headers := copyHeaders(params.Headers)
	if tenant := strings.TrimSpace(params.TenantUUID); tenant != "" {
		if headers == nil {
			headers = make(map[string]string, 1)
		}
		if strings.TrimSpace(headers["tenant_uuid"]) == "" {
			headers["tenant_uuid"] = tenant
		}
	}
	resp, err := s.gateway.Invoke(ctx, gateway.InvokeRequest{
		CapabilityID:      params.CapabilityID,
		Action:            normalizedAction,
		PreferredProtocol: normalizedProtocol,
		Payload:           params.Payload,
		Headers:           headers,
		RequestID:         params.RequestID,
		TenantUUID:        params.TenantUUID,
	})
	if err != nil {
		duration := time.Since(start)
		invErr := s.toInvokeError(err)
		result := invocationResultFromCategory(invErr.Category)
		observability.ObserveCapabilityInvocation(params.CapabilityID, params.TenantUUID, result, duration)
		if invErr.Category == ErrorCategoryRateLimited {
			observability.IncrementCapabilityRateLimit(params.CapabilityID, params.TenantUUID)
		}
		paramsForLog := params
		paramsForLog.Action = normalizedAction
		paramsForLog.PreferredProtocol = normalizedProtocol
		s.logFailure(paramsForLog, invErr, duration)
		return nil, invErr
	}
	result := &InvokeResult{
		TraceID: resp.TraceID,
		Status:  resp.Status,
		Data:    resp.Data,
	}
	if resp.RawData != nil {
		result.Raw = resp.RawData
	}
	duration := time.Since(start)
	observability.ObserveCapabilityInvocation(params.CapabilityID, params.TenantUUID, observability.CapabilityInvocationResultSuccess, duration)
	paramsForLog := params
	paramsForLog.Action = normalizedAction
	paramsForLog.PreferredProtocol = normalizedProtocol
	s.logSuccess(paramsForLog, result.TraceID, result.Status, duration)
	return result, nil
}

func (s *Service) validate(params InvokeParams) error {
	if strings.TrimSpace(params.CapabilityID) == "" {
		return &InvokeError{Category: ErrorCategoryValidation, Message: "capabilityId is required"}
	}
	if params.Payload == nil {
		return &InvokeError{Category: ErrorCategoryValidation, Message: "payload is required"}
	}
	return nil
}

func (s *Service) toInvokeError(err error) *InvokeError {
	var invErr *gateway.InvocationError
	if errors.As(err, &invErr) {
		category := categorizeStatus(invErr.StatusCode)
		msg := ""
		code := ""
		if len(invErr.Errors) > 0 {
			code = invErr.Errors[0].Code
			msg = invErr.Errors[0].Message
		}
		if msg == "" {
			msg = err.Error()
		}
		return &InvokeError{
			Category: category,
			TraceID:  invErr.TraceID,
			Status:   invErr.StatusCode,
			Code:     code,
			Message:  msg,
			Cause:    err,
		}
	}
	return &InvokeError{
		Category: ErrorCategoryUpstream,
		Message:  err.Error(),
		Cause:    err,
	}
}

func categorizeStatus(status int) ErrorCategory {
	switch status {
	case http.StatusBadRequest:
		return ErrorCategoryValidation
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrorCategoryUnauthorized
	case http.StatusTooManyRequests:
		return ErrorCategoryRateLimited
	default:
		return ErrorCategoryUpstream
	}
}

func (s *Service) logSuccess(params InvokeParams, traceID, status string, duration time.Duration) {
	payloadMethod, payloadEndpoint := payloadRouteSummary(params.Payload)
	observability.EmitCapabilityTrace(observability.CapabilityInvocationTrace{
		Logger:            s.logger,
		CapabilityID:      params.CapabilityID,
		TenantUUID:        params.TenantUUID,
		Action:            params.Action,
		PreferredProtocol: params.PreferredProtocol,
		PayloadMethod:     payloadMethod,
		PayloadEndpoint:   payloadEndpoint,
		Status:            status,
		TraceID:           traceID,
		RequestID:         params.RequestID,
		Result:            observability.CapabilityInvocationResultSuccess,
		Duration:          duration,
	})
}

func (s *Service) logFailure(params InvokeParams, err *InvokeError, duration time.Duration) {
	if err == nil {
		return
	}
	result := invocationResultFromCategory(err.Category)
	payloadMethod, payloadEndpoint := payloadRouteSummary(params.Payload)
	observability.EmitCapabilityTrace(observability.CapabilityInvocationTrace{
		Logger:            s.logger,
		CapabilityID:      params.CapabilityID,
		TenantUUID:        params.TenantUUID,
		Action:            params.Action,
		PreferredProtocol: params.PreferredProtocol,
		PayloadMethod:     payloadMethod,
		PayloadEndpoint:   payloadEndpoint,
		TraceID:           err.TraceID,
		RequestID:         params.RequestID,
		Result:            result,
		Duration:          duration,
		ErrorCode:         err.Code,
		ErrorMessage:      err.Message,
		StatusCode:        err.Status,
	})

	if err.Category == ErrorCategoryRateLimited || err.Category == ErrorCategoryUnauthorized {
		s.logAuditDenial(params, err)
	}
}

func payloadRouteSummary(payload any) (string, string) {
	p, ok := payload.(map[string]any)
	if !ok || p == nil {
		return "", ""
	}
	method := strings.TrimSpace(strings.ToUpper(fmt.Sprint(p["method"])))
	endpoint := strings.TrimSpace(fmt.Sprint(p["endpoint"]))
	return method, endpoint
}

func normalizePreferredProtocol(preferred string, payload any) string {
	if hasRESTRoutePayload(payload) {
		return "rest"
	}
	if hasGRPCRoutePayload(payload) {
		return "grpc"
	}
	return strings.TrimSpace(preferred)
}

func normalizeAction(action, preferred string, payload any) string {
	if strings.EqualFold(strings.TrimSpace(preferred), "rest") && hasRESTRoutePayload(payload) {
		return ""
	}
	return strings.TrimSpace(action)
}

func hasRESTRoutePayload(payload any) bool {
	p, ok := payload.(map[string]any)
	if !ok || p == nil {
		return false
	}
	method := strings.TrimSpace(strings.ToUpper(fmt.Sprint(p["method"])))
	endpoint := strings.TrimSpace(fmt.Sprint(p["endpoint"]))
	return method != "" && endpoint != ""
}

func hasGRPCRoutePayload(payload any) bool {
	p, ok := payload.(map[string]any)
	if !ok || p == nil {
		return false
	}
	endpoint := strings.TrimSpace(fmt.Sprint(p["endpoint"]))
	rpc := strings.TrimSpace(fmt.Sprint(p["rpc"]))
	return endpoint != "" && rpc != ""
}

func (s *Service) logAuditDenial(params InvokeParams, err *InvokeError) {
	logger := s.logger
	if logger == nil {
		logger = slog.Default()
	}
	fields := []any{
		slog.String("capabilityId", params.CapabilityID),
	}
	if params.TenantUUID != "" {
		fields = append(fields, slog.String("tenantUUID", params.TenantUUID))
	}
	if params.Action != "" {
		fields = append(fields, slog.String("action", params.Action))
	}
	if params.PreferredProtocol != "" {
		fields = append(fields, slog.String("preferredProtocol", params.PreferredProtocol))
	}
	if err.TraceID != "" {
		fields = append(fields, slog.String("traceId", err.TraceID))
	}
	if params.RequestID != "" {
		fields = append(fields, slog.String("requestId", params.RequestID))
	}
	if err.Status > 0 {
		fields = append(fields, slog.Int("statusCode", err.Status))
	}
	if err.Code != "" {
		fields = append(fields, slog.String("code", err.Code))
	}
	if err.Message != "" {
		fields = append(fields, slog.String("message", err.Message))
	}
	logger.Warn("audit.capability.invocation.denied", fields...)
}

func invocationResultFromCategory(category ErrorCategory) observability.CapabilityInvocationResult {
	switch category {
	case ErrorCategoryValidation:
		return observability.CapabilityInvocationResultValidation
	case ErrorCategoryUnauthorized:
		return observability.CapabilityInvocationResultUnauthorized
	case ErrorCategoryRateLimited:
		return observability.CapabilityInvocationResultRateLimited
	default:
		return observability.CapabilityInvocationResultUpstream
	}
}

func copyHeaders(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		if v == "" {
			continue
		}
		dst[k] = v
	}
	return dst
}
