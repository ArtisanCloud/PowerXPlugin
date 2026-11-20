package runtime_ops

import (
	"errors"
	"strings"
	"sync"
	"time"
)

// RetryAction describes next step after a failed dispatch.
type RetryAction string

const (
	RetryActionRetry        RetryAction = "retry"
	RetryActionCreateTicket RetryAction = "create_ticket"
)

const (
	DispatchStatusQueued  = "queued"
	DispatchStatusFailed  = "failed"
	DispatchStatusPaused  = "paused"
	DispatchStatusRunning = "processing"
)

var (
	ErrDispatchIDRequired = errors.New("dispatch_id is required")
	ErrDispatchNotFound   = errors.New("dispatch not found")
	ErrRetryNotExhausted  = errors.New("retry attempts not exhausted")
)

// RetryDecision describes retry strategy output.
type RetryDecision struct {
	Action         RetryAction `json:"action"`
	CurrentAttempt int         `json:"current_attempt"`
	MaxAttempts    int         `json:"max_attempts"`
	Exhausted      bool        `json:"exhausted"`
}

// DispatchRetryState tracks retry/pause state for a dispatch.
type DispatchRetryState struct {
	DispatchID       string    `json:"dispatch_id"`
	Status           string    `json:"status"`
	CurrentAttempt   int       `json:"current_attempt"`
	MaxAttempts      int       `json:"max_attempts"`
	LastErrorCode    string    `json:"last_error_code,omitempty"`
	LastErrorMessage string    `json:"last_error_message,omitempty"`
	Exhausted        bool      `json:"exhausted"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// SchedulerRetryService handles bounded retry decisions.
type SchedulerRetryService struct {
	mu     sync.RWMutex
	states map[string]*DispatchRetryState
}

// NewSchedulerRetryService constructs retry service.
func NewSchedulerRetryService() *SchedulerRetryService {
	return &SchedulerRetryService{
		states: make(map[string]*DispatchRetryState),
	}
}

// Decide returns retry decision by attempt window.
func (s *SchedulerRetryService) Decide(currentAttempt, maxAttempts int) RetryDecision {
	maxAttempts = normalizeMaxAttempts(maxAttempts)
	if currentAttempt < maxAttempts {
		return RetryDecision{
			Action:         RetryActionRetry,
			CurrentAttempt: currentAttempt,
			MaxAttempts:    maxAttempts,
			Exhausted:      false,
		}
	}
	return RetryDecision{
		Action:         RetryActionCreateTicket,
		CurrentAttempt: currentAttempt,
		MaxAttempts:    maxAttempts,
		Exhausted:      true,
	}
}

// Retry advances one retry attempt and returns bounded decision.
func (s *SchedulerRetryService) Retry(dispatchID string, maxAttempts int, lastErrorCode, lastErrorMessage string) (RetryDecision, error) {
	dispatchID = strings.TrimSpace(dispatchID)
	if dispatchID == "" {
		return RetryDecision{}, ErrDispatchIDRequired
	}
	maxAttempts = normalizeMaxAttempts(maxAttempts)

	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.states[dispatchID]
	if !ok {
		state = &DispatchRetryState{
			DispatchID:     dispatchID,
			MaxAttempts:    maxAttempts,
			Status:         DispatchStatusFailed,
			CurrentAttempt: 0,
		}
		s.states[dispatchID] = state
	}
	state.MaxAttempts = maxAttempts
	state.CurrentAttempt++
	state.LastErrorCode = strings.TrimSpace(lastErrorCode)
	state.LastErrorMessage = strings.TrimSpace(lastErrorMessage)

	decision := s.Decide(state.CurrentAttempt, state.MaxAttempts)
	state.Exhausted = decision.Exhausted
	state.UpdatedAt = time.Now().UTC()
	if state.Exhausted {
		state.Status = DispatchStatusFailed
	} else {
		state.Status = DispatchStatusRunning
	}
	return decision, nil
}

// Pause marks an exhausted dispatch as paused.
func (s *SchedulerRetryService) Pause(dispatchID string) error {
	dispatchID = strings.TrimSpace(dispatchID)
	if dispatchID == "" {
		return ErrDispatchIDRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.states[dispatchID]
	if !ok {
		return ErrDispatchNotFound
	}
	if !state.Exhausted {
		return ErrRetryNotExhausted
	}
	state.Status = DispatchStatusPaused
	state.UpdatedAt = time.Now().UTC()
	return nil
}

// Resume clears retry exhaustion and unpauses a dispatch.
func (s *SchedulerRetryService) Resume(dispatchID string) error {
	dispatchID = strings.TrimSpace(dispatchID)
	if dispatchID == "" {
		return ErrDispatchIDRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.states[dispatchID]
	if !ok {
		return ErrDispatchNotFound
	}
	state.CurrentAttempt = 0
	state.Exhausted = false
	state.LastErrorCode = ""
	state.LastErrorMessage = ""
	state.Status = DispatchStatusQueued
	state.UpdatedAt = time.Now().UTC()
	return nil
}

// GetState returns a snapshot of dispatch retry state.
func (s *SchedulerRetryService) GetState(dispatchID string) (DispatchRetryState, bool) {
	dispatchID = strings.TrimSpace(dispatchID)
	if dispatchID == "" {
		return DispatchRetryState{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.states[dispatchID]
	if !ok || state == nil {
		return DispatchRetryState{}, false
	}
	cp := *state
	return cp, true
}

func normalizeMaxAttempts(maxAttempts int) int {
	if maxAttempts <= 0 {
		return 3
	}
	return maxAttempts
}
