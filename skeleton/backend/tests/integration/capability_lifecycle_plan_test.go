package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	lifecycle "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/capability_lifecycle"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
)

func TestCapabilityLifecyclePlanFlow(t *testing.T) {
	t.Setenv("POWERXPLUGIN_LIFECYCLE_STORAGE", filepath.Join(t.TempDir(), "plans.json"))

	svc := lifecycle.NewPlanService(&app.Deps{})
	ctx := context.Background()

	plan, err := svc.CreatePlan(ctx, &lifecycle.PlanInput{
		CapabilityID:         "com.powerx.demo.template.create",
		ChangeType:           "upgrade",
		DiffSummary:          "新增字段 template_title，旧版调用需在灰度期内迁移。",
		NotificationChannels: []string{"email", "webhook"},
		GracePeriodHours:     48,
		DualRunUntil:         time.Now().UTC().Add(72 * time.Hour).Format(time.RFC3339),
		RollbackPlan:         "如 SLA 降级立即切回 v1 endpoint。",
		Windows: []lifecycle.RolloutWindow{
			{
				Label:   "wave-1",
				StartAt: time.Now().UTC().Format(time.RFC3339),
				EndAt:   time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339),
				Percent: 20,
			},
			{
				Label:   "wave-2",
				StartAt: time.Now().UTC().Add(4 * time.Hour).Format(time.RFC3339),
				EndAt:   time.Now().UTC().Add(8 * time.Hour).Format(time.RFC3339),
				Percent: 80,
			},
		},
		Metadata: map[string]string{
			"impact_scope": "demo-tenant",
		},
		Actor: "ops.lead@powerx.dev",
	})
	if err != nil {
		t.Fatalf("CreatePlan returned error: %v", err)
	}
	if plan.Status != "pending" {
		t.Fatalf("expected status pending, got %s", plan.Status)
	}
	if plan.ID == "" {
		t.Fatalf("plan ID should be generated")
	}

	list := svc.ListPlans(plan.CapabilityID)
	if len(list) != 1 {
		t.Fatalf("expected one plan, got %d", len(list))
	}

	updated, err := svc.UpdateStatus(ctx, &lifecycle.StatusInput{
		PlanID: plan.ID,
		Status: "approved",
		Actor:  "ops.lead@powerx.dev",
		Notes:  "风险已确认",
	})
	if err != nil {
		t.Fatalf("UpdateStatus returned error: %v", err)
	}
	if updated.Status != "approved" {
		t.Fatalf("expected status approved, got %s", updated.Status)
	}

	storage := os.Getenv("POWERXPLUGIN_LIFECYCLE_STORAGE")
	if storage == "" {
		t.Fatalf("storage env not set")
	}
	if _, err := os.Stat(storage); err != nil {
		t.Fatalf("expected lifecycle storage file to exist: %v", err)
	}
}
