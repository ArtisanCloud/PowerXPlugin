package runtime_ops

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	fwprovider "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

func TestWSBusTestFlowHostModeDoesNotPublishLocalEcho(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("POWERX_PROXY", "1")

	var hostGrantCount atomic.Int32
	var hostPublishCount atomic.Int32
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/admin/runtime/ws-bus/grant":
			hostGrantCount.Add(1)
		case "/api/v1/admin/runtime/ws-bus/publish":
			hostPublishCount.Add(1)
		default:
			t.Errorf("unexpected host path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"ok":true}}`))
	}))
	defer host.Close()

	hub := fwwsbus.NewMemoryHub()
	localEvents := make(chan fwwsbus.Event, 1)
	hub.Subscribe("_topic.system.notification", func(ev fwwsbus.Event) {
		localEvents <- ev
	})
	deps := hostStrictWSBusDeps(host.URL, hub)

	router := gin.New()
	router.POST("/test-flow", WSBusTestFlowHandler(deps))
	resp := postRuntimeOpsJSON(t, router, "/test-flow", map[string]any{
		"topic":    "_topic.system.notification",
		"trace_id": "trace-host-flow",
		"payload":  map[string]any{"type": "framework.wsbus.test"},
	}, map[string]string{"tenant_uuid": "tenant-001"})
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}

	var body struct {
		Data struct {
			FlowMode    string `json:"flow_mode"`
			EchoOK      bool   `json:"echo_ok"`
			EchoSkipped bool   `json:"echo_skipped"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Data.FlowMode != "host_strict_ok" {
		t.Fatalf("flow_mode=%q body=%s", body.Data.FlowMode, resp.Body.String())
	}
	if body.Data.EchoOK || !body.Data.EchoSkipped {
		t.Fatalf("host mode should skip local echo, body=%s", resp.Body.String())
	}
	if hostGrantCount.Load() != 1 || hostPublishCount.Load() != 1 {
		t.Fatalf("host grant/publish counts = %d/%d", hostGrantCount.Load(), hostPublishCount.Load())
	}
	assertNoLocalWSEvent(t, localEvents)
}

func TestNotificationTestHostModeDoesNotPublishLocalEcho(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("POWERX_PROXY", "1")

	var hostNotifyCount atomic.Int32
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/notifications/test" {
			t.Errorf("unexpected host path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		hostNotifyCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"ok":true}}`))
	}))
	defer host.Close()

	hub := fwwsbus.NewMemoryHub()
	localEvents := make(chan fwwsbus.Event, 1)
	hub.Subscribe("_topic.system.notification", func(ev fwwsbus.Event) {
		localEvents <- ev
	})
	deps := hostStrictWSBusDeps(host.URL, hub)

	router := gin.New()
	router.POST("/notifications/test", NotificationTestHandler(deps))
	resp := postRuntimeOpsJSON(t, router, "/notifications/test", map[string]any{
		"tenant_uuid": "tenant-001",
		"topic":       "_topic.system.notification",
		"title":       "PowerX host notification",
		"message":     "host only",
		"trace_id":    "trace-host-notify",
	}, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}

	var body struct {
		Data struct {
			FlowMode        string `json:"flow_mode"`
			EffectiveTarget string `json:"effective_target"`
			HostPublishOK   bool   `json:"host_publish_ok"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Data.FlowMode != "host_strict_ok" || body.Data.EffectiveTarget != "host" || !body.Data.HostPublishOK {
		t.Fatalf("unexpected host response body=%s", resp.Body.String())
	}
	if hostNotifyCount.Load() != 1 {
		t.Fatalf("host notification count=%d", hostNotifyCount.Load())
	}
	assertNoLocalWSEvent(t, localEvents)
}

func hostStrictWSBusDeps(baseURL string, hub fwwsbus.LocalHub) *app.Deps {
	return &app.Deps{
		WSBusHub:     hub,
		ProviderMode: fwprovider.ModeLocal,
		Config: &config.Config{
			Gateway: &config.GatewayConfig{
				BaseURL:    strings.TrimSpace(baseURL),
				APIPrefix:  "/api/v1",
				AuthScheme: "apikey",
				APIKey:     "test-api-key",
				Timeout:    2 * time.Second,
			},
		},
	}
}

func postRuntimeOpsJSON(t *testing.T, router http.Handler, path string, body map[string]any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	router.ServeHTTP(resp, req)
	return resp
}

func assertNoLocalWSEvent(t *testing.T, events <-chan fwwsbus.Event) {
	t.Helper()
	select {
	case ev := <-events:
		t.Fatalf("host mode must not publish local ws event, got topic=%s trace_id=%s", ev.Topic, ev.TraceID)
	default:
	}
}
