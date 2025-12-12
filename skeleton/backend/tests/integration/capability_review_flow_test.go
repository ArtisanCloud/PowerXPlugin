package integration_test

import (
	"context"
	"testing"
	"time"

	capsvc "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/capability"
	reviewsvc "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/capability_review"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
)

func TestCapabilityReviewWorkflow(t *testing.T) {
	svc := reviewsvc.NewWorkflowService(&app.Deps{})
	fixed := time.Date(2025, 12, 5, 8, 0, 0, 0, time.UTC)
	svc.SetClock(func() time.Time { return fixed })

	record := &capsvc.CapabilityRecord{
		ID:          "com.powerx.demo.template.create",
		Sensitivity: "high",
		TenantScope: "global",
		Owner:       capsvc.ContactInfo{Email: "owner@example.com"},
		Scenario:    "用于合同模版生成",
		Tags:        []string{"workflow"},
	}

	tasks, err := svc.EnsureTasks(context.Background(), record)
	if err != nil {
		t.Fatalf("EnsureTasks returned error: %v", err)
	}
	if len(tasks) < 2 {
		t.Fatalf("expected at least two review tasks for high sensitivity, got %d", len(tasks))
	}

	first := tasks[0]
	if _, err := svc.AddComment(context.Background(), first.ID, reviewsvc.CommentInput{
		Author:  "security.lead",
		Message: "请补充数据脱敏说明",
	}); err != nil {
		t.Fatalf("AddComment failed: %v", err)
	}

	if _, err := svc.Resolve(context.Background(), first.ID, reviewsvc.DecisionInput{
		Actor:    "security.lead",
		Decision: "request_changes",
		Note:     "缺少字段加密策略",
	}); err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	reopened, err := svc.Resubmit(context.Background(), record.ID, reviewsvc.ResubmitInput{
		Actor: "dev.lead",
		Note:  "已提交整改文档",
	})
	if err != nil {
		t.Fatalf("Resubmit failed: %v", err)
	}
	if len(reopened) != len(tasks) {
		t.Fatalf("expected %d tasks after resubmit, got %d", len(tasks), len(reopened))
	}
	for _, task := range reopened {
		if task.Status != reviewsvc.StatusPending {
			t.Fatalf("expected task %s reset to pending, got %s", task.ID, task.Status)
		}
		if task.ReworkCount == 0 {
			t.Fatalf("expected rework count incremented")
		}
	}

	// Fast-forward to trigger SLA escalation.
	later := fixed.Add(47 * time.Hour)
	escalations := svc.EvaluateSLA(context.Background(), later)
	if len(escalations) == 0 {
		t.Fatalf("expected escalation events when SLA window is close")
	}
}
