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

func TestLoggingProbeHandlerContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalPolicy := currentLoggingPolicyForTenant("tenant-001")
	t.Cleanup(func() {
		setLoggingPolicyForTenant("tenant-001", originalPolicy)
	})
	setLoggingPolicyForTenant("tenant-001", runtimelogging.Policy{
		Mode:   runtimelogging.ModeStandalone,
		Sinks:  []runtimelogging.SinkType{runtimelogging.SinkStdout, runtimelogging.SinkFile},
		Format: "json",
		Level:  "info",
		Retry: runtimelogging.RetryPolicy{
			Enabled:     true,
			MaxAttempts: 2,
			BackoffMS:   1,
		},
	})

	router := gin.New()
	router.POST("/logging/probe", LoggingProbeHandler())

	body := map[string]any{
		"message":     "probe-us2",
		"level":       "info",
		"component":   "runtime.ops.test",
		"tenant_uuid": "tenant-001",
		"trace_id":    "trace-us2-001",
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal payload failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/logging/probe", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status mismatch, got=%d body=%s", rec.Code, rec.Body.String())
	}
	var envelope struct {
		Code int    `json:"code"`
		Data struct {
			TraceID  string                       `json:"trace_id"`
			Outcomes []runtimelogging.SinkOutcome `json:"outcomes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if envelope.Code != 0 {
		t.Fatalf("expected code=0 envelope: %s", rec.Body.String())
	}
	if envelope.Data.TraceID != "trace-us2-001" {
		t.Fatalf("trace_id mismatch, got=%s", envelope.Data.TraceID)
	}
	if len(envelope.Data.Outcomes) != 2 {
		t.Fatalf("outcomes length mismatch, got=%d", len(envelope.Data.Outcomes))
	}
	for _, outcome := range envelope.Data.Outcomes {
		if outcome.Status != runtimelogging.OutcomeSuccess {
			t.Fatalf("expected success outcome, got=%+v", outcome)
		}
	}
}

func TestLoggingProbeHandlerInvalidPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/logging/probe", LoggingProbeHandler())

	req := httptest.NewRequest(http.MethodPost, "/logging/probe", bytes.NewBufferString("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status mismatch, got=%d body=%s", rec.Code, rec.Body.String())
	}
}
