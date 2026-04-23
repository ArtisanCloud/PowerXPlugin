package logging

import (
	"context"
	"time"
)

func executeWithRetry(ctx context.Context, sinkName SinkType, retry RetryPolicy, emit func() error) []SinkOutcome {
	attempts := max(1, retry.MaxAttempts)
	outcomes := make([]SinkOutcome, 0, attempts)
	for i := 1; i <= attempts; i++ {
		err := emit()
		if err == nil {
			outcomes = append(outcomes, SinkOutcome{Sink: sinkName, Status: OutcomeSuccess, Attempt: i})
			return outcomes
		}
		if i < attempts && retry.Enabled {
			outcomes = append(outcomes, SinkOutcome{Sink: sinkName, Status: OutcomeRetrying, Attempt: i, ErrorCode: "sink_emit_failed", Error: err.Error()})
			if retry.BackoffMS > 0 {
				wait := time.Duration(retry.BackoffMS) * time.Millisecond
				if !sleepWithContext(ctx, wait) {
					outcomes = append(outcomes, SinkOutcome{Sink: sinkName, Status: OutcomeDropped, Attempt: i + 1, ErrorCode: "retry_interrupted", Error: "retry interrupted by context cancellation"})
					return outcomes
				}
			}
			continue
		}
		outcomes = append(outcomes, SinkOutcome{Sink: sinkName, Status: OutcomeFailed, Attempt: i, ErrorCode: "sink_emit_failed", Error: err.Error()})
		return outcomes
	}
	return outcomes
}

func sleepWithContext(ctx context.Context, wait time.Duration) bool {
	if ctx == nil {
		time.Sleep(wait)
		return true
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
