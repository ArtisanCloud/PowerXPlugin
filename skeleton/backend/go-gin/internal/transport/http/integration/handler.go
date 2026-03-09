package integration

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	frameworkgateway "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	capgateway "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/integrations/gateway"
	integrationService "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/integration"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Handler 提供 integration HTTP API 的入口。
type Handler struct {
	deps     *app.Deps
	dispatch *integrationService.DispatchService
	logger   *logrus.Entry
}

type capabilityInvokeRequest struct {
	CapabilityID      string                 `json:"capabilityId"`
	Action            string                 `json:"action"`
	PreferredProtocol string                 `json:"preferredProtocol"`
	Payload           map[string]any         `json:"payload"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// NewHandler 构造新的 Handler。
func NewHandler(deps *app.Deps) *Handler {
	var logger *logrus.Entry
	if deps != nil {
		logger = deps.RuntimeLogger(deps.Ctx, "integration_http", nil)
	}
	h := &Handler{
		deps:   deps,
		logger: logger,
	}
	h.dispatch = h.buildDispatchService()
	return h
}

func (h *Handler) buildDispatchService() *integrationService.DispatchService {
	if h.deps == nil {
		return nil
	}

	logger := h.logger
	if logger == nil {
		logger = logrus.WithField("component", "integration_http")
	}
	service := integrationService.BuildDispatchService(h.deps, logger)
	if service == nil {
		return nil
	}
	return service
}

// ListGrantMatrix 返回当前 GrantMatrix 视图。
func (h *Handler) ListGrantMatrix(c *gin.Context) {
	respondPlaceholder(c, http.StatusOK, "grant matrix listing not implemented")
}

// SubmitGrantMatrix 接收数据库覆盖项。
func (h *Handler) SubmitGrantMatrix(c *gin.Context) {
	respondPlaceholder(c, http.StatusAccepted, "grant matrix override submission pending approval workflow")
}

// CreateSubscription 注册 webhook 订阅。
func (h *Handler) CreateSubscription(c *gin.Context) {
	respondPlaceholder(c, http.StatusCreated, "webhook subscription endpoint not implemented")
}

// ListSubscriptions 查询 webhook 订阅。
func (h *Handler) ListSubscriptions(c *gin.Context) {
	respondPlaceholder(c, http.StatusOK, "webhook subscription list not implemented")
}

// ReplayDLQ 触发 DLQ 补发。
func (h *Handler) ReplayDLQ(c *gin.Context) {
	respondPlaceholder(c, http.StatusAccepted, "webhook DLQ replay not implemented")
}

// CreateSecret 注册外部凭证。
func (h *Handler) CreateSecret(c *gin.Context) {
	respondPlaceholder(c, http.StatusCreated, "secret lifecycle endpoint not implemented")
}

// RotateSecret 触发凭证轮换。
func (h *Handler) RotateSecret(c *gin.Context) {
	respondPlaceholder(c, http.StatusAccepted, "secret rotation workflow not implemented")
}

// InvokeCapability 代理插件侧前端对 PowerX Gateway 的能力调用。
func (h *Handler) InvokeCapability(c *gin.Context) {
	ensureInvokeCORS(c)

	if h == nil || h.deps == nil || h.deps.CapabilityGateway == nil {
		contracts.ResponseServiceUnavailable(c, "capability gateway unavailable", nil)
		return
	}

	var req capabilityInvokeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid capability payload")
		return
	}

	capabilityID := strings.TrimSpace(req.CapabilityID)
	action := strings.TrimSpace(req.Action)
	if capabilityID == "" {
		contracts.ResponseBadRequest(c, "capabilityId is required")
		return
	}

	payload := ensurePayload(req.Payload)
	preferredProtocol := strings.TrimSpace(req.PreferredProtocol)
	if preferredProtocol == "" {
		preferredProtocol = inferPreferredProtocol(payload)
	}
	if isGRPCPayload(payload) && !isRESTPayload(payload) {
		preferredProtocol = "grpc"
	}
	if isRESTPayload(payload) {
		preferredProtocol = "rest"
	}
	if preferredProtocol == "" {
		preferredProtocol = "rest"
	}
	if strings.EqualFold(preferredProtocol, "rest") {
		if err := validateRestPayload(payload); err != nil {
			contracts.ResponseBadRequest(c, err.Error())
			return
		}
	}
	actionForInvoke := action

	headers := collectCapabilityHeaders(c)
	warnings := collectCapabilityWarnings(headers)
	requestID := strings.TrimSpace(c.GetHeader("X-Request-ID"))
	h.logCapabilityInvokeAttempt(capabilityID, action, preferredProtocol, requestID, payload, headers)

	result, err := h.deps.CapabilityGateway.Invoke(c.Request.Context(), capgateway.InvokeParams{
		CapabilityID:      capabilityID,
		Action:            actionForInvoke,
		PreferredProtocol: preferredProtocol,
		Payload:           payload,
		Headers:           headers,
		RequestID:         requestID,
	})
	if err != nil {
		h.logCapabilityInvokeError(capabilityID, action, preferredProtocol, requestID, err)
		h.writeCapabilityError(c, err, warnings)
		return
	}

	if result == nil {
		contracts.ResponseError(c, http.StatusBadGateway, contracts.ErrCodeInternalError, "capability invoke returned no result")
		return
	}

	if result.TraceID != "" {
		c.Header("X-Trace-Id", result.TraceID)
	}
	if result.Mock {
		msg := result.MockReason
		if strings.TrimSpace(msg) == "" {
			msg = "PX_USE_MOCK 生效，返回模拟数据"
		}
		warnings = append(warnings, msg)
	}

	response := gin.H{
		"traceId": result.TraceID,
		"status":  result.Status,
		"data":    result.Data,
	}
	if len(result.Raw) > 0 {
		response["raw"] = json.RawMessage(result.Raw)
	}
	if len(warnings) > 0 {
		response["warnings"] = warnings
	}
	h.logCapabilityInvokeSuccess(capabilityID, action, preferredProtocol, requestID, result)
	h.logInvokeCORSHeaders(c)

	contracts.ResponseSuccess(c, response)
}

func ensureInvokeCORS(c *gin.Context) {
	if c == nil {
		return
	}
	origin := strings.TrimSpace(c.GetHeader("Origin"))
	allowOrigin := origin
	allowCredentials := "true"
	if allowOrigin == "" {
		allowOrigin = "*"
		allowCredentials = "false"
	}
	c.Header("Access-Control-Allow-Origin", allowOrigin)
	c.Header("Vary", "Origin")
	c.Header("Access-Control-Allow-Credentials", allowCredentials)
	c.Header("Access-Control-Allow-Methods", "POST, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
	c.Header("Access-Control-Expose-Headers", "X-Trace-Id, X-Correlation-Id, X-Request-Id")
}

func (h *Handler) logCapabilityInvokeAttempt(capabilityID, action, preferredProtocol, requestID string, payload map[string]any, headers map[string]string) {
	if h == nil || h.logger == nil {
		return
	}
	inboundAuth := strings.TrimSpace(headers["Authorization"])
	inboundAuthScheme := authSchemeFromHeader(inboundAuth)
	h.logger.WithFields(logrus.Fields{
		"capability_id":       capabilityID,
		"action":              action,
		"preferred_protocol":  preferredProtocol,
		"request_id":          requestID,
		"payload_method":      strings.TrimSpace(strings.ToUpper(fmt.Sprint(payload["method"]))),
		"payload_endpoint":    strings.TrimSpace(fmt.Sprint(payload["endpoint"])),
		"payload_query_count": len(extractPayloadMap(payload, "query")),
		"headers_count":       len(headers),
		"inbound_auth_scheme": inboundAuthScheme,
		"forwarded_auth":      inboundAuth != "",
	}).Info("capability invoke request accepted")
}

func (h *Handler) logCapabilityInvokeSuccess(capabilityID, action, preferredProtocol, requestID string, result *capgateway.InvokeResult) {
	if h == nil || h.logger == nil || result == nil {
		return
	}
	h.logger.WithFields(logrus.Fields{
		"capability_id":      capabilityID,
		"action":             action,
		"preferred_protocol": preferredProtocol,
		"request_id":         requestID,
		"trace_id":           strings.TrimSpace(result.TraceID),
		"status":             strings.TrimSpace(result.Status),
		"mock":               result.Mock,
	}).Info("capability invoke success")
}

func (h *Handler) logCapabilityInvokeError(capabilityID, action, preferredProtocol, requestID string, err error) {
	if h == nil || h.logger == nil || err == nil {
		return
	}
	fields := logrus.Fields{
		"capability_id":      capabilityID,
		"action":             action,
		"preferred_protocol": preferredProtocol,
		"request_id":         requestID,
	}
	var invocationErr *frameworkgateway.InvocationError
	if errors.As(err, &invocationErr) {
		fields["trace_id"] = strings.TrimSpace(invocationErr.TraceID)
		fields["status_code"] = invocationErr.StatusCode
	}
	h.logger.WithFields(fields).WithError(err).Error("capability invoke failed")
}

func extractPayloadMap(payload map[string]any, key string) map[string]any {
	if payload == nil {
		return nil
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return nil
	}
	if result, ok := raw.(map[string]any); ok {
		return result
	}
	return nil
}

func authSchemeFromHeader(authHeader string) string {
	auth := strings.TrimSpace(authHeader)
	if auth == "" {
		return "none"
	}
	lowered := strings.ToLower(auth)
	switch {
	case strings.HasPrefix(lowered, "bearer "):
		return "bearer"
	case strings.HasPrefix(lowered, "apikey "), strings.HasPrefix(lowered, "api_key "), strings.HasPrefix(lowered, "api-key "):
		return "apikey"
	default:
		return "unknown"
	}
}

func shouldOmitActionForREST(preferredProtocol string, payload map[string]any) bool {
	if strings.EqualFold(strings.TrimSpace(preferredProtocol), "rest") {
		return isRESTPayload(payload)
	}
	return isRESTPayload(payload)
}

func isRESTPayload(payload map[string]any) bool {
	method := strings.TrimSpace(strings.ToUpper(fmt.Sprint(payload["method"])))
	endpoint := strings.TrimSpace(fmt.Sprint(payload["endpoint"]))
	return method != "" && endpoint != ""
}

func isGRPCPayload(payload map[string]any) bool {
	endpoint := strings.TrimSpace(fmt.Sprint(payload["endpoint"]))
	rpc := strings.TrimSpace(fmt.Sprint(payload["rpc"]))
	return endpoint != "" && rpc != ""
}

func respondPlaceholder(c *gin.Context, status int, message string) {
	ensureInvokeCORS(c)
	switch status {
	case http.StatusCreated:
		contracts.ResponseCreated(c, gin.H{"status": "pending", "message": message})
	default:
		contracts.ResponseSuccess(c, gin.H{"status": "pending", "message": message})
	}
}

func collectCapabilityHeaders(c *gin.Context) map[string]string {
	if c == nil {
		return nil
	}
	forward := []string{"X-PX-Use-Mock", "Authorization"}
	headers := make(map[string]string)
	for _, name := range forward {
		if value := strings.TrimSpace(c.GetHeader(name)); value != "" {
			if strings.EqualFold(name, "Authorization") {
				switch authSchemeFromHeader(value) {
				case "apikey", "bearer":
					headers[name] = value
				}
				continue
			}
			headers[name] = value
		}
	}
	if len(headers) == 0 {
		return nil
	}
	return headers
}

func collectCapabilityWarnings(headers map[string]string) []string {
	if len(headers) == 0 {
		return nil
	}
	module := strings.TrimSpace(headers["X-PX-Use-Mock"])
	if module == "" {
		return nil
	}
	return []string{fmt.Sprintf("通过 X-PX-Use-Mock 请求 Mock 模块: %s", module)}
}

func ensurePayload(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}

func inferPreferredProtocol(payload map[string]any) string {
	if len(payload) == 0 {
		return ""
	}
	if method := strings.TrimSpace(strings.ToUpper(fmt.Sprint(payload["method"]))); method != "" {
		return "rest"
	}
	if proto := strings.TrimSpace(fmt.Sprint(payload["preferred_protocol"])); proto != "" {
		return strings.ToLower(proto)
	}
	if proto := strings.TrimSpace(fmt.Sprint(payload["preferredProtocol"])); proto != "" {
		return strings.ToLower(proto)
	}
	return ""
}

func validateRestPayload(payload map[string]any) error {
	method := strings.TrimSpace(strings.ToUpper(fmt.Sprint(payload["method"])))
	endpoint := strings.TrimSpace(fmt.Sprint(payload["endpoint"]))
	if method == "" {
		return errors.New("REST 协议需要提供 method 字段")
	}
	if endpoint == "" {
		return errors.New("REST 协议需要提供 endpoint 字段")
	}
	return nil
}

func (h *Handler) writeCapabilityError(c *gin.Context, err error, warnings []string) {
	if c == nil {
		return
	}
	ensureInvokeCORS(c)

	var unavailable *capgateway.UnavailableError
	if errors.As(err, &unavailable) {
		details := gin.H{"traceId": ""}
		if len(warnings) > 0 {
			details["warnings"] = warnings
		}
		h.logInvokeCORSHeaders(c)
		contracts.ResponseErrorWithDetails(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", unavailable.Error(), details)
		return
	}

	var invocationErr *frameworkgateway.InvocationError
	if errors.As(err, &invocationErr) {
		if trace := strings.TrimSpace(invocationErr.TraceID); trace != "" {
			c.Header("X-Trace-Id", trace)
		}
		status := invocationErr.StatusCode
		if status <= 0 {
			status = http.StatusBadGateway
		}
		errPayload := gin.H{
			"code":    firstGatewayCode(invocationErr),
			"message": firstGatewayMessage(invocationErr),
			"type":    categorizeGatewayError(status),
		}
		if body := parseGatewayBody(invocationErr.Body); body != nil {
			errPayload["details"] = body
		}
		details := gin.H{
			"traceId": invocationErr.TraceID,
			"error":   errPayload,
		}
		if len(warnings) > 0 {
			details["warnings"] = warnings
		}
		h.logInvokeCORSHeaders(c)
		contracts.ResponseErrorWithDetails(c, status, firstGatewayCode(invocationErr), firstGatewayMessage(invocationErr), details)
		return
	}

	details := gin.H{"traceId": ""}
	if len(warnings) > 0 {
		details["warnings"] = warnings
	}
	h.logInvokeCORSHeaders(c)
	contracts.ResponseErrorWithDetails(c, http.StatusBadGateway, contracts.ErrCodeInternalError, err.Error(), details)
}

func (h *Handler) logInvokeCORSHeaders(c *gin.Context) {
	if h == nil || h.logger == nil || c == nil {
		return
	}
	h.logger.WithFields(logrus.Fields{
		"origin":                         strings.TrimSpace(c.GetHeader("Origin")),
		"cors_allow_origin":              strings.TrimSpace(c.Writer.Header().Get("Access-Control-Allow-Origin")),
		"cors_allow_credentials":         strings.TrimSpace(c.Writer.Header().Get("Access-Control-Allow-Credentials")),
		"cors_expose_headers":            strings.TrimSpace(c.Writer.Header().Get("Access-Control-Expose-Headers")),
		"cors_allow_headers":             strings.TrimSpace(c.Writer.Header().Get("Access-Control-Allow-Headers")),
		"cors_allow_methods":             strings.TrimSpace(c.Writer.Header().Get("Access-Control-Allow-Methods")),
		"cors_vary":                      strings.TrimSpace(c.Writer.Header().Get("Vary")),
		"response_trace_header_prewrite": strings.TrimSpace(c.Writer.Header().Get("X-Trace-Id")),
	}).Info("capability invoke response headers")
}

func firstGatewayCode(err *frameworkgateway.InvocationError) string {
	if err == nil {
		return ""
	}
	if len(err.Errors) == 0 {
		return ""
	}
	return err.Errors[0].Code
}

func firstGatewayMessage(err *frameworkgateway.InvocationError) string {
	if err == nil {
		return ""
	}
	if len(err.Errors) == 0 || strings.TrimSpace(err.Errors[0].Message) == "" {
		text := strings.TrimSpace(string(err.Body))
		if text != "" {
			return text
		}
		return err.Error()
	}
	return err.Errors[0].Message
}

func categorizeGatewayError(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "validation"
	case http.StatusUnauthorized, http.StatusForbidden:
		return "unauthorized"
	case http.StatusTooManyRequests:
		return "rate_limited"
	default:
		return "upstream"
	}
}

func parseGatewayBody(body []byte) any {
	if len(body) == 0 {
		return nil
	}
	text := strings.TrimSpace(string(body))
	if text == "" {
		return nil
	}
	var decoded any
	if json.Unmarshal(body, &decoded) == nil {
		return decoded
	}
	return text
}
