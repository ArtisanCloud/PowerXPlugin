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
