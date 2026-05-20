package scheduler

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/eventbridge"
)

func TestLocalProviderCreateTriggerAndList(t *testing.T) {
	emitter := eventbridge.NewLocalEmitter(4)
	provider := NewLocalProvider(LocalProviderConfig{
		Emitter:        emitter,
		SourcePlugin:   "com.powerx.plugins.ai-craft",
		PayloadVersion: "v1",
		Now: func() time.Time {
			return time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
		},
	})

	job, err := provider.CreateJob(context.Background(), JobSpec{
		TenantUUID:     "tenant-001",
		OwnerType:      OwnerTypePlugin,
		OwnerID:        "com.powerx.plugins.ai-craft",
		Name:           "sample_progress_50",
		ScheduleType:   ScheduleTypeOnce,
		ScheduleExpr:   "2026-05-15T10:30:00Z",
		IdempotencyKey: "order-001:sample_progress_50",
		Payload: map[string]any{
			"business_action": "sample_progress_50",
			"order_id":        "order-001",
			"trace_id":        "trace-001",
		},
	})
	if err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}
	if job.JobID == "" {
		t.Fatal("CreateJob() should assign job id")
	}
	if job.Topic != TriggeredTopic {
		t.Fatalf("topic = %q, want %q", job.Topic, TriggeredTopic)
	}
	jobs, err := provider.ListJobs(context.Background(), ListJobsInput{TenantUUID: "tenant-001"})
	if err != nil {
		t.Fatalf("ListJobs() error = %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("ListJobs() len = %d, want 1", len(jobs))
	}
	if err := provider.EmitDueTrigger(context.Background(), job.JobID, "tenant-001", ScheduleTypeOnce); err != nil {
		t.Fatalf("EmitDueTrigger() error = %v", err)
	}
	after, err := provider.GetJob(context.Background(), job.JobID, "tenant-001")
	if err != nil {
		t.Fatalf("GetJob() after trigger error = %v", err)
	}
	if after.Status != StatusCompleted {
		t.Fatalf("status after once trigger = %q, want %q", after.Status, StatusCompleted)
	}
	if after.LastRunAt.IsZero() {
		t.Fatal("LastRunAt should be set after trigger")
	}
	events := emitter.Drain()
	if len(events) != 1 {
		t.Fatalf("events len = %d, want 1", len(events))
	}
	if string(events[0].Topic) != TriggeredTopic {
		t.Fatalf("event topic = %q, want %q", events[0].Topic, TriggeredTopic)
	}
	if events[0].Meta.TraceID != "trace-001" {
		t.Fatalf("trace id = %q, want trace-001", events[0].Meta.TraceID)
	}
	var payload map[string]any
	if err := json.Unmarshal(events[0].Payload, &payload); err != nil {
		t.Fatalf("payload decode: %v", err)
	}
	if payload["business_action"] != "sample_progress_50" {
		t.Fatalf("business_action = %#v", payload["business_action"])
	}
	if err := provider.TriggerJob(context.Background(), job.JobID, "tenant-001"); err != nil {
		t.Fatalf("TriggerJob() completed once error = %v", err)
	}
	if events := emitter.Drain(); len(events) != 0 {
		t.Fatalf("completed once should not emit again, got %d events", len(events))
	}
}

func TestLocalProviderTriggerJobPreviewDoesNotCompleteOnceJob(t *testing.T) {
	emitter := eventbridge.NewLocalEmitter(4)
	provider := NewLocalProvider(LocalProviderConfig{
		Emitter:        emitter,
		SourcePlugin:   "com.powerx.plugins.ai-craft",
		PayloadVersion: "v1",
		Now: func() time.Time {
			return time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
		},
	})

	job, err := provider.CreateJob(context.Background(), JobSpec{
		TenantUUID:   "tenant-001",
		OwnerType:    OwnerTypePlugin,
		OwnerID:      "com.powerx.plugins.ai-craft",
		Name:         "sample_preview",
		ScheduleType: ScheduleTypeOnce,
		ScheduleExpr: "2026-05-15T10:30:00Z",
		Payload: map[string]any{
			"trace_id": "trace-preview",
		},
	})
	if err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}
	if err := provider.TriggerJobPreview(context.Background(), job.JobID, "tenant-001"); err != nil {
		t.Fatalf("TriggerJobPreview() error = %v", err)
	}
	afterPreview, err := provider.GetJob(context.Background(), job.JobID, "tenant-001")
	if err != nil {
		t.Fatalf("GetJob() after preview error = %v", err)
	}
	if afterPreview.Status != StatusActive {
		t.Fatalf("status after preview = %q, want %q", afterPreview.Status, StatusActive)
	}
	if afterPreview.LastRunAt.IsZero() {
		t.Fatal("LastRunAt should be set after preview trigger")
	}
	if err := provider.EmitDueTrigger(context.Background(), job.JobID, "tenant-001", ScheduleTypeOnce); err != nil {
		t.Fatalf("EmitDueTrigger() error = %v", err)
	}
	afterDue, err := provider.GetJob(context.Background(), job.JobID, "tenant-001")
	if err != nil {
		t.Fatalf("GetJob() after due error = %v", err)
	}
	if afterDue.Status != StatusCompleted {
		t.Fatalf("status after due trigger = %q, want %q", afterDue.Status, StatusCompleted)
	}
	if events := emitter.Drain(); len(events) != 2 {
		t.Fatalf("preview and due trigger should both emit, got %d events", len(events))
	}
}
