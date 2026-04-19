package providers

import (
	"context"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/challenge"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/contracts"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/risk"
)

type fakeProvider struct{ key string }

func (f fakeProvider) Key() string { return f.key }
func (f fakeProvider) BuildAuthorizeURL(context.Context, contracts.AuthorizeRequest) (contracts.AuthorizeResponse, error) {
	return contracts.AuthorizeResponse{AuthorizeURL: "https://example.test"}, nil
}
func (f fakeProvider) ExchangeCode(context.Context, contracts.ExchangeCodeRequest) (contracts.ProviderToken, error) {
	return contracts.ProviderToken{AccessToken: "token"}, nil
}
func (f fakeProvider) ResolveIdentity(context.Context, contracts.ResolveIdentityRequest) (contracts.ExternalIdentity, error) {
	return contracts.ExternalIdentity{Provider: f.key, ExternalUserID: "u-1"}, nil
}

func TestRegistryRegisterAndGet(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(fakeProvider{key: "WeCom"}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if err := registry.Register(fakeProvider{key: "wecom"}); !contracts.HasCode(err, contracts.ErrorCodeProviderRegistered) {
		t.Fatalf("Register duplicate err = %v, want %s", err, contracts.ErrorCodeProviderRegistered)
	}
	provider, ok := registry.Get("WECOM")
	if !ok || provider.Key() != "WeCom" {
		t.Fatalf("Get() = (%v, %v), want provider exists", provider, ok)
	}
	if _, err := registry.MustGet("missing"); !contracts.HasCode(err, contracts.ErrorCodeProviderNotFound) {
		t.Fatalf("MustGet missing err = %v, want %s", err, contracts.ErrorCodeProviderNotFound)
	}
}

func TestChallengeManagerIssueAndConsume(t *testing.T) {
	mgr := challenge.NewManager()
	now := time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC)
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
	if consumed.ConsumedAt == nil {
		t.Fatalf("ConsumedAt = nil, want non-nil")
	}

	_, err = mgr.ValidateAndConsume(context.Background(), contracts.ChallengeConsumeRequest{State: issued.State, Now: now.Add(20 * time.Second)})
	if !contracts.HasCode(err, contracts.ErrorCodeChallengeReplay) {
		t.Fatalf("reconsume err = %v, want %s", err, contracts.ErrorCodeChallengeReplay)
	}
}

func TestRiskEvaluatorBlocksReplay(t *testing.T) {
	evaluator := risk.NewEvaluator(time.Minute)
	now := time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC)
	challengeObj := contracts.LoginChallenge{
		State:      "state-1",
		Nonce:      "nonce-1",
		TenantUUID: "tenant-a",
		ExpiresAt:  now.Add(time.Minute),
	}
	first := evaluator.EvaluateCallback(context.Background(), contracts.RiskEvaluateRequest{
		Challenge:      challengeObj,
		State:          "state-1",
		Nonce:          "nonce-1",
		Code:           "code-1",
		TenantUUID:     "tenant-a",
		SignatureValid: true,
		Now:            now,
	})
	if !first.Allowed {
		t.Fatalf("first decision = %+v, want allowed", first)
	}
	second := evaluator.EvaluateCallback(context.Background(), contracts.RiskEvaluateRequest{
		Challenge:      challengeObj,
		State:          "state-1",
		Nonce:          "nonce-1",
		Code:           "code-1",
		TenantUUID:     "tenant-a",
		SignatureValid: true,
		Now:            now.Add(2 * time.Second),
	})
	if second.Allowed || second.Code != contracts.ErrorCodeRiskReplay {
		t.Fatalf("second decision = %+v, want replay reject", second)
	}
}
