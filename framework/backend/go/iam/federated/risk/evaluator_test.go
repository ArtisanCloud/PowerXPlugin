package risk

import (
	"context"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/contracts"
)

func TestEvaluateCallbackRejectsSignatureInvalid(t *testing.T) {
	evaluator := NewEvaluator(time.Minute)
	now := time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC)
	decision := evaluator.EvaluateCallback(context.Background(), contracts.RiskEvaluateRequest{
		Challenge: contracts.LoginChallenge{State: "s", Nonce: "n", TenantUUID: "t", ExpiresAt: now.Add(time.Minute)},
		State:     "s", Nonce: "n", TenantUUID: "t", SignatureValid: false, Now: now,
	})
	if decision.Allowed || decision.Code != contracts.ErrorCodeRiskSignature {
		t.Fatalf("decision = %+v, want signature reject", decision)
	}
}

func TestEvaluateCallbackRejectsTenantBoundary(t *testing.T) {
	evaluator := NewEvaluator(time.Minute)
	now := time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC)
	decision := evaluator.EvaluateCallback(context.Background(), contracts.RiskEvaluateRequest{
		Challenge:      contracts.LoginChallenge{State: "s", Nonce: "n", TenantUUID: "tenant-a", ExpiresAt: now.Add(time.Minute)},
		State:          "s",
		Nonce:          "n",
		TenantUUID:     "tenant-b",
		SignatureValid: true,
		Now:            now,
	})
	if decision.Allowed || decision.Code != contracts.ErrorCodeRiskTenantBoundary {
		t.Fatalf("decision = %+v, want tenant boundary reject", decision)
	}
}

func TestEvaluateCallbackRejectsExpiredChallenge(t *testing.T) {
	evaluator := NewEvaluator(time.Minute)
	now := time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC)
	decision := evaluator.EvaluateCallback(context.Background(), contracts.RiskEvaluateRequest{
		Challenge:      contracts.LoginChallenge{State: "s", Nonce: "n", TenantUUID: "tenant-a", ExpiresAt: now.Add(-time.Second)},
		State:          "s",
		Nonce:          "n",
		TenantUUID:     "tenant-a",
		SignatureValid: true,
		Now:            now,
	})
	if decision.Allowed || decision.Code != contracts.ErrorCodeChallengeExpired {
		t.Fatalf("decision = %+v, want expired reject", decision)
	}
}

func TestEvaluateCallbackRejectsReplayCode(t *testing.T) {
	evaluator := NewEvaluator(time.Minute)
	now := time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC)
	req := contracts.RiskEvaluateRequest{
		Challenge:      contracts.LoginChallenge{State: "s", Nonce: "n", TenantUUID: "tenant-a", ExpiresAt: now.Add(time.Minute)},
		State:          "s",
		Nonce:          "n",
		TenantUUID:     "tenant-a",
		SignatureValid: true,
		Code:           "code-replay",
		Now:            now,
	}
	first := evaluator.EvaluateCallback(context.Background(), req)
	if !first.Allowed {
		t.Fatalf("first decision = %+v, want allow", first)
	}
	second := evaluator.EvaluateCallback(context.Background(), req)
	if second.Allowed || second.Code != contracts.ErrorCodeRiskReplay {
		t.Fatalf("second decision = %+v, want replay reject", second)
	}
}
