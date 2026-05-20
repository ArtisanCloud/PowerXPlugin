package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	gatewaypb "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/internal/integration/gateway/pb"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const (
	defaultUserAgent         = "powerx-plugin-gateway-client/0.1"
	defaultTimeout           = 10 * time.Second
	invokePath               = "/tenant/invocations"
	defaultContractDigestRel = "dist/capability-contracts.json"
)

// Config 描述构造 Gateway Client 所需的参数。
type Config struct {
	BaseURL            string
	APIPrefix          string
	TenantUUID         string
	BearerToken        string
	APIKey             string
	AuthScheme         string
	HTTPClient         *http.Client
	RequestTimeout     time.Duration
	UserAgent          string
	GRPCTarget         string
	GRPCDialOptions    []grpc.DialOption
	ContractVersion    string
	ContractDigestPath string
}

// InvokeRequest 描述一次能力调用请求体。
type InvokeRequest struct {
	CapabilityID      string
	Action            string
	PreferredProtocol string
	Payload           any
	RequestID         string
	Headers           map[string]string
	TenantUUID        string
	DisableAuth       bool
}

// GatewayError 映射 Gateway 返回的错误条目。
type GatewayError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Response 封装标准响应数据。
type Response struct {
	TraceID           string
	Status            string
	UpstreamRequestID string
	Data              map[string]any
	RawData           json.RawMessage
	Errors            []GatewayError
}

// ContractStatus 描述当前契约摘要与期望版本的比对状态。
type ContractStatus struct {
	CurrentVersion  string `json:"currentVersion"`
	CurrentHash     string `json:"currentHash"`
	ExpectedVersion string `json:"expectedVersion"`
	GeneratedAt     string `json:"generatedAt"`
	DigestSource    string `json:"digestSource"`
	Outdated        bool   `json:"outdated"`
	Message         string `json:"message"`
}

// InvocationError 提供标准化错误对象。
type InvocationError struct {
	TraceID           string
	UpstreamRequestID string
	StatusCode        int
	Errors            []GatewayError
	Body              []byte
}

func (e *InvocationError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if len(e.Errors) == 0 {
		return fmt.Sprintf("gateway invocation failed: status=%d trace=%s", e.StatusCode, e.TraceID)
	}
	return fmt.Sprintf("gateway invocation failed: %s (trace=%s status=%d)", e.Errors[0].Message, e.TraceID, e.StatusCode)
}

// Client 负责 REST/gRPC 能力调用封装。
type Client struct {
	baseURL        string
	apiPrefix      string
	tenantUUID     string
	authScheme     string
	credential     string
	userAgent      string
	requestTimeout time.Duration

	httpClient *http.Client

	grpcTarget      string
	grpcDialOptions []grpc.DialOption

	grpcOnce   sync.Once
	grpcConn   *grpc.ClientConn
	grpcClient gatewaypb.IntegrationGatewayTenantServiceClient
	grpcErr    error

	contractStatus *ContractStatus
}

// NewClient 构造 Gateway Client。
func NewClient(cfg Config) (*Client, error) {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		return nil, errors.New("gateway: base URL is required")
	}
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}
	authScheme, credential, err := resolveAuth(cfg.AuthScheme, cfg.BearerToken, cfg.APIKey)
	if err != nil {
		return nil, err
	}
	tenant := strings.TrimSpace(cfg.TenantUUID)
	timeout := cfg.RequestTimeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}
	ua := strings.TrimSpace(cfg.UserAgent)
	if ua == "" {
		ua = defaultUserAgent
	}
	client := &Client{
		baseURL:         strings.TrimRight(baseURL, "/"),
		apiPrefix:       normalizeAPIPrefix(cfg.APIPrefix),
		tenantUUID:      tenant,
		authScheme:      authScheme,
		credential:      credential,
		userAgent:       ua,
		requestTimeout:  timeout,
		httpClient:      httpClient,
		grpcTarget:      strings.TrimSpace(cfg.GRPCTarget),
		grpcDialOptions: cfg.GRPCDialOptions,
	}
	client.inspectContractVersion(cfg.ContractVersion, cfg.ContractDigestPath)
	return client, nil
}

// Invoke 通过 REST `/tenant/invocations` 触发能力调用。
func (c *Client) Invoke(ctx context.Context, req InvokeRequest) (*Response, error) {
	if err := validateInvokeRequest(req); err != nil {
		return nil, err
	}
	body := map[string]any{
		"capability_id": req.CapabilityID,
		"payload":       ensurePayload(req.Payload),
	}
	if value := strings.TrimSpace(req.Action); value != "" {
		body["action"] = value
	}
	if value := strings.TrimSpace(req.PreferredProtocol); value != "" {
		body["preferred_protocol"] = value
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("gateway: encode payload: %w", err)
	}
	requestID := strings.TrimSpace(req.RequestID)
	if requestID == "" {
		requestID = uuid.NewString()
	}
	endpoint := buildGatewayEndpoint(c.baseURL, c.apiPrefix, invokePath)
	slog.Default().Info("gateway.invoke.request",
		slog.String("endpoint", endpoint),
		slog.String("capability_id", req.CapabilityID),
		slog.String("preferred_protocol", strings.TrimSpace(req.PreferredProtocol)),
		slog.String("action", strings.TrimSpace(req.Action)),
		slog.String("auth_scheme", c.authScheme),
		slog.String("request_body", string(bodyBytes)),
	)

	ctxReq, cancel := context.WithTimeout(ctx, c.requestTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctxReq, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("gateway: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	if !req.DisableAuth {
		httpReq.Header.Set("Authorization", buildAuthHeader(c.authScheme, c.credential))
	}
	httpReq.Header.Set("X-Request-ID", requestID)
	if c.userAgent != "" {
		httpReq.Header.Set("User-Agent", c.userAgent)
	}
	for k, v := range req.Headers {
		if strings.TrimSpace(v) == "" {
			continue
		}
		httpReq.Header.Set(k, v)
	}
	slog.Default().Info("gateway.invoke.auth",
		slog.String("request_id", requestID),
		slog.String("authorization_scheme", extractAuthScheme(httpReq.Header.Get("Authorization"))),
	)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gateway: http invoke failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gateway: read response: %w", err)
	}

	envelope := parseEnvelope(respBody)
	traceID := envelope.TraceID
	if traceID == "" {
		traceID = resp.Header.Get("X-Trace-Id")
	}

	result := &Response{
		TraceID:           traceID,
		Status:            envelope.Status,
		UpstreamRequestID: firstNonEmpty(strings.TrimSpace(resp.Header.Get("X-Request-ID")), strings.TrimSpace(envelope.RequestID)),
		RawData:           envelope.Data,
		Errors:            envelope.Errors,
	}
	if len(envelope.Data) > 0 && string(envelope.Data) != "null" {
		var data map[string]any
		if err := json.Unmarshal(envelope.Data, &data); err == nil {
			result.Data = data
		}
	}

	if resp.StatusCode >= http.StatusBadRequest || len(envelope.Errors) > 0 {
		return result, &InvocationError{
			TraceID:           traceID,
			UpstreamRequestID: result.UpstreamRequestID,
			StatusCode:        resp.StatusCode,
			Errors:            envelope.Errors,
			Body:              respBody,
		}
	}

	return result, nil
}

// InvokeGRPC 通过 gRPC `InvokeCapability` 触发能力调用。
func (c *Client) InvokeGRPC(ctx context.Context, req InvokeRequest) (*Response, error) {
	if err := validateInvokeRequest(req); err != nil {
		return nil, err
	}
	client, err := c.grpcClientOrInit(ctx)
	if err != nil {
		return nil, err
	}
	payloadBytes, err := json.Marshal(req.Payload)
	if err != nil {
		return nil, fmt.Errorf("gateway: encode payload: %w", err)
	}
	requestID := strings.TrimSpace(req.RequestID)
	if requestID == "" {
		requestID = uuid.NewString()
	}

	grpcReq := &gatewaypb.InvokeCapabilityRequest{
		CapabilityId: req.CapabilityID,
		Action:       req.Action,
		PayloadJson:  payloadBytes,
		RequestId:    requestID,
	}

	mdValues := map[string]string{
		"x-request-id": requestID,
	}
	if !req.DisableAuth {
		mdValues["authorization"] = buildAuthHeader(c.authScheme, c.credential)
	}
	md := metadata.New(mdValues)
	ctxCall, cancel := context.WithTimeout(ctx, c.requestTimeout)
	defer cancel()
	ctxCall = metadata.NewOutgoingContext(ctxCall, md)

	resp, err := client.InvokeCapability(ctxCall, grpcReq)
	if err != nil {
		return nil, fmt.Errorf("gateway: grpc invoke failed: %w", err)
	}

	result := &Response{TraceID: resp.GetTraceId(), Status: resp.GetStatus()}
	if data := resp.GetResultJson(); len(data) > 0 && string(data) != "null" {
		var parsed map[string]any
		if err := json.Unmarshal(data, &parsed); err == nil {
			result.Data = parsed
			result.RawData = json.RawMessage(data)
		}
	}
	if len(resp.GetErrors()) > 0 {
		result.Errors = convertErrors(resp.GetErrors())
		return result, &InvocationError{TraceID: result.TraceID, StatusCode: http.StatusBadGateway, Errors: result.Errors}
	}
	return result, nil
}

// Close 释放底层 gRPC 连接。
func (c *Client) Close() error {
	if c.grpcConn != nil {
		return c.grpcConn.Close()
	}
	return nil
}

// ContractStatus 返回当前缓存的契约版本状态。
func (c *Client) ContractStatus() *ContractStatus {
	if c == nil || c.contractStatus == nil {
		return nil
	}
	status := *c.contractStatus
	return &status
}

func (c *Client) grpcClientOrInit(ctx context.Context) (gatewaypb.IntegrationGatewayTenantServiceClient, error) {
	c.grpcOnce.Do(func() {
		if c.grpcClient != nil {
			return
		}
		if c.grpcTarget == "" {
			c.grpcErr = errors.New("gateway: grpc target not configured")
			return
		}
		opts := c.grpcDialOptions
		if len(opts) == 0 {
			opts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		}
		dialCtx, cancel := context.WithTimeout(ctx, c.requestTimeout)
		defer cancel()
		conn, err := grpc.DialContext(dialCtx, c.grpcTarget, opts...)
		if err != nil {
			c.grpcErr = fmt.Errorf("gateway: dial grpc: %w", err)
			return
		}
		c.grpcConn = conn
		c.grpcClient = gatewaypb.NewIntegrationGatewayTenantServiceClient(conn)
	})
	if c.grpcErr != nil {
		return nil, c.grpcErr
	}
	if c.grpcClient == nil {
		return nil, errors.New("gateway: grpc client not initialized")
	}
	return c.grpcClient, nil
}

func validateInvokeRequest(req InvokeRequest) error {
	if strings.TrimSpace(req.CapabilityID) == "" {
		return errors.New("gateway: capability ID is required")
	}
	return nil
}

func ensurePayload(value any) any {
	if value == nil {
		return map[string]any{}
	}
	return value
}

func convertErrors(src []*gatewaypb.GatewayError) []GatewayError {
	if len(src) == 0 {
		return nil
	}
	out := make([]GatewayError, 0, len(src))
	for _, err := range src {
		if err == nil {
			continue
		}
		out = append(out, GatewayError{Code: err.GetCode(), Message: err.GetMessage()})
	}
	return out
}

type restEnvelope struct {
	TraceID   string          `json:"traceId"`
	RequestID string          `json:"request_id"`
	Status    string          `json:"status"`
	Data      json.RawMessage `json:"data"`
	Errors    []GatewayError  `json:"errors"`
}

func parseEnvelope(body []byte) restEnvelope {
	if len(body) == 0 {
		return restEnvelope{}
	}
	var env restEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return restEnvelope{}
	}
	return env
}

func resolveAuth(rawScheme, toolToken, apiKey string) (scheme string, credential string, err error) {
	scheme = normalizeAuthScheme(rawScheme)
	bearer := strings.TrimSpace(toolToken)
	key := strings.TrimSpace(apiKey)

	switch scheme {
	case "bearer":
		if bearer == "" {
			return "", "", errors.New("gateway: bearer credential is required when auth_scheme=bearer")
		}
		return scheme, bearer, nil
	case "apikey":
		if key == "" {
			return "", "", errors.New("gateway: PX_GATEWAY_API_KEY is required when auth_scheme=apikey")
		}
		return scheme, key, nil
	default:
		return "", "", errors.New("gateway: auth_scheme must be bearer or apikey")
	}
}

func normalizeAuthScheme(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "bearer":
		return "bearer"
	case "apikey", "api_key", "api-key":
		return "apikey"
	default:
		return ""
	}
}

func normalizeAPIPrefix(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "/api/v1"
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	value = "/" + strings.Trim(strings.TrimSpace(value), "/")
	if value == "/" {
		return "/api/v1"
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func buildGatewayEndpoint(baseURL, apiPrefix, routePath string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	route := "/" + strings.TrimLeft(strings.TrimSpace(routePath), "/")
	prefix := normalizeAPIPrefix(apiPrefix)
	endpoint := ""
	if prefix == "" {
		endpoint = base + route
	} else if strings.HasSuffix(base, prefix) {
		endpoint = base + route
	} else {
		endpoint = base + prefix + route
	}
	return endpoint
}

func buildAuthHeader(scheme, credential string) string {
	switch normalizeAuthScheme(scheme) {
	case "apikey":
		return "ApiKey " + strings.TrimSpace(credential)
	default:
		return "Bearer " + strings.TrimSpace(credential)
	}
}

func extractAuthScheme(authHeader string) string {
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

func (c *Client) inspectContractVersion(expectedVersion, digestPath string) {
	snapshot, err := loadContractDigest(digestPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			slog.Default().Warn("gateway: failed to load contract digest", slog.String("path", digestPath), slog.String("error", err.Error()))
		}
		return
	}
	status := &ContractStatus{
		CurrentVersion:  snapshot.ManifestVersion,
		CurrentHash:     snapshot.BundleHash,
		DigestSource:    snapshot.SourcePath,
		ExpectedVersion: strings.TrimSpace(expectedVersion),
	}
	if !snapshot.GeneratedAt.IsZero() {
		status.GeneratedAt = snapshot.GeneratedAt.Format(time.RFC3339)
	}
	if status.ExpectedVersion == "" {
		status.Message = "gateway contract digest loaded"
	} else if !strings.EqualFold(status.ExpectedVersion, status.CurrentHash) && !strings.EqualFold(status.ExpectedVersion, status.CurrentVersion) {
		status.Outdated = true
		status.Message = fmt.Sprintf("Gateway 契约版本需升级：期望 %s，当前哈希 %s", status.ExpectedVersion, status.CurrentHash)
	} else {
		status.Message = "gateway contract matches expected version"
	}
	c.contractStatus = status
	logger := slog.Default()
	fields := []any{
		slog.String("currentHash", status.CurrentHash),
		slog.String("manifestVersion", status.CurrentVersion),
		slog.String("digestSource", status.DigestSource),
	}
	if status.ExpectedVersion != "" {
		fields = append(fields, slog.String("expectedVersion", status.ExpectedVersion))
	}
	if status.Outdated {
		logger.Warn("gateway contract outdated", fields...)
	} else {
		logger.Info("gateway contract verified", fields...)
	}
}

type contractDigestSnapshot struct {
	BundleHash      string
	ManifestVersion string
	GeneratedAt     time.Time
	SourcePath      string
}

type contractDigestFile struct {
	GeneratedAt string `json:"generatedAt"`
	Manifest    struct {
		Version string `json:"version"`
	} `json:"manifest"`
	Digest struct {
		BundlesHash string `json:"bundlesHash"`
	} `json:"digest"`
}

func loadContractDigest(pathArg string) (*contractDigestSnapshot, error) {
	pathValue := strings.TrimSpace(pathArg)
	if pathValue == "" {
		pathValue = defaultContractDigestRel
	}
	absPath := pathValue
	if !filepath.IsAbs(absPath) {
		if base, err := os.Getwd(); err == nil {
			absPath = filepath.Join(base, pathValue)
		} else {
			return nil, fmt.Errorf("resolve contract digest path: %w", err)
		}
	}
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	var parsed contractDigestFile
	if err := json.Unmarshal(content, &parsed); err != nil {
		return nil, fmt.Errorf("decode contract digest: %w", err)
	}
	snapshot := &contractDigestSnapshot{
		BundleHash:      strings.TrimSpace(parsed.Digest.BundlesHash),
		ManifestVersion: strings.TrimSpace(parsed.Manifest.Version),
		SourcePath:      absPath,
	}
	if parsed.GeneratedAt != "" {
		if ts, err := time.Parse(time.RFC3339Nano, parsed.GeneratedAt); err == nil {
			snapshot.GeneratedAt = ts
		}
	}
	return snapshot, nil
}
