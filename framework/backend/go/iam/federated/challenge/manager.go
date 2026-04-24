package challenge

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/contracts"
	"github.com/google/uuid"
)

const defaultChallengeTTL = 5 * time.Minute

// Manager 是内存 challenge 管理器。
type Manager struct {
	mu         sync.Mutex
	challenges map[string]contracts.LoginChallenge
}

// NewManager 创建 challenge 管理器。
func NewManager() *Manager {
	return &Manager{challenges: make(map[string]contracts.LoginChallenge)}
}

// Issue 签发 challenge。
func (m *Manager) Issue(_ context.Context, req contracts.ChallengeIssueRequest) (contracts.LoginChallenge, error) {
	now := req.Now
	if now.IsZero() {
		now = time.Now()
	}
	ttl := req.TTL
	if ttl <= 0 {
		ttl = defaultChallengeTTL
	}

	challenge := contracts.LoginChallenge{
		State:      uuid.NewString(),
		Nonce:      uuid.NewString(),
		TenantUUID: strings.TrimSpace(req.TenantUUID),
		Provider:   strings.TrimSpace(req.Provider),
		TraceID:    strings.TrimSpace(req.TraceID),
		IssuedAt:   now,
		ExpiresAt:  now.Add(ttl),
	}

	m.mu.Lock()
	m.challenges[challenge.State] = challenge
	m.mu.Unlock()
	return challenge, nil
}

// ValidateAndConsume 校验并消费 challenge。
func (m *Manager) ValidateAndConsume(_ context.Context, req contracts.ChallengeConsumeRequest) (contracts.LoginChallenge, error) {
	now := req.Now
	if now.IsZero() {
		now = time.Now()
	}
	state := strings.TrimSpace(req.State)
	if state == "" {
		return contracts.LoginChallenge{}, contracts.NewError(contracts.ErrorCodeInvalidChallenge, "state is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	challenge, ok := m.challenges[state]
	if !ok {
		return contracts.LoginChallenge{}, contracts.NewError(contracts.ErrorCodeInvalidChallenge, "challenge not found")
	}
	if challenge.ConsumedAt != nil {
		return contracts.LoginChallenge{}, contracts.NewError(contracts.ErrorCodeChallengeReplay, "challenge already consumed")
	}
	if now.After(challenge.ExpiresAt) {
		return contracts.LoginChallenge{}, contracts.NewError(contracts.ErrorCodeChallengeExpired, "challenge expired")
	}
	if nonce := strings.TrimSpace(req.Nonce); nonce != "" && nonce != challenge.Nonce {
		return contracts.LoginChallenge{}, contracts.NewError(contracts.ErrorCodeInvalidChallenge, "nonce mismatch")
	}
	if tenantUUID := strings.TrimSpace(req.TenantUUID); tenantUUID != "" && tenantUUID != challenge.TenantUUID {
		return contracts.LoginChallenge{}, contracts.NewError(contracts.ErrorCodeChallengeTenantMiss, "tenant mismatch")
	}
	if provider := strings.TrimSpace(req.Provider); provider != "" && provider != challenge.Provider {
		return contracts.LoginChallenge{}, contracts.NewError(contracts.ErrorCodeInvalidChallenge, "provider mismatch")
	}

	challenge.ConsumedAt = &now
	m.challenges[state] = challenge
	return challenge, nil
}
