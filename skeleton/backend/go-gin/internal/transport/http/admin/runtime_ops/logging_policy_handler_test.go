package runtime_ops

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	runtimelogging "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/common/logging"
	"github.com/gin-gonic/gin"
)

func TestLoggingPolicyHandlerGetContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "tenant-logging-get"

	original := currentLoggingPolicyForTenant(tenantUUID)
	t.Cleanup(func() {
		setLoggingPolicyForTenant(tenantUUID, original)
	})
	setLoggingPolicyForTenant(tenantUUID, runtimelogging.Policy{
		PolicyVersion: "v1",
		Mode:          runtimelogging.ModeStandalone,
		Sinks:         []runtimelogging.SinkType{runtimelogging.SinkStdout},
		Format:        "json",
		Level:         "info",
		Retry: runtimelogging.RetryPolicy{
			Enabled:     true,
			MaxAttempts: 2,
			BackoffMS:   100,
		},
	})

	router := gin.New()
	handler := NewLoggingPolicyHandler()
	router.GET("/logging/policy", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/logging/policy?tenant_uuid="+tenantUUID, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status mismatch, got=%d body=%s", rec.Code, rec.Body.String())
	}
	var envelope struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Mode   runtimelogging.PolicyMode `json:"mode"`
			Sinks  []runtimelogging.SinkType `json:"sinks"`
			Format string                    `json:"format"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if envelope.Code != 0 || envelope.Message != "ok" {
		t.Fatalf("unexpected envelope: %s", rec.Body.String())
	}
	if envelope.Data.Mode != runtimelogging.ModeStandalone {
		t.Fatalf("mode mismatch, got=%s", envelope.Data.Mode)
	}
	if len(envelope.Data.Sinks) != 1 || envelope.Data.Sinks[0] != runtimelogging.SinkStdout {
		t.Fatalf("sinks mismatch, got=%v", envelope.Data.Sinks)
	}
	if envelope.Data.Format != "json" {
		t.Fatalf("format mismatch, got=%s", envelope.Data.Format)
	}
}

func TestLoggingPolicyHandlerPutPreservesStandalonePolicyWhenPowerXProxyEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("POWERX_PROXY", "1")

	router := gin.New()
	handler := NewLoggingPolicyHandler()
	router.PUT("/logging/policy", handler.Put)

	body := map[string]any{
		"tenant_uuid":            "tenant-logging-put",
		"policy_version":         "v2",
		"mode":                   "standalone",
		"sinks":                  []string{"file", "loki"},
		"format":                 "text",
		"level":                  "debug",
		"authorized_extra_sinks": []string{"loki"},
		"retry": map[string]any{
			"enabled":      true,
			"max_attempts": 2,
			"backoff_ms":   100,
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal payload failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/logging/policy", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status mismatch, got=%d body=%s", rec.Code, rec.Body.String())
	}
	var envelope struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Mode                 runtimelogging.PolicyMode `json:"mode"`
			Sinks                []runtimelogging.SinkType `json:"sinks"`
			Format               string                    `json:"format"`
			AuthorizedExtraSinks []runtimelogging.SinkType
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if envelope.Code != 0 || envelope.Message != "ok" {
		t.Fatalf("unexpected envelope: %s", rec.Body.String())
	}
	if envelope.Data.Mode != runtimelogging.ModeStandalone {
		t.Fatalf("expected mode=standalone, got=%s", envelope.Data.Mode)
	}
	if envelope.Data.Format != "text" {
		t.Fatalf("expected format=text, got=%s", envelope.Data.Format)
	}
	if len(envelope.Data.Sinks) != 2 || envelope.Data.Sinks[0] != runtimelogging.SinkFile || envelope.Data.Sinks[1] != runtimelogging.SinkLoki {
		t.Fatalf("expected sinks=[file loki], got=%v", envelope.Data.Sinks)
	}
	if len(envelope.Data.AuthorizedExtraSinks) != 1 || envelope.Data.AuthorizedExtraSinks[0] != runtimelogging.SinkLoki {
		t.Fatalf("expected authorized_extra_sinks=[loki], got=%v", envelope.Data.AuthorizedExtraSinks)
	}
}
