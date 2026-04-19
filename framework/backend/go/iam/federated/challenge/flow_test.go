package challenge

import (
	"context"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/contracts"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/risk"
)

func TestChallengeCallbackMainFlow(t *testing.T) {
	mgr := NewManager()
	evaluator := risk.NewEvaluator(time.Minute)
	now := time.Date(2026, 4, 13, 1, 0, 0, 0, time.UTC)

	issued, err := mgr.Issue(context.Background(), contracts.ChallengeIssueRequest{
		TenantUUID: "tenant-a",
		Provider:   "wecom",
		TraceID:    "trace-1",
		TTL:        time.Minute,
		Now:        now,
	})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	consumed, err := mgr.ValidateAndConsume(context.Background(), contracts.ChallengeConsumeRequest{
		State:      issued.State,
		Nonce:      issued.Nonce,
		TenantUUID: "tenant-a",
		Provider:   "wecom",
		Now:        now.Add(10 * time.Second),
	})
	if err != nil {
		t.Fatalf("ValidateAndConsume() error = %v", err)
	}

	decision := evaluator.EvaluateCallback(context.Background(), contracts.RiskEvaluateRequest{
		Challenge:      consumed,
		State:          issued.State,
		Nonce:          issued.Nonce,
		Code:           "auth-code-1",
		TenantUUID:     "tenant-a",
		SignatureValid: true,
		Now:            now.Add(12 * time.Second),
	})
	if !decision.Allowed {
		t.Fatalf("decision = %+v, want allowed", decision)
	}
}
