package integration

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/event"
	fweventbridge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/eventbridge"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	integrationjobs "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/jobs/integration"
)

func TestSchedulerManualCronParityAndTracePropagation(t *testing.T) {
	emitter := fweventbridge.NewLocalEmitter(16)

	cfg := &config.Config{
		EventBridge: &config.EventBridgeConfig{
			SourcePlugin:   "com.powerx.plugins.base",
			PayloadVersion: "v1",
		},
		GRPCUpstream: &config.GRPCUpstream{TenantUUID: "00000000-0000-0000-0000-000000000001"},
	}
	dispatcher := integrationjobs.NewSchedulerEventDispatcher(cfg, emitter, logrus.NewEntry(logrus.New()))

	manualTraceID := "manual-trace-001"
	manualMeta, err := event.NewMetaBuilder("com.powerx.plugins.base", "v1").Build("00000000-0000-0000-0000-000000000001", manualTraceID, manualTraceID)
	if err != nil {
		t.Fatalf("build manual meta failed: %v", err)
	}
	manualPayload, _ := json.Marshal(map[string]any{
		"source":          "manual",
		"trigger_source":  "manual",
		"job_name":        "runtime.scheduler.trigger",
		"business_action": "reconcile",
		"status":          "queued",
		"trace_id":        manualTraceID,
	})
	if err := emitter.Emit(context.Background(), event.Event{
		Topic:   event.Topic(integrationjobs.SchedulerTriggeredTopic),
		Meta:    manualMeta,
		Payload: manualPayload,
	}); err != nil {
		t.Fatalf("emit manual event failed: %v", err)
	}

	cronTraceID, err := dispatcher.DispatchCronTrigger(context.Background(), "runtime.scheduler.trigger", map[string]any{
		"business_action": "reconcile",
		"status":          "queued",
	})
	if err != nil {
		t.Fatalf("dispatch cron event failed: %v", err)
	}
	if cronTraceID == "" {
		t.Fatal("cron trace id should not be empty")
	}

	events := emitter.Drain()
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	for i, ev := range events {
		if string(ev.Topic) != integrationjobs.SchedulerTriggeredTopic {
			t.Fatalf("event[%d] topic mismatch: %s", i, ev.Topic)
		}
		var payload map[string]any
		if err := json.Unmarshal(ev.Payload, &payload); err != nil {
			t.Fatalf("event[%d] payload decode failed: %v", i, err)
		}
		if payload["business_action"] != "reconcile" {
			t.Fatalf("event[%d] business_action mismatch: %#v", i, payload["business_action"])
		}
		if payload["status"] != "queued" {
			t.Fatalf("event[%d] status mismatch: %#v", i, payload["status"])
		}
		traceFromPayload, _ := payload["trace_id"].(string)
		if traceFromPayload == "" {
			t.Fatalf("event[%d] trace_id should not be empty", i)
		}
		if ev.Meta.TraceID != traceFromPayload {
			t.Fatalf("event[%d] trace mismatch meta=%s payload=%s", i, ev.Meta.TraceID, traceFromPayload)
		}
	}
}
