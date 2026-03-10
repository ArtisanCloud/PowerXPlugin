package capabilityinvoker

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/internal/integration/gateway"
)

type stubGateway struct {
	resp    *gateway.Response
	err     error
	lastReq gateway.InvokeRequest
}

func (s *stubGateway) Invoke(ctx context.Context, req gateway.InvokeRequest) (*gateway.Response, error) {
	s.lastReq = req
	return s.resp, s.err
}

func TestServiceInvokeSuccess(t *testing.T) {
	stub := &stubGateway{
		resp: &gateway.Response{
			TraceID: "trace-1",
			Status:  "ok",
			Data: map[string]any{
				"ok": true,
			},
		},
	}
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))
	svc := NewService(stub, logger)

	result, err := svc.Invoke(context.Background(), InvokeParams{
		CapabilityID: "com.corex.demo",
		Action:       "List",
		Payload: map[string]any{
			"page": 1,
		},
		RequestID:  "req-1",
		TenantUUID: "tenant-success",
	})
	if err != nil {
		t.Fatalf("Invoke returned error: %v", err)
	}
	if result.TraceID != "trace-1" {
		t.Fatalf("expected trace trace-1, got %s", result.TraceID)
	}
	if stub.lastReq.CapabilityID != "com.corex.demo" {
		t.Fatalf("capability ID not forwarded, got %s", stub.lastReq.CapabilityID)
	}
	if stub.lastReq.Action != "List" {
		t.Fatalf("action not forwarded, got %s", stub.lastReq.Action)
	}
	if stub.lastReq.RequestID != "req-1" {
		t.Fatalf("request ID not forwarded, got %s", stub.lastReq.RequestID)
	}
	if val, ok := result.Data["ok"].(bool); !ok || !val {
		t.Fatalf("expected data map with ok=true, got %#v", result.Data)
	}
	if stub.lastReq.Headers["tenant_uuid"] != "tenant-success" {
		t.Fatalf("tenant header not forwarded, got %s", stub.lastReq.Headers["tenant_uuid"])
	}
	if !strings.Contains(logBuf.String(), "capability.invoke.success") {
		t.Fatalf("success log missing, got %s", logBuf.String())
	}
	if !strings.Contains(logBuf.String(), "tenant-success") {
		t.Fatalf("tenant uuid missing in log, got %s", logBuf.String())
	}
}

func TestServiceInvokeValidation(t *testing.T) {
	svc := NewService(nil, slog.Default())
	_, err := svc.Invoke(context.Background(), InvokeParams{
		CapabilityID: "",
		Action:       "Create",
		Payload:      map[string]any{},
	})

	var invErr *InvokeError
	if !errors.As(err, &invErr) {
		t.Fatalf("expected InvokeError, got %v", err)
	}
	if invErr.Category != ErrorCategoryValidation {
		t.Fatalf("expected validation category, got %s", invErr.Category)
	}
}

func TestServiceInvokeGatewayError(t *testing.T) {
	stub := &stubGateway{
		err: &gateway.InvocationError{
			TraceID:    "trace-err",
			StatusCode: http.StatusTooManyRequests,
			Errors: []gateway.GatewayError{
				{Code: "RATE_LIMIT", Message: "too many"},
			},
		},
	}
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))
	svc := NewService(stub, logger)

	_, err := svc.Invoke(context.Background(), InvokeParams{
		CapabilityID: "com.corex.demo",
		Action:       "Create",
		Payload: map[string]any{
			"name": "demo",
		},
		RequestID:  "rid",
		TenantUUID: "tenant-failure",
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	var invErr *InvokeError
	if !errors.As(err, &invErr) {
		t.Fatalf("expected InvokeError, got %v", err)
	}
	if invErr.Category != ErrorCategoryRateLimited {
		t.Fatalf("expected rate limited category, got %s", invErr.Category)
	}
	if invErr.Code != "RATE_LIMIT" {
		t.Fatalf("expected code RATE_LIMIT, got %s", invErr.Code)
	}
	if invErr.TraceID != "trace-err" {
		t.Fatalf("expected trace trace-err, got %s", invErr.TraceID)
	}
	if !strings.Contains(logBuf.String(), "capability.invoke.rate_limit") {
		t.Fatalf("rate limit log missing, got %s", logBuf.String())
	}
	if !strings.Contains(logBuf.String(), "audit.capability.invocation.denied") {
		t.Fatalf("audit log missing, got %s", logBuf.String())
	}
}

func TestServiceInvokeUnauthorizedAudit(t *testing.T) {
	stub := &stubGateway{
		err: &gateway.InvocationError{
			TraceID:    "trace-unauth",
			StatusCode: http.StatusUnauthorized,
			Errors: []gateway.GatewayError{
				{Code: "UNAUTHORIZED", Message: "forbidden"},
			},
		},
	}
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))
	svc := NewService(stub, logger)

	_, err := svc.Invoke(context.Background(), InvokeParams{
		CapabilityID: "com.corex.demo",
		Action:       "Read",
		Payload:      map[string]any{"id": "1"},
		RequestID:    "req-auth",
		TenantUUID:   "tenant-auth",
	})
	if err == nil {
		t.Fatalf("expected unauthorized error")
	}
	if !strings.Contains(logBuf.String(), "audit.capability.invocation.denied") {
		t.Fatalf("missing audit log for unauthorized: %s", logBuf.String())
	}
}
