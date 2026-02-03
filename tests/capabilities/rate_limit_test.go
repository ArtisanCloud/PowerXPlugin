package capabilities_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/router"
)

func TestCapabilityRateLimitEmitsAuditEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Trace-Id", "trace-limit-audit")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"traceId":"trace-limit-audit","errors":[{"code":"RATE_LIMIT","message":"slow down"}]}`))
	}))
	defer server.Close()

	app := bootstrap.NewApp(&bootstrap.Config{
		Gateway: bootstrap.GatewayConfig{
			BaseURL:   server.URL,
			ToolToken: "test-token",
			TenantID:  "tenant-123",
		},
	})

	var logBuf bytes.Buffer
	app.Logger = slog.New(slog.NewJSONHandler(&logBuf, nil))

	recorder := newRecordingRouter()
	app.AttachRouter(recorder)
	router.RegisterFrameworkRoutes(app)

	handler := recorder.handler(http.MethodPost, router.APIPrefix+"/integration/capabilities/invoke")
	if handler == nil {
		t.Fatalf("capability proxy handler not registered")
	}

	body, _ := json.Marshal(map[string]any{
		"capabilityId": "com.corex.media.assets.manage",
		"action":       "Create",
		"payload":      map[string]any{"assetName": "doc.pdf"},
	})
	ctx := newMockContext(body)
	ctx.reqHeaders["X-PowerX-Tenant"] = "tenant-321"
	ctx.reqHeaders["X-Request-ID"] = "req-limit-1"

	handler(ctx)

	if ctx.status != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", ctx.status)
	}

	logs := logBuf.String()
	if !strings.Contains(logs, "capability.invoke.rate_limit") {
		t.Fatalf("expected capability.invoke.rate_limit log, got %s", logs)
	}
	if !strings.Contains(logs, "audit.capability.invocation.denied") {
		t.Fatalf("expected audit log for denied invocation, got %s", logs)
	}
	if !strings.Contains(logs, "trace-limit-audit") {
		t.Fatalf("expected traceId in logs, got %s", logs)
	}
}
