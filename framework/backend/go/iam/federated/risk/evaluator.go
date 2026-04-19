package risk

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/contracts"
)

const defaultReplayTTL = 10 * time.Minute

// Evaluator 是内存风险判定器。
type Evaluator struct {
	mu      sync.Mutex
	replays map[string]time.Time
	ttl     time.Duration
}

// NewEvaluator 创建风险判定器。
func NewEvaluator(ttl time.Duration) *Evaluator {
	if ttl <= 0 {
		ttl = defaultReplayTTL
	}
	return &Evaluator{replays: make(map[string]time.Time), ttl: ttl}
}

// EvaluateCallback 判定回调风险。
func (e *Evaluator) EvaluateCallback(_ context.Context, req contracts.RiskEvaluateRequest) contracts.RiskDecision {
	now := req.Now
	if now.IsZero() {
		now = time.Now()
	}

	if !req.SignatureValid {
		return reject(contracts.ErrorCodeRiskSignature, "signature invalid", map[string]string{"rule": "signature"})
	}
	if strings.TrimSpace(req.State) == "" || strings.TrimSpace(req.Nonce) == "" {
		return reject(contracts.ErrorCodeRiskStateNonce, "missing state or nonce", map[string]string{"rule": "state_nonce"})
	}
	if req.State != req.Challenge.State || req.Nonce != req.Challenge.Nonce {
		return reject(contracts.ErrorCodeRiskStateNonce, "state or nonce mismatch", map[string]string{"rule": "state_nonce"})
	}
	if now.After(req.Challenge.ExpiresAt) {
		return reject(contracts.ErrorCodeChallengeExpired, "challenge expired", map[string]string{"rule": "challenge_ttl"})
	}
	if tenant := strings.TrimSpace(req.TenantUUID); tenant != "" && tenant != req.Challenge.TenantUUID {
		return reject(contracts.ErrorCodeRiskTenantBoundary, "tenant boundary violated", map[string]string{"rule": "tenant_boundary"})
	}

	if decision := e.markReplay(strings.TrimSpace(req.Code), now); !decision.Allowed {
		return decision
	}
	return contracts.RiskDecision{Allowed: true}
}

func (e *Evaluator) markReplay(code string, now time.Time) contracts.RiskDecision {
	if code == "" {
		return contracts.RiskDecision{Allowed: true}
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	for k, exp := range e.replays {
		if now.After(exp) {
			delete(e.replays, k)
		}
	}
	if exp, exists := e.replays[code]; exists && now.Before(exp) {
		return reject(contracts.ErrorCodeRiskReplay, "authorization code replayed", map[string]string{"rule": "replay"})
	}
	e.replays[code] = now.Add(e.ttl)
	return contracts.RiskDecision{Allowed: true}
}

func reject(code contracts.ErrorCode, reason string, evidence map[string]string) contracts.RiskDecision {
	return contracts.RiskDecision{Allowed: false, Code: code, Reason: reason, Evidence: evidence}
}
