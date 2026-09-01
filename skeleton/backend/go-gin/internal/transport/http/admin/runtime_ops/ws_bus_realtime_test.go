package runtime_ops

import (
	"net/http"
	"net/http/httptest"
	"testing"

	frameworkrealtime "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/realtime"
	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

func TestWSBusPublishRejectsUndeclaredTopic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/publish", WSBusPublishHandler(&app.Deps{
		WSBusHub: fwwsbus.NewMemoryHub(),
		RealtimeDescriptors: []frameworkrealtime.Descriptor{{
			Key:        "_topic.template.update",
			Protocols:  []frameworkrealtime.Protocol{frameworkrealtime.ProtocolWS},
			Actions:    []frameworkrealtime.Action{frameworkrealtime.ActionPublish},
			Scope:      frameworkrealtime.ScopeTenant,
			EventTypes: []string{"message"},
		}},
	}))

	resp := postRuntimeOpsJSON(t, router, "/publish", map[string]any{
		"topic":       "_topic.undeclared",
		"tenant_uuid": "tenant-001",
	}, map[string]string{"tenant_uuid": "tenant-001"})
	if resp.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func TestWSBusTestFlowPublishesDeclaredTopic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hub := fwwsbus.NewMemoryHub()
	events := make(chan fwwsbus.Event, 1)
	hub.Subscribe("_topic.template.update", func(event fwwsbus.Event) { events <- event })
	router := gin.New()
	router.POST("/test-flow", WSBusTestFlowHandler(&app.Deps{
		WSBusHub: hub,
		RealtimeDescriptors: []frameworkrealtime.Descriptor{{
			Key:        "_topic.template.update",
			Protocols:  []frameworkrealtime.Protocol{frameworkrealtime.ProtocolWS},
			Actions:    []frameworkrealtime.Action{frameworkrealtime.ActionPublish},
			Scope:      frameworkrealtime.ScopeTenant,
			EventTypes: []string{"message"},
		}},
	}))

	resp := postRuntimeOpsJSON(t, router, "/test-flow", map[string]any{
		"topic":    "_topic.template.update",
		"trace_id": "trace-001",
	}, map[string]string{"tenant_uuid": "tenant-001"})
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	select {
	case event := <-events:
		if event.Topic != "_topic.template.update" || event.TenantUUID != "tenant-001" {
			t.Fatalf("unexpected event=%+v", event)
		}
	default:
		t.Fatal("expected local websocket event")
	}
}

func TestAllowWSBusPublishRejectsMissingTenantScope(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/publish", nil)
	allowed := allowWSBusPublish(c, &app.Deps{RealtimeDescriptors: []frameworkrealtime.Descriptor{{
		Key:        "_topic.template.update",
		Protocols:  []frameworkrealtime.Protocol{frameworkrealtime.ProtocolWS},
		Actions:    []frameworkrealtime.Action{frameworkrealtime.ActionPublish},
		Scope:      frameworkrealtime.ScopeTenant,
		EventTypes: []string{"message"},
	}}}, "_topic.template.update", "", "", "trace-001")
	if allowed {
		t.Fatal("missing tenant scope must be rejected")
	}
}
