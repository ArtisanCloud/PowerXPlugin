package risk

import (
	"context"
	"fmt"
	"sort"
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

func TestEvaluateCallbackP95UnderBudget(t *testing.T) {
	evaluator := NewEvaluator(time.Minute)
	now := time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC)
	base := contracts.RiskEvaluateRequest{
		Challenge:      contracts.LoginChallenge{State: "s", Nonce: "n", TenantUUID: "tenant-a", ExpiresAt: now.Add(time.Minute)},
		State:          "s",
		Nonce:          "n",
		TenantUUID:     "tenant-a",
		SignatureValid: true,
	}
	const samples = 1000
	latency := make([]time.Duration, 0, samples)
	for i := 0; i < samples; i++ {
		req := base
		req.Code = fmt.Sprintf("code-%d", i)
		req.Now = now.Add(time.Duration(i) * time.Microsecond)
		start := time.Now()
		decision := evaluator.EvaluateCallback(context.Background(), req)
		if !decision.Allowed {
			t.Fatalf("decision[%d]=%+v, want allow", i, decision)
		}
		latency = append(latency, time.Since(start))
	}
	sort.Slice(latency, func(i, j int) bool { return latency[i] < latency[j] })
	p95 := latency[int(float64(len(latency))*0.95)-1]
	t.Logf("risk evaluator p95 latency = %s", p95)
	if p95 >= 200*time.Millisecond {
		t.Fatalf("p95=%s, want < 200ms", p95)
	}
}
