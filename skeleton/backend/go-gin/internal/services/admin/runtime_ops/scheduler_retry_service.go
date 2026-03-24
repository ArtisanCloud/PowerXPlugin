package runtime_ops

// RetryAction describes next step after a failed dispatch.
type RetryAction string

const (
	RetryActionRetry        RetryAction = "retry"
	RetryActionCreateTicket RetryAction = "create_ticket"
)

// RetryDecision describes retry strategy output.
type RetryDecision struct {
	Action         RetryAction `json:"action"`
	CurrentAttempt int         `json:"current_attempt"`
	MaxAttempts    int         `json:"max_attempts"`
	Exhausted      bool        `json:"exhausted"`
}

// SchedulerRetryService handles bounded retry decisions.
type SchedulerRetryService struct{}

// NewSchedulerRetryService constructs retry service.
func NewSchedulerRetryService() *SchedulerRetryService {
	return &SchedulerRetryService{}
}

// Decide returns retry decision by attempt window.
func (s *SchedulerRetryService) Decide(currentAttempt, maxAttempts int) RetryDecision {
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
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
